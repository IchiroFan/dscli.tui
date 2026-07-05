package aiagent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// resolveDSCLIPath locates the dscli executable.
//
// Priority (first match wins):
//  1. hint parameter (explicit path from config / ChatSessionOptions)
//  2. $DSCLI_PATH environment variable
//  3. "dscli" in $PATH
//  4. "./dscli" (current directory)
//
// Each candidate is validated by running "dscli version" and checking
// that the output looks like dscli (contains "dscli" or Chinese characters).
//
// Returns the resolved path, or an error if none is found.
func resolveDSCLIPath(hint string) (string, error) {
	type candidate struct {
		path  string
		label string
	}

	var candidates []candidate

	if hint != "" {
		candidates = append(candidates, candidate{path: hint, label: "config"})
	}
	if env := os.Getenv("DSCLI_PATH"); env != "" {
		candidates = append(candidates, candidate{path: env, label: "$DSCLI_PATH"})
	}
	candidates = append(candidates, candidate{path: "dscli", label: "PATH"})
	candidates = append(candidates, candidate{path: "./dscli", label: "cwd"})

	for _, c := range candidates {
		cmd := exec.Command(c.path, "version")
		out, err := cmd.CombinedOutput()
		if err != nil {
			continue // not found or not executable
		}
		output := strings.TrimSpace(string(out))
		if strings.Contains(output, "dscli") || containsChinese(output) {
			return c.path, nil
		}
	}

	return "", fmt.Errorf("dscli not found: checked hint=%q, $DSCLI_PATH, PATH, cwd", hint)
}

// containsChinese is a simple heuristic for detecting dscli output.
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
