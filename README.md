# dscli.tui — Terminal UI for dscli

```text
               +-------------------+
     o        | dscli.tui         |
    /|\       | Terminal UI       |
     |   +----+ AI-powered CLI    |
    / \  |    +-------------------+
  ~~~~~~~|~~~~
 dscli   |  TUI frontend
```

## 🎯 What is dscli.tui?

**dscli.tui** is a terminal UI frontend for [dscli](https://github.com/dscli/dscli) — the AI-powered developer CLI. It wraps dscli commands in an interactive Bubble Tea interface, providing a visual, keyboard-driven experience over the same powerful backend.

Think of it as **dscli with training wheels off** — all the power of the CLI, presented in a navigable, scrollable, and chat-friendly TUI.

### Why separate from dscli?

| Project | Role |
|---------|------|
| **dscli** | Pure CLI backend — AI chat, tool execution, session management |
| **dscli.tui** | Terminal UI layer — menus, lists, chat bubbles, modal dialogs |

Separation by a clean `AIAgent` interface means each can evolve independently. The TUI never imports dscli internals — all communication goes through the interface.

---

## ✨ Features

### 🖥️ Main Menu — Command Dashboard

A clean command palette with 13 entries covering the full dscli feature set:

| Menu Item | Description |
|-----------|-------------|
| 💬 Chat | Interactive AI chat session |
| 📊 Balance | Check dscli account balance |
| 🤖 Models | List available AI models |
| ℹ️  Version | Show dscli version information |
| 🔍 Flycheck | Run static analysis on a file or project |
| 📝 History | Browse and inspect conversation history |
| 🛠  Skill | Browse installed skills with detail view |
| 💾 Memory | Browse and search persistent memories |
| 📁 Project | Manage dscli projects with delete confirmation |
| 👤 Role | View AI role configurations |
| 🧰 Tool | Browse available tools with pagination |
| ✉️  Mail | Send and receive AI mail |
| 🔧 Service | Manage dscli services |
| 🚪 Quit | Exit dscli.tui |

### 💬 Chat — Interactive AI Conversation

Full-screen chat interface with:

- **Streaming responses** — AI output appears incrementally as chunks arrive
- **Reasoning display** — Thinking/reasoning content shown in distinct mauve bubbles
- **Multi-line input** — `Ctrl+J` for newline, `Enter` to send
- **Interleaved chat (插入对话)** — Type and send messages while the AI is still responding; uses dscli's climein mechanism
- **Scrollable history** — `PgUp/PgDn` or `Ctrl+↑/↓` to review earlier messages
- **Bubble rendering** — User messages right-aligned in green, assistant in blue, reasoning in mauve

### 📝 History List — Conversation Browser

Navigate paginated conversation history with:

- **Pagination** — 20 items per page, keyboard-driven page flips
- **Column display** — ID, Role (with icons), reasoning preview, content preview, timestamp
- **Detail view** — Press `Enter` on any item to see full message content
- **Vim-style navigation** — `j`/`k`, `PgUp`/`PgDn`, `g`/`G` for top/bottom

### 🛠  Skill List — Browse Skills

Scrollable skill list with:

- Name, scope (global/local/built-in), and auto-inject indicator
- Detail view on `Enter` — shows full skill definition

### 💾 Memory List — Persistent Memory

Browse and search dscli memories:

- **Search** — Press `/` or `s` to enter search mode via AskUser modal
- **Detail view** — `Enter` shows full memory content
- **Timestamps** — Created/Updated dates displayed

### 🧰 Tool List — Tool Browser

Paginated tool catalogue:

- **10 tools per page** with page indicator
- Name, category, and description columns
- `PgUp/PgDn`, `g`/`G` navigation

### 📁 Project List — Project Manager

Manage dscli projects with:

- Current project marker (`→`)
- Maintainer display (AI persona names)
- **Delete with confirmation** — Press `d` to delete, then confirm via AskUser modal

### ❓ AskUser Modal — Interactive Dialogs

Three semantic modes for AI-initiated questions:

- **Confirm** — `y`/`n` for yes/no prompts
- **Choice** — Arrow-key selection from option list
- **Input** — Free-text input with `Enter` to submit

All displayed in a centered rounded-border box.

### 📊 Status Bar

Full-width bottom bar showing at all times:

`dscli v0.8.0  │  📁 ~/go_project/dscli.tui  │  🤖 deepseek-chat  │  💬 Chat`

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────┐
│                 dscli.tui                     │
│  ┌────────────────────────────────────────┐  │
│  │   Bubble Tea TUI (charmbracelet)       │  │
│  │   ┌──────────┐  ┌──────────────────┐   │  │
│  │   │ Menu     │  │ Chat             │   │  │
│  │   │ Lists    │  │  • streaming     │   │  │
│  │   │ Output   │  │  • bubbles       │   │  │
│  │   │ Modal    │  │  • interleaved   │   │  │
│  │   └──────────┘  └──────────────────┘   │  │
│  └────────────────────────────────────────┘  │
│                      │                       │
│              AIAgent interface                │
│                      │                       │
│         ┌────────────┴────────────┐           │
│         │   execAgent (stdio)     │           │
│         └────────────┬────────────┘           │
└──────────────────────┼───────────────────────┘
                        │
┌──────────────────────┼───────────────────────┐
│              dscli (subprocess)               │
└──────────────────────────────────────────────┘
```

### Key Design Decisions

- **Single Model** — All state lives in `RootModel`, no sub-models. Screen enumeration (`Screen iota`) dispatches Update/View.
- **AIAgent Interface** — The TUI never imports dscli packages. All communication goes through `AIAgent` (interface in `internal/aiagent/`).
- **Screen FSM** — The model is a finite state machine with defined transitions:

```
                ┌── History List ──→ RunningCmd ──→ ShowOutput
                │
MainMenu ───────┼── Skill List  ──→ RunningCmd ──→ ShowOutput
                │
                ├── Memory List ──→ RunningCmd ──→ ShowOutput
                │     │
                │     └── AskUser (search) ──→ RunningCmd
                │
                ├── Tool List   ──→ RunningCmd ──→ ShowOutput
                │
                ├── Project List ──→ AskUser (delete confirm)
                │
                ├── Chat ──→ AskUser (modal)
                │
                └── Cmd (Balance, Models, etc.) ──→ RunningCmd ──→ ShowOutput
```

### Package Layout

```
cmd/dscli-tui/           — Entry point (main.go)
internal/
├── aiagent/             — AIAgent interface & execAgent implementation
│   ├── agent.go         — Interface definition + message types
│   ├── exec.go          — dscli subprocess management
│   └── resolve.go       — dscli binary resolution
└── tui/                 — Bubble Tea application
    ├── model.go         — RootModel, screens, menu items
    ├── update.go        — Update loop (all screen handlers, 1600+ lines)
    ├── view.go          — View rendering (all screen views)
    ├── commands.go      — tea.Cmd factories
    ├── styles.go        — Tokyo Night color palette & lipgloss styles
    ├── model_test.go    — Model tests
    ├── history_arrow_test.go
    ├── history_flow_test.go
    ├── history_show_scroll_test.go
    └── protocol/        — Wire protocol types
        ├── types.go     — Message, Payload, MessageType
        └── payloads.go  — Concrete payload structs
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.26+**
- **dscli** — must be installed and available in `$PATH`

  ```bash
  go install github.com/dscli/dscli@latest
  ```

- **DeepSeek API key** — set via environment variable or dscli config

  ```bash
  export DEEPSEEK_API_KEY="sk-your-key-here"
  ```

### Install dscli.tui

```bash
# Option 1: go install
go install github.com/dscli/dscli.tui/cmd/dscli-tui@latest

# Option 2: Build from source
git clone https://github.com/dscli/dscli.tui.git
cd dscli.tui
make build       # or: go build -o dscli-tui ./cmd/dscli-tui
make install     # installs to $GOPATH/bin
```

### Run

```bash
dscli-tui
```

A `dscli` executable must be in your `$PATH`. The TUI resolves it automatically on startup.

---

## ⌨️ Keyboard Reference

### Global

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit immediately |

### Main Menu

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `Space` | Select item |
| `q` | Quit |

### Chat

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+J` / `Shift+Enter` | Newline in input |
| `Esc` | Exit chat (back to menu) |
| `PgUp` / `Shift+↑` / `Ctrl+↑` | Scroll up |
| `PgDn` / `Shift+↓` / `Ctrl+↓` | Scroll down |

### List Screens (History, Skill, Memory, Tool)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `Space` | Show detail |
| `PgUp` | Previous page |
| `PgDn` | Next page |
| `g` / `Home` | Go to first item |
| `G` / `End` | Go to last item |
| `Esc` / `q` | Back to menu |

### Memory List (additional)

| Key | Action |
|-----|--------|
| `/` / `s` | Search memories by keyword |

### Project List (additional)

| Key | Action |
|-----|--------|
| `d` / `D` | Delete selected project (with confirmation) |

### Show Output (scrollable text)

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up one line |
| `↓` / `j` | Scroll down one line |
| `PgUp` | Scroll up one page |
| `PgDn` | Scroll down one page |
| `g` / `Home` | Scroll to top |
| `G` / `End` | Scroll to bottom |
| `Esc` / `q` / `Enter` | Back (to list or menu) |

### AskUser Modal

| Mode | Key | Action |
|------|-----|--------|
| **Confirm** | `y` / `Y` | Yes |
| | `n` / `N` / `Esc` | No |
| **Choice** | `↑` / `k` | Previous option |
| | `↓` / `j` | Next option |
| | `Enter` / `Space` | Confirm selection |
| | `Esc` | Cancel |
| **Input** | `Enter` | Submit |
| | `Esc` | Cancel |

---

## 🔄 Workflow

1. **Start** — `dscli-tui` resolves dscli, fetches version lazily on first command
2. **Navigate** — Choose from the main menu: Chat, lists, or direct commands
3. **Interact** — Each screen provides focused keyboard-driven interaction
4. **Return** — `Esc`/`q` always returns to the main menu
5. **Quit** — `q` from menu or `Ctrl+C` anywhere

---

## 🧪 Tests

```bash
make test      # run all tests with race detector
go test ./...  # or directly
```

Current coverage: `internal/tui/` package — 5 test files, all passing.

---

## 🎨 Design

- **Color palette**: Tokyo Night (`#1a1b26` background, `#7aa2f7` primary, `#9ece6a` success)
- **Font**: Terminal default (no special glyph requirements)
- **Frameworks**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)

---

## 🤝 Contributing

Contributions, bug reports, and feature requests are welcome!

- Repository: [github.com/dscli/dscli.tui](https://github.com/dscli/dscli.tui)
- Issues: Create an Issue on the repository

Please ensure tests pass before submitting:

```bash
make test
```

### Development Guidelines

- **AIAgent interface** — Keep the TUI decoupled from dscli internals. New features should extend the interface, not bypass it.
- **Screen enum** — Each new screen gets a `Screen*` constant, an `update*` handler, and a `view*` method.
- **Style** — Follow existing patterns (Tokyo Night palette, lipgloss styles in `styles.go`).
- **Tests** — Add tests for new list parsers and model behavior.

---

## 📄 License

Apache License 2.0

Copyright © 2026 JUN JIE NAN <nanjunjie@gmail.com>

---

**dscli.tui** — The beautiful face of dscli.
