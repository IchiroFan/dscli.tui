# dscli.tui — Phase B: Model Refactoring

## Status: ✅ Complete

### Changes Summary

| File | Change |
|------|--------|
| `go.mod` | Add `bubbles v1.0.0` dependency |
| `internal/tui/model.go` | `AppState` → `Screen`, `state` → `screen`, export `Width`/`Height`, replace `chatInput []rune` / `askInput []rune` with `textinput.Model`, add `spinner.Model`, add `chatScroll`/`chatScrollMax`, `askPrevState` → `prevScreen` |
| `internal/tui/styles.go` | Add `SpinnerStyle` / `SpinnerDoneStyle` |
| `internal/tui/update.go` | Update all state/screen refs, use `textinput.Model` API, add `spinner.TickMsg` handler, add PgUp/PgDn scroll support |
| `internal/tui/view.go` | Update all refs, use `textinput.View()`, add spinner animation, add scroll indicators, use lipgloss styles |

### Key Decisions

1. **`textinput.Model`** replaces manual `[]rune` + cursor management — handles cursor movement, blinking, paste, and selection automatically
2. **`spinner.Model`** runs always in the background via `Init()` → harmless when `spinnerOn == false`
3. **`chatScroll`** tracks user-scrolled offset from bottom (0 = auto-scroll follow)
4. **Exported `Width`/`Height`** enables styles to reference terminal dimensions
5. **`prevScreen`** generalizes `askPrevState` — stores any previous screen

### Next: Phase C — Test Suite

Add unit tests for Model, Update, View.
