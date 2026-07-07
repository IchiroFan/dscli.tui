// Package jsonline implements a streaming JSON-line encoder/decoder.
//
// JSON-line format: each message is a single JSON object terminated by '\n'.
// This is the wire format between dscli.tui and dscli.
//
// This package will eventually move to the dscli project — it is the shared
// codec that both sides use.  For now it lives here, co-located with the
// TUI that is its first real consumer.
//
// Usage (encoding):
//
//	enc := jsonline.NewEncoder(w)
//	enc.Encode(msg)  // writes JSON + '\n'
//
// Usage (decoding):
//
//	dec := jsonline.NewDecoder(r)
//	for dec.Decode() {
//	    msg := dec.Message()
//	    // use msg
//	}
//	if err := dec.Err(); err != nil { ... }
package jsonline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── Encoder ────────────────────────────────────────────────

// Encoder writes protocol.Message values as JSON lines.
type Encoder struct {
	w io.Writer
}

// NewEncoder creates an Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes msg as a single JSON line.
func (enc *Encoder) Encode(msg *protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("jsonline encode: %w", err)
	}
	data = append(data, '\n')
	_, err = enc.w.Write(data)
	return err
}

// ─── Decoder ────────────────────────────────────────────────

// Decoder reads protocol.Message values from a JSON-line stream.
type Decoder struct {
	scanner *bufio.Scanner
	msg     *protocol.Message
	lastErr error
}

// NewDecoder creates a Decoder reading from r.
// The buffer size is increased to 1 MiB to handle large payloads.
func NewDecoder(r io.Reader) *Decoder {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 65536), 1024*1024) // 1 MiB max line
	return &Decoder{scanner: scanner}
}

// Decode reads the next JSON line and decodes it into a Message.
// Returns true on success, false on end-of-stream or error.
// Call Message() to retrieve the decoded value.
func (d *Decoder) Decode() bool {
	if !d.scanner.Scan() {
		d.lastErr = d.scanner.Err()
		return false
	}

	line := d.scanner.Bytes()
	if len(line) == 0 {
		return d.Decode() // skip empty lines
	}

	var msg protocol.Message
	if err := json.Unmarshal(line, &msg); err != nil {
		d.lastErr = fmt.Errorf("jsonline decode: %w\nraw: %s", err, string(line))
		return false
	}
	d.msg = &msg
	return true
}

// Message returns the last successfully decoded message.
func (d *Decoder) Message() *protocol.Message { return d.msg }

// Err returns any error encountered during decoding.
func (d *Decoder) Err() error {
	if d.lastErr == io.EOF {
		return nil
	}
	return d.lastErr
}
