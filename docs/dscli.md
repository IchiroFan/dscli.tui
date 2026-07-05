# dscli.tui 架构设计

## 1. 项目背景

现有 `dscli.gitcode` 项目将 dscli 功能与 TUI 紧耦合在一起：
- TUI 直接导入 `dsc.Client` 调用 DeepSeek API
- TUI 直接依赖 `prompt`/`skills`/`toolcall` 等内部包
- 没有 `ask_user` 工具调用的处理机制

目标：将 TUI 分离为独立项目 `dscli.tui`，通过 AIAgent 接口与 dscli 解耦。

## 2. 三个项目的定位

| 项目 | 路径 | 模块 | 职责 |
|------|------|------|------|
| **dscli** | `/home/ichiro/go_project/dscli` | `github.com/dscli/dscli` | CLI 工具，添加 `--json` 模式支持 |
| **dscli.gitcode** | `/home/ichiro/go_project/dscli.gitcode` | `gitcode.com/dscli/dscli` | 旧合并项目，分离后删除 TUI 代码 |
| **dscli.tui** | `/home/ichiro/go_project/dscli.tui` | `gitcode.com/dscli/dscli.tui` | **新**独立 TUI 项目（本项目） |

dscli 的 `--json` 模式修改在 dscli 项目的 `feature/chat-json-mode` 分支进行，不在本项目范围内。


## 3. TUI 设计参考（来自 dscli.gitcode）

本项目 TUI 的设计模式、架构、配色和交互方式继承自 `dscli.gitcode` 的 `internal/tui/` 实现。以下是从该实现中提炼的关键设计决策，dscli.tui 在移植时应保持一致。

### 3.1 架构模式

| 模式 | 说明 |
|------|------|
| **Screen iota 枚举** | `ScreenDashboard`, `ScreenBalance`, `ScreenModels`, `ScreenHistory`, `ScreenHistoryDetail`, `ScreenSkills`, `ScreenPrompt`, `ScreenChat` 八个屏幕 |
| **单 Model 结构** | 所有状态集中在 `Model` 一个结构中，无子模型 |
| **Update 路由** | `Update()` → `tea.KeyMsg` → `handleKeyPress(key)` → 按 `m.Screen` 分发到各屏幕 handler |
| **View 路由** | `View()` → 按 `m.Screen` 分发到 `viewDashboard()`, `viewBalance()` 等 |
| **Vim 导航键** | `j`/`k` 或 `↑`/`↓` 导航，`Enter` 选择，`q`/`esc` 返回 |
| **tea.Cmd 封装** | 数据获取通过 `tea.Cmd` 回调（`fetchBalance`, `fetchModels` 等）返回自定义消息 |

### 3.2 自定义消息类型

每个屏幕有自己的数据加载消息：

```go
type balanceMsg struct { resp *dsc.BalanceResponse; err error }
type modelsMsg struct { resp *dsc.ModelsResponse; err error }
type historyMsg struct { messages []*prompt.Message; err error }
type skillsMsg struct { infos []skills.SkillInfo; err error }
type promptContentMsg struct { content string; err error }
type chatStreamMsg struct {
    lines       []chatLine
    err         error
    done        bool
    replaceLast bool  // 流式增量：替换上一条而非追加
}
```

dscli.tui 中，这些消息将包装 `aiagent.AIAgent` 的返回值，而非 dscli 内部类型。

### 3.3 配色方案（DeepSeek 风格）

| 用途 | 颜色 | HEX |
|------|------|-----|
| 背景 | 深色 | `#1a1b26` |
| 面板 | 次深色 | `#24253e` |
| 边框 | 灰色 | `#565f89` |
| 文字 | 浅灰 | `#c0caf5` |
| 次要文字 | 浅灰蓝 | `#9aa5ce` |
| 主色调 | 蓝色 | `#7aa2f7` |
| 成功/用户 | 绿色 | `#9ece6a` |
| 暖色强调 | 桃色 | `#ff9e64` |
| 错误 | 红色 | `#f7768e` |
| 青色 | 青 | `#2ac3de` |
| 紫色 | 紫 | `#bb9af7` |
| 黄色/工具 | 金色 | `#e0af68` |
| 蓝绿色 | 青绿 | `#1abc9c` |

### 3.4 Chat 界面设计

Chat 是全屏模式，布局如下：

```
┌────────────────────────────────┐
│ 💬 Chat                        │  header
│                                │  spacer
│ ▲ scrolled (N lines)           │  指示器（有滚动时显示）
│ ┌──────────────────────────┐   │
│ │ 🧠 助手消息气泡           │   │  chat area
│ │ ──────────────────────── │   │  （行级滚动）
│ │ 💭 思考过程:              │   │
│ │  ✨ reasoning content    │   │
│ │ ──────────────────────── │   │
│ │ 回复内容                  │   │
│ └──────────────────────────┘   │
│   👤 用户消息气泡               │
│ ┌──────────────────────┐       │
│ │ 输入框                │       │  输入区域（固定底部）
│ └──────────────────────┘       │
│ enter send • i • j/k • G • esc │  帮助栏（固定底部）
│ dscli v1.0 │ 📁 proj │ 🤖     │  状态栏（固定底部）
└────────────────────────────────┘
```

**气泡设计**：
- 用户消息：右对齐，绿色边框（`👤` 前缀）
- 助手消息组：左对齐，蓝色边框（`🧠` 前缀）
- 连续的非用户消息（assistant + tool + reasoning + state）合并为**一个统一气泡**，内部分段显示
- 思考/推理内容：紫色边框，斜体文字
- 工具调用结果：金色边框，斜体文字显示 `工具名: 结果（截断至200字）`
- 截断警告：红色粗体
- 会话统计（时间/花费/余额）：灰色斜体状态行

**流式更新**：`replaceLast` 机制——每个新 chunk 替换上一条占位消息，实现内容逐渐增长的效果。

### 3.5 状态栏

全宽底栏，格式：
```
dscli v1.0 │ 📁 ~/project │ 🤖 deepseek-chat │ 💬 Chat
```

包含：版本徽标（紫色背景）、项目路径、当前模型、当前屏幕名。

### 3.6 其它屏幕模式

| 屏幕 | 渲染模式 |
|------|----------|
| Dashboard | Logo（ASCII art 渐变）+ 菜单列表 |
| Balance | 字段标签 + 值（Currency/Total/Granted/Topped Up） |
| Models | 列表（每行：ID + Object + OwnedBy） |
| History | 每项 2 行（主行：ID + 角色 + 时间；预览行：截断至80字的内容） |
| History Detail | 字段 + Reasoning Content + Content（文本区垂直滚动） |
| Skills | 列表（每行：Name + Scope + 自动注入标识） |
| Prompt | 文本区（垂直滚动） |

### 3.7 关键实现细节

- **滚动计算**：`visibleItems(linesPerItem)` 根据终端高度和每项行数计算可见项数
- **溢出指示器**：列表顶部显示 `▲ N items above`，底部显示 `▼ N items below`
- **气泡宽度**：最大可用宽度的 72%（`bubbleMaxPercent = 72`）
- **渲染注意**：使用纯空格对齐代替 lipgloss 对齐，避免 ANSI-on-ANSI 渲染问题
- **聊天滚动**：基于显示行数（而非消息数），`ChatScroll` 从底部向上偏移
- **`lastChatMaxScroll`**：全局变量缓存 View 中计算的最大滚动值，供键盘 handler 使用
## 4. 项目结构
/home/ichiro/go_project/dscli.tui/
├── main.go                      # 入口
├── root.go                      # cobra 根命令 + tui 子命令注册
├── go.mod                       # 独立 module: gitcode.com/dscli/dscli.tui
├── internal/
│   ├── aiagent/                 # AIAgent 接口 + 实现
│   │   ├── agent.go             #   接口定义 + ChatSession + 结果消息类型
│   │   ├── exec.go              #   ExecAgent 实现（含 ChatSession 创建）
│   │   ├── resolve.go           #   dscli 路径查找（执行版验证）
│   │   └── agent_test.go
│   ├── tui/
│   │   ├── protocol/            # JSON-line 协议类型定义（最终将移至 dscli）
│   │   │   ├── types.go         #   Message, MessageType, Payload sealed interface
│   │   │   └── payloads.go      #   ChatRequest, ChatChunk, AskUser, Command 等负载
│   │   ├── tui.go               #   Model 定义 + Init + New
│   │   ├── update.go            #   Update 方法
│   │   ├── view.go              #   View 方法
│   │   ├── styles.go            #   Lip Gloss 样式
│   │   ├── chat.go              #   Chat 相关逻辑
│   │   └── tui_test.go
│   └── types/                   # 通用类型（不与 dscli 绑定）
│       └── types.go
├── pkg/
│   └── jsonline/                # JSON-line 编解码器（最终移至 dscli）
│       └── codec.go
└── docs/
    └── adr/                     # 架构决策记录


## 5. AIAgent 接口设计

### 5.1 核心原则
- 每个方法对应一个 dscli **顶级子命令**
- 输入输出使用 `protocol.*Payload` 类型（而非 `string`），提供编译期类型安全
- TUI 只依赖 `aiagent.AIAgent` 接口和 `protocol` 包，**不依赖 dscli 的任何内部包**
- 通信协议（JSON-line）和协议类型最终将定义在 dscli 项目中，dscli.tui 是遵循者

### 5.2 接口定义

```go
package aiagent

import "context"

// AIAgent 是 dscli 功能的抽象层。
// TUI 只通过此接口与 dscli 交互，不直接导入任何 dscli 包。
type AIAgent interface {
    // ─── 一级命令（无子子命令） ───────────────────────────

    // Balance 查询账户余额
    // 对应: dscli balance [--format json]
    Balance(ctx context.Context, format string) (string, error)

    // Models 列出可用模型
    // 对应: dscli models [--format json] [--price]
    Models(ctx context.Context, format string, showPrice bool) (string, error)

    // Version 显示版本信息
    // 对应: dscli version
    Version(ctx context.Context) (string, error)

    // Flycheck 静态代码检查
    // 对应: dscli flycheck [--emacs] <path>
    Flycheck(ctx context.Context, path string, emacs bool) (string, error)

    // FIM Fill-in-the-middle
    // 对应: dscli fim [...args]
    FIM(ctx context.Context, args ...string) (string, error)

    // ─── 二级命令（有子子命令） ───────────────────────────

    History(ctx context.Context, subcmd string, args ...string) (string, error)
    Skill(ctx context.Context, subcmd string, args ...string) (string, error)
    Prompt(ctx context.Context, subcmd string, args ...string) (string, error)
    Memory(ctx context.Context, subcmd string, args ...string) (string, error)
    Project(ctx context.Context, subcmd string, args ...string) (string, error)
    Role(ctx context.Context, subcmd string, args ...string) (string, error)
    Tool(ctx context.Context, subcmd string, args ...string) (string, error)
    Mail(ctx context.Context, subcmd string, args ...string) (string, error)
    Service(ctx context.Context, subcmd string, args ...string) (string, error)

    // ─── 交互式对话 ─────────────────────────────────────

    // NewChatSession 创建交互式聊天会话。
    // 使用 JSON-line 协议通过 stdio 与 dscli chat --json 通信。
    NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error)
}
```

### 5.3 ChatSession 类型定义

```go
// ChatSessionOptions 会话配置
type ChatSessionOptions struct {
    Model      string // 模型名
    Role       string // 角色 (dev/expert/review/test)
    HistSize   int    // 历史消息数
    Stream     bool   // 流式输出
    DscliPath  string // dscli 可执行文件路径
    ProjectDir string // 项目目录
}

// ChatSession 活跃的聊天会话
type ChatSession struct {
    Events <-chan ChatEvent   // 从 dscli 收到的事件
    Send   chan<- ChatMessage // 发送消息到 dscli
    Done   <-chan struct{}   // 会话结束信号
    close  func() error      // 关闭会话
}

func (s *ChatSession) Close() error { return s.close() }

// ChatEvent 从 dscli 收到的事件
type ChatEvent struct {
    Type     ChatEventType
    Delta    string       // EventDelta 时的增量内容
    Content  string       // EventContent 时的完整内容
    Role     string       // 角色
    Done     bool         // 对话回合结束
    AskUser  *AskUserInfo // EventAskUser 时的信息
    ToolCall *ToolCallInfo // 工具调用信息
    Usage    *UsageInfo   // token 用量
    Err      error        // 错误
}

type ChatEventType int
const (
    EventDelta   ChatEventType = iota // 流式增量
    EventContent                      // 完整响应（含工具调用标志）
    EventAskUser                      // AI 询问用户 → 需交互
    EventDone                         // 回合结束
    EventError                        // 错误
)

type AskUserInfo struct {
    ToolCallID string // 工具调用 ID
    Question   string // 问题内容
    Timeout    int    // 超时秒数
}

type ToolCallInfo struct {
    IDs       []string // 本次所有工具调用 ID
    Names     []string // 本次所有工具名
}

type UsageInfo struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// ChatMessage 发送到 dscli 的消息
type ChatMessage struct {
    Type       ChatMsgType
    Content    string // 用户消息内容
    ToolResult string // 工具调用结果（回复 ask_user）
    ToolCallID string // 对应的工具调用 ID
}

type ChatMsgType int
const (
    MsgUserMessage ChatMsgType = iota // 用户发送消息
    MsgToolResult                    // 工具调用结果（含空回复）
    MsgCancel                        // 取消当前回合
)
```

## 6. exec 实现方案

### 6.1 非交互命令

```go
type execAgent struct {
    dscliPath string // dscli 可执行文件路径
}

func (a *execAgent) Balance(ctx context.Context, format string) (string, error) {
    args := []string{"balance"}
    if format != "" {
        args = append(args, "--format", format)
    }
    return a.execDS(ctx, args...)
}

func (a *execAgent) History(ctx context.Context, subcmd string, args ...string) (string, error) {
    cmdArgs := append([]string{"history", subcmd}, args...)
    return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) execDS(ctx context.Context, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, a.dscliPath, args...)
    out, err := cmd.CombinedOutput()
    return string(out), err
}
```

### 6.2 dscli 路径查找（执行版验证）

替换 `exec.LookPath` 为实际执行 `dscli version` 验证：

```go
func resolveDSCLIPath(hint string) string {
    candidates := []string{}
    if hint != "" {
        candidates = append(candidates, hint)
    }
    if p := os.Getenv("DSCLI_PATH"); p != "" {
        candidates = append(candidates, p)
    }
    candidates = append(candidates, "dscli")
    candidates = append(candidates, "./dscli")

    for _, c := range candidates {
        cmd := exec.Command(c, "version")
        out, err := cmd.CombinedOutput()
        if err != nil {
            continue
        }
        output := strings.TrimSpace(string(out))
        if strings.Contains(output, "版本") || strings.Contains(output, "dscli") {
            return c
        }
    }
    return "" // 未找到
}
```

这样同时验证「路径存在」+「确实可执行」+「是 dscli 工具」三个条件。

### 6.3 依赖

AIAgent 的唯一外部依赖是 `os/exec` — **不导入任何 dscli 包**。

## 7. Chat JSON-line 协议（dscli.tui 视角）

### 7.1 为什么需要 JSON-line 协议

dscli 的 `ask_user` 工具调用当前通过打开编辑器（vim/nano）获取用户输入。在 TUI 中不能打开编辑器，需要：
1. dscli 输出问题 → TUI 显示模态框
2. TUI 获取用户输入 → 发送回复给 dscli
3. dscli 恢复对话循环

JSON-line over stdio 是实现此交互的最简洁方式。

### 7.2 协议概览

dscli 以 `dscli chat --json` 启动，通过 stdio 交换 JSON lines。

#### Stdout（dscli → TUI）

| 类型 | 触发时机 | 字段 |
|------|---------|------|
| `delta` | 流式输出每个 chunk | `role`, `content` |
| `content` | 完整响应（含工具调用标志） | `role`, `content`, `reasoning`, `tool_calls` |
| `tool_call` | 响应包含工具调用 | `tool_calls[]` |
| `ask_user` | ask_user 工具需要用户输入 | `tool_call_id`, `question`, `timeout` |
| `tool_result` | 工具执行完毕 | `tool_call_id`, `name`, `result`, `success` |
| `done` | 回合结束 | `usage` |
| `error` | 致命错误 | `message` |

#### Stdin（TUI → dscli）

| 类型 | 触发时机 | 字段 |
|------|---------|------|
| `message` | 用户发送新消息 | `content` |
| `tool_result` | 用户回复 ask_user 问题 | `tool_call_id`, `content` |
| `cancel` | 用户取消当前回合 | — |

### 7.3 ask_user 三种响应语义

| 用户操作 | 消息类型 | 含义 | dscli 行为 |
|---------|---------|------|-----------|
| 输入内容后按 Enter | `MsgToolResult` + 非空 `Content` | 用户回答了问题 | 正常继续工具执行 |
| 直接按 Enter（内容空） | `MsgToolResult` + 空 `Content` | 回答了但内容为空 | 空字符串作为合法工具结果传给 DeepSeek |
| 按 Esc | `MsgCancel` | 取消本轮 | **终止当前 ChatRound** |
| 超时（dscli 侧） | —（内部检测） | 用户没回应 | **以 error 终止 ChatRound** |

### 7.4 完整 ask_user 交互序列

```
dscli                                TUI
  │                                    │
  │── stdout: {"type":"content",...}   │  显示助手消息
  │── stdout: {"type":"tool_call",..}  │  显示工具调用信息
  │                                    │
  │── stdout: {"type":"ask_user",      │
  │     "tool_call_id":"call_1",       │
  │     "question":"标准库还是 Gin?",  │
  │     "timeout":60}                  │
  │                                    │
  │                                    │── 显示模态框 + 倒计时
  │                                    │   ╭─────────────────────╮
  │                                    │   │ 🤖 AI 询问:         │
  │                                    │   │ 标准库还是 Gin?      │
  │                                    │   │                     │
  │                                    │   │ > [用户输入]        │
  │                                    │   │ [Enter] 发送 [Esc]取消│
  │                                    │   ╰─────────────────────╯
  │                                    │
  │  ┌─ 场景 A：用户回答 ───────────┐  │
  │  │  stdin: {"type":"tool_result",│  │
  │  │    "tool_call_id":"call_1",   │  │  ← 用户输入 "Gin" 按 Enter
  │  │    "content":"Gin"}           │  │
  │  │  handleAskUser 返回 content   │  │  dscli 继续执行
  │  │  stdout: {"type":"tool_result",│  │
  │  │    "tool_call_id":"call_1",   │  │
  │  │    "name":"ask_user",         │  │
  │  │    "result":"Gin","success":true}│  │
  │  │  (继续 ChatRound)             │  │
  │  └───────────────────────────────┘  │
  │                                    │
  │  ┌─ 场景 B：用户按 Esc ─────────┐  │
  │  │  stdin: {"type":"cancel"}     │  │  ← 用户按 Esc
  │  │  dscli 检测取消信号           │  │
  │  │  stdout: {"type":"error",     │  │
  │  │    "message":"cancelled"}     │  │
  │  │  ChatRound 终止               │  │
  │  └───────────────────────────────┘  │
  │                                    │
  │  ┌─ 场景 C：超时 ────────────────┐  │
  │  │  dscli 读取 stdin 超时（60s）  │  │  TUI 超时提示
  │  │  stdout: {"type":"error",     │  │
  │  │    "message":"ask_user timeout"}│  │
  │  │  ChatRound 终止               │  │
  │  └───────────────────────────────┘  │
```

### 7.5 ChatSession 内部实现

```go
func (a *execAgent) NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error) {
    args := []string{"chat", "--json", "--role", opts.Role}
    if opts.Stream {
        args = append(args, "--stream")
    }

    ctx, cancel := context.WithCancel(ctx)
    cmd := exec.CommandContext(ctx, a.dscliPath, args...)
    cmd.Dir = opts.ProjectDir

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    events := make(chan ChatEvent, 100)
    sendCh := make(chan ChatMessage, 10)
    done := make(chan struct{})

    if err := cmd.Start(); err != nil {
        cancel()
        return nil, fmt.Errorf("start dscli chat --json: %w", err)
    }

    go readEvents(stdout, events, done)
    go writeMessages(stdin, sendCh, done)
    go io.Copy(os.Stderr, stderr)

    go func() {
        cmd.Wait()
        close(done)
        cancel()
    }()

    return &ChatSession{
        Events: events,
        Send:   sendCh,
        Done:   done,
        close: func() error {
            cancel()
            return cmd.Wait()
        },
    }, nil
}
```

## 8. TUI Model 改造

### 8.1 当前耦合点 vs 解耦方案

| 当前 TUI (dscli.gitcode) | 改造后 TUI (dscli.tui) |
|--------------------------|------------------------|
| `client dsc.Client` | `agent aiagent.AIAgent` |
| `*dsc.BalanceResponse` | `string`（解析 json/table 输出） |
| `[]dsc.Model` | `string` |
| `[]*prompt.Message` | `string` |
| `startChatStream(ctx, client, input, ch)` | `session := agent.NewChatSession(...)` |
| 工具调用在 goroutine 中自动执行 | JSON-line 协议内部处理 |
| 无 `ask_user` 支持 | `EventAskUser` → TUI 模态框 |

### 8.2 Model 新结构

```go
type Model struct {
    // AIAgent（替代 dsc.Client）
    agent aiagent.AIAgent

    // App info
    Version     string
    Build       string
    ProjectRoot string

    // Screen
    Screen     Screen
    Width, Height int
    Cursor, Scroll int
    ErrorMsg   string

    // Data（从 agent 返回的文本中提取）
    BalanceInfos []map[string]string
    ModelList    []modelInfo
    HistoryItems []historyItem
    SkillInfos   []skillInfo
    PromptContent string

    // Chat
    ChatInput    textinput.Model
    ChatMessages []chatLine
    ChatLoading  bool
    ChatSpinner  spinner.Model
    ChatSession  *aiagent.ChatSession

    // Ask User Modal
    ShowAskModal  bool
    AskQuestion   string
    AskToolCallID string
    AskInput      textinput.Model
}

type modelInfo struct {
    ID      string
    Object  string
    OwnedBy string
}
type historyItem struct {
    ID      int64
    Role    string
    Content string
    Time    string
}
type skillInfo struct {
    Name       string
    Scope      string
    AutoInject string
}
```

### 8.3 Update 路由变化

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // ... 不变 ...

    case tea.KeyMsg:
        if m.ShowAskModal {
            return m.handleAskModalKey(msg)
        }
        if m.Screen == ScreenChat && m.ChatInput.Focused() && !m.ChatLoading {
            return m.handleChatInputKeys(msg)
        }
        return m.handleKeyPress(msg.String())

    // ── Agent 结果 ──
    case agentResultMsg:
        return m.handleAgentResult(msg)

    // ── Chat 事件 ──
    case chatEventMsg:
        return m.handleChatEvent(msg.event)

    // ... 其余不变 ...
    }
}

func (m Model) handleChatEvent(ev aiagent.ChatEvent) (tea.Model, tea.Cmd) {
    switch ev.Type {
    case aiagent.EventDelta:
        m.appendChatDelta(ev.Delta)
    case aiagent.EventContent:
        m.finalizeChatContent(ev)
    case aiagent.EventAskUser:
        m.ShowAskModal = true
        m.AskQuestion = ev.AskUser.Question
        m.AskToolCallID = ev.AskUser.ToolCallID
        m.AskInput.Focus()
    case aiagent.EventDone:
        m.ChatLoading = false
    case aiagent.EventError:
        m.ChatLoading = false
        m.ErrorMsg = ev.Err.Error()
    }
    return m, nil
}

func (m Model) handleAskModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        answer := m.AskInput.Value()
        m.ShowAskModal = false
        m.AskInput.SetValue("")
        m.ChatSession.Send <- aiagent.ChatMessage{
            Type:       aiagent.MsgToolResult,
            ToolResult: answer,
            ToolCallID: m.AskToolCallID,
        }
        return m, nil
    case "esc":
        m.ShowAskModal = false
        m.AskInput.SetValue("")
        m.ChatSession.Send <- aiagent.ChatMessage{
            Type: aiagent.MsgCancel,
        }
        return m, nil
    }
    var cmd tea.Cmd
    m.AskInput, cmd = m.AskInput.Update(msg)
    return m, cmd
}
```

### 8.4 Chat 启动

```go
func (m Model) startChatSession(userInput string) (tea.Model, tea.Cmd) {
    m.ChatLoading = true
    m.ChatMessages = append(m.ChatMessages, chatLine{Role: "user", Content: userInput})

    session, err := m.agent.NewChatSession(m.ctx, aiagent.ChatSessionOptions{
        Model:      "deepseek-chat",
        Role:       "dev",
        HistSize:   8,
        Stream:     true,
        ProjectDir: m.ProjectRoot,
    })
    if err != nil {
        m.ErrorMsg = err.Error()
        m.ChatLoading = false
        return m, nil
    }
    m.ChatSession = session

    session.Send <- aiagent.ChatMessage{
        Type:    aiagent.MsgUserMessage,
        Content: userInput,
    }

    return m, tea.Batch(
        m.ChatSpinner.Tick,
        listenChatSession(session),
    )
}

func listenChatSession(session *aiagent.ChatSession) tea.Cmd {
    return func() tea.Msg {
        select {
        case ev, ok := <-session.Events:
            if !ok { return chatEventMsg{} }
            return chatEventMsg{event: ev}
        case <-session.Done:
            return chatEventMsg{done: true}
        }
    }
}
```

## 9. 类型解析：将 dscli 输出转为结构化数据

对于 Balance、Models 等命令，TUI 需要从 dscli 的文本输出中提取结构化数据以渲染 UI。

### 9.1 方案：优先使用 JSON 输出

```go
// Balance: 使用 --format json
func (m Model) fetchBalance() tea.Cmd {
    return func() tea.Msg {
        text, err := m.agent.Balance(m.ctx, "json")
        if err != nil { return agentResultMsg{err: err} }

        var resp struct {
            IsAvailable  bool                 `json:"is_available"`
            BalanceInfos []map[string]string  `json:"balance_infos"`
        }
        json.Unmarshal([]byte(text), &resp)

        return balanceDataMsg{
            infos: resp.BalanceInfos,
            available: resp.IsAvailable,
        }
    }
}
```

### 9.2 对不支持 JSON 的命令

对 `history list`、`skill list` 等没有 `--format json` 的子命令，TUI 直接显示 dscli 的输出文本，或通过简单的行解析提取关键字段。

## 10. 实施阶段

### 10.1 Phase 1: 基础设施
- [x] 初始化 `dscli.tui` go.mod + 骨架
- [x] 定义 `tui/protocol` 协议类型（Message, Payload, 所有负载类型）
- [x] 实现 `pkg/jsonline` JSON-line 编解码器
- [x] 实现 `aiagent.AIAgent` 接口定义（使用 `protocol.*Payload` 类型返回）
- [x] 实现 `aiagent.execAgent` 非交互命令 + ChatSession
- [x] 实现 `aiagent.resolveDSCLIPath`（dscli version 验证）
- [ ] 移植 TUI 核心框架（Model + Init + Update + View 骨架）

### 10.2 Phase 2: 非交互式屏幕
- [ ] 移植 Dashboard 屏幕
- [ ] 移植 Balance 屏幕（通过 agent.Balance）
- [ ] 移植 Models 屏幕（通过 agent.Models）
- [ ] 移植 History 屏幕（通过 agent.History）
- [ ] 移植 Skills 屏幕（通过 agent.Skill）
- [ ] 移植 Prompt 屏幕（通过 agent.Prompt）

### 10.3 Phase 3: 交互式对话 + ask_user
- [ ] 实现 `aiagent.ChatSession`（JSON-line 读写）
- [ ] 移植 Chat 屏幕（通过 ChatSession）
- [ ] 实现 AskUser 模态框（含三种语义：回答/空回复/取消）
- [ ] 注意：依赖 dscli 的 `chat --json` 模式实现

### 10.4 Phase 4: 完善
- [ ] 移植剩余屏幕（Project/Role/Tool/Mail/Service/Flycheck）
- [ ] 错误处理（dscli 未安装、版本不兼容等）
- [ ] 测试

## 11. 边界情况

| 场景 | 处理方式 |
|------|----------|
| dscli 不在 PATH 中 | 启动时报错 + 提示配置 `DSCLI_PATH` |
| `dscli version` 执行失败 | 视为 dscli 未安装，报错 |
| ChatSession 中断 | 返回 `EventError`，TUI 显示错误提示 |
| ask_user 超时 | dscli 侧检测 → error 事件 → 终止 ChatRound |
| ask_user 取消（Esc） | TUI 发送 `MsgCancel` → dscli error → 终止 ChatRound |
| ask_user 空回复 | 用户未输入内容但按 Enter → 空字符串作为合法工具结果 |
| 多个 ask_user 同时 | 当前设计不支持——一轮只有一个 ask_user（串行执行） |
| dscli 版本不兼容 | 接口方法兼容：增加方法不破坏旧版本 |
| 终端尺寸过小 | TUI 已有最小尺寸保护逻辑 |
