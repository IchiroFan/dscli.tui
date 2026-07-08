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
| **dscli** | - | `github.com/dscli/dscli` | CLI 工具，添加 `--json` 模式支持 |
| **dscli.tui** | - | `github.com/dscli/dscli.tui` | **新**独立 TUI 项目（本项目） |

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
dscli.tui/
├── main.go                      # 入口
├── root.go                      # cobra 根命令 + tui 子命令注册
├── go.mod                       # 独立 module: github.com/dscli/dscli.tui
├── internal/
│   ├── aiagent/                 # AIAgent 接口 + 实现
│   │   ├── agent.go             #   接口定义 + ChatSession + 结果消息类型
│   │   ├── exec.go              #   ExecAgent 实现（含 ChatSession 创建）
│   │   ├── resolve.go           #   dscli 路径查找（执行版验证）
│   │   └── agent_test.go
│   ├── tui/
│   │   ├── protocol/            # 消息类型定义（AIAgent 接口的输入输出载体）
│   │   │   ├── types.go         #   Message, MessageType, Payload 密封接口
│   │   │   └── payloads.go      #   ChatRequest, ChatChunk, AskUser, Command 等负载
│   │   ├── tui.go               #   Model 定义 + Init + New
│   │   ├── update.go            #   Update 方法
│   │   ├── view.go              #   View 方法
│   │   ├── styles.go            #   Lip Gloss 样式
│   │   ├── chat.go              #   Chat 相关逻辑
│   │   └── tui_test.go
│   └── types/                   # 通用类型（不与 dscli 绑定）
│       └── types.go
└── docs/
    └── adr/                     # 架构决策记录


## 5. AIAgent 接口设计

### 5.1 核心原则
- 每个方法对应一个 dscli **顶级子命令**
- 输入输出使用 `protocol.*Payload` 类型（而非 `string`），提供编译期类型安全
- TUI 只依赖 `aiagent.AIAgent` 接口和 `protocol` 包，**不依赖 dscli 的任何内部包**

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
    // 通过 stdin/stdout 与 dscli chat 子进程通信。
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

### 6.4 Raw Exec 执行方案（永久方案）
非交互命令（Balance, Models, History 等）采用 **Raw Exec** 方式：直接执行 `dscli <args>`，捕获 stdout+stderr，包裹为 `CommandResultPayload` 返回。**不依赖 `--json-line` 标志**。

```go
// execDSRaw 直接执行 dscli <args> 并返回原始文本输出。
func (a *execAgent) execDSRaw(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error) {
    cmd := exec.CommandContext(ctx, a.dscliPath, args...)
    out, err := cmd.CombinedOutput()
    output := strings.TrimSpace(string(out))
    if err != nil {
        return &protocol.CommandResultPayload{
            Success: false,
            Data:    output,
        }, fmt.Errorf("dscli %v: %w\n%s", args, err, output)
    }
    return &protocol.CommandResultPayload{
        Success: true,
        Data:    output,
    }, nil
}
```

**策略**：
1. 所有非交互命令使用 `execDSRaw`，不使用 JSON-line 协议
2. `execDS()`（旧 JSON-line 路径）保留但不再调用，仅作参考
3. ChatSession 使用直接 stdin/stdout 通信（见 §7），不依赖任何协议标志

**优点**：零额外依赖，所有菜单项不受 dscli 协议版本影响。

## 7. Chat 通信协议（dscli.tui 视角）

### 7.1 会话生命周期

Chat 会话采用单轮问答模式（one-shot exchange）：

1. TUI 通过 `AIAgent.NewChatSession()` 创建 `ChatSession`
2. dscli 子进程启动后立即就绪（发射 `TypeReady`）
3. TUI 发送用户消息（`ChatRequestPayload`），关闭 stdin 信号 EOF
4. dscli 处理请求，输出响应到 stdout
5. TUI 按字节读取 stdout，以固定阈值切分后通过 channel 发射 `ChatChunkPayload`
6. 读取完毕（EOF）后发射 `ChatDonePayload`
7. dscli 进程退出，`cmd.Wait()` 收集退出码

**关键设计**：不依赖 dscli 的 `--json` 或 `--json-line` 标志，直接与 `dscli chat` 的标准 stdin/stdout 通信。

### 7.2 流式输出机制

dscli chat 默认一次性输出完整响应（非流式 API）。TUI 通过**字节级读取 + 固定阈值切分**实现增量显示：

```go
const chunkThreshold = 10           // 积累 10 字节或遇到 \n 时发射 chunk
const chunkFlushDelay = 30 * time.Millisecond // chunk 间最小间隔，控制输出节奏
```

- 使用 `bufio.NewReader(stdout).ReadByte()` 逐字节读取
- 每积累 `chunkThreshold` 字节或遇到 `\n` 即发射一个 chunk
- 两次 chunk 间 sleep `chunkFlushDelay`，避免 burst 导致 TUI 渲染阻塞
- 这种方式即使对于无换行的长文本也能平滑增量显示

### 7.3 AskUser 交互语义

dscli 的 `ask_user` 工具调用通过 `$EDITOR` 环境变量打开编辑器获取用户输入。TUI 将 EDITOR 替换为自身的 socket 客户端，桥接回 TUI 模态框。

三种响应语义（socket 桥接的详细协议见 §12）：

| 用户操作 | 语义 | dscli 行为 |
|---------|------|-----------|
| 输入内容后按 Enter | 用户回答了问题 | 正常继续工具执行 |
| 直接按 Enter（内容空） | 回答了但内容为空 | 空字符串作为合法工具结果传给 DeepSeek |
| 按 Esc | 取消本轮 | **终止当前 ChatRound** |
| 超时（300s） | 用户没回应 | 以 error 终止 ChatRound |

### 7.4 ChatSession 核心实现

```go
func (a *execAgent) NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error) {
    args := []string{"chat"}
    ctx, cancel := context.WithCancel(ctx)
    cmd := exec.CommandContext(ctx, a.dscliPath, args...)
    cmd.Dir = opts.ProjectDir
    cmd.Env = append(os.Environ(), opts.Env...)

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

    cmd.Start()

    events := make(chan *protocol.Message, 3)
    sendCh := make(chan *protocol.Message, 10)
    done := make(chan struct{})
    waitDone := make(chan error, 1)

    go func() {
        // 1. Emit TypeReady
        // 2. Wait for ChatRequestPayload from TUI
        // 3. Write user message to stdin, close stdin → EOF
        // 4. Read stdout byte-by-byte, emit ChatChunkPayload per chunkThreshold
        // 5. On EOF, emit ChatDonePayload
        // 6. cmd.Wait(), report exit code
    }()

    return &ChatSession{Events: events, Send: sendCh, Done: done,
        close: func() error { cancel(); return <-waitDone }}
}
```

完整实现见 `internal/aiagent/exec.go`。AskUser socket 桥接见 §12。

## 8. TUI Model 改造

### 8.1 当前耦合点 vs 解耦方案

| 当前 TUI (dscli.gitcode) | 改造后 TUI (dscli.tui) |
|--------------------------|------------------------|
| `client dsc.Client` | `agent aiagent.AIAgent` |
| `*dsc.BalanceResponse` | `string`（解析 json/table 输出） |
| `[]dsc.Model` | `string` |
| `[]*prompt.Message` | `string` |
| `startChatStream(ctx, client, input, ch)` | `session := agent.NewChatSession(...)` |
| 工具调用在 goroutine 中自动执行 | socket 桥接 / 主 goroutine 处理 |
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

### 10.1 Phase 1: 基础设施 ✓
- [x] 初始化 `dscli.tui` go.mod + 骨架
- [x] 定义 `tui/protocol` 协议类型（Message, Payload, 所有负载类型）
- [x] 实现 `aiagent.AIAgent` 接口定义（使用 `protocol.*Payload` 类型返回）
- [x] 实现 `aiagent.execAgent` 非交互命令 + ChatSession
- [x] 实现 `aiagent.resolveDSCLIPath`（dscli version 验证）
- [x] 移植 TUI 核心框架（Model + Init + Update + View + 6 屏幕路由）
- [x] Screen 枚举 / textinput / spinner / 滚动支持
- [x] 单元测试 30 个，全部通过
- [x] Logo（渐变 ASCII art + 双线边框，与 dscli.gitcode 一致）

### 10.2 Phase 2: 显示 dscli 原始输出 ✓
**目标**：让 TUI 各菜单项能调用 `dscli <subcmd>` 并展示原始输出。
**方案**：在 `execAgent` 中使用 `execDSRaw()` 直接执行 `dscli <args>` 捕获 stdout/stderr。
- [x] 添加 `execDSRaw(ctx, args...)` 方法，返回 `*CommandResultPayload`
- [x] 修改 `Balance()` / `Models()` / `Version()` / `Flycheck()` / `FIM()` 使用 raw exec
- [x] 修改所有子命令（History/Skill/Prompt/Memory/Project/Role/Tool/Mail/Service）使用 raw exec
- [x] 更新 `formatCommandResult()` 适配原始文本输出
- [x] 验证所有非交互菜单项（1-13）可正常显示 dscli 输出
- [x] 错误处理：dscli 未安装、命令执行失败
- [x] 更新测试覆盖 raw exec 路径
- [x] Status Bar（底部显示版本 / 项目路径 / 模型 / 屏幕名）
- [x] Chat 气泡（UserBubbleBase / AssistantBubbleBase 边框）
- [x] Chat 输入框蓝色边框
- [x] AppStyle.Width() 全屏宽度对齐
- [x] AskUser 模态框 lipgloss 美化（替代 ASCII 手绘框）
- [x] 帮助栏完善


### 10.4 Phase 4: 交互式对话（当前阶段）

**目标**：实现 TUI Chat 屏幕与 dscli chat 进程的完整交互，包括流式输出、多轮对话和 AskUser 支持。

**方案**：dscli chat 以非 JSON-line 模式启动（`dscli chat`），通过 stdin/stdout 进行单轮问答。

- [x] `aiagent.ChatSession` 骨架（stdin/stdout 读写循环）
- [x] 流式输出（byte-by-byte chunk 读取 + chunkFlushDelay 节流）
- [x] Chat 屏幕 UI（气泡渲染、输入框、滚动）
- [x] AskUser 模态框 UI（三种语义：Confirm / Choice / Input）
- [ ] Chat 屏幕完整联调（多轮对话、错误恢复）
- [ ] AskUser 通过 Unix socket 桥接回 TUI（见 §12）


### 10.5 Phase 5: 剩余屏幕 + 完善
- [ ] 移植剩余屏幕（Project / Role / Tool / Mail / Service / Flycheck 详情视图）
- [ ] 错误处理完善（版本兼容性检测、友好提示）
- [ ] 集成测试
- [ ] 性能优化
## 11. 边界情况

| 场景 | 处理方式 |
|------|----------|
| dscli 不在 PATH 中 | 启动时报错 + 提示配置 `DSCLI_PATH` |
| `dscli version` 执行失败 | 视为 dscli 未安装，报错 |
| ChatSession 中断 | 返回 `EventError`，TUI 显示错误提示 |
| ask_user（Socket 路径）| dscli 启动 `dscli-tui client` → Unix socket → TUI 模态框 → 写回文件 |
| Socket 文件不存在 | 客户端退出码非零，dscli 认为编辑器失败，ask_user 返回 error |
| 用户按 Esc 取消 | Socket 返回空内容，dscli 读回空字符串作为工具结果 |
| Socket 超时（300s） | 服务端关闭连接，客户端写空内容退出，ask_user 返回 error |
| ask_user 空回复 | 用户未输入内容但按 Enter → 空字符串作为合法工具结果 |
| 多个 ask_user 同时 | 当前设计不支持——一轮只有一个 ask_user（串行执行） |
| dscli 版本不兼容 | 接口方法兼容：增加方法不破坏旧版本 |
| 终端尺寸过小 | TUI 已有最小尺寸保护逻辑 |


## 12. AskUser 通过 Unix Socket 桥接回 TUI

### 12.1 背景与动机

dscli 的 `ask_user` 工具调用通过 `$EDITOR` 环境变量打开外部编辑器：
1. 创建临时文件（如 `/tmp/dscli_editor_xxx.md`）
2. 将问题内容写入文件
3. 启动 `$EDITOR <file>`，等待编辑器退出
4. 读取编辑后的文件内容作为用户回复

在 TUI 环境中不能直接打开终端编辑器（会破坏 Bubble Tea 的渲染）。本方案利用 `$EDITOR` 机制，将编辑器替换为一个 **Unix socket 客户端**，将编辑操作桥接回 TUI 的模态框。

### 12.2 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│ dscli.tui (Bubble Tea)                                      │
│                                                             │
│  ┌──────────┐  ┌──────────────┐  ┌────────────────────┐    │
│  │  TUI      │  │  Socket      │  │  dscli chat        │    │
│  │  Model    │◄─┤  Service     │  │  (子进程)           │    │
│  │           │  │  (goroutine) │  │                    │    │
│  │  Screen   │  │              │  │  $EDITOR =         │    │
│  │  AskUser  │  │  .dscli/     │  │  dscli-tui client  │    │
│  │           │  │  dscli-tui   │  │                    │    │
│  │           │  │  .sock       │  │  ask_user → exec   │    │
│  └──────────┘  └──────┬───────┘  │  dscli-tui client   │    │
│                        │         │  /tmp/xxx.md        │    │
│                        │         └────────┬───────────┘    │
│                        │  Unix socket     │                │
│                        └──────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

### 12.3 交互序列

```
dscli chat                        TUI Socket Service
    │                                    │
    │  (ask_user 工具调用)                │
    │                                    │
    │  1. 创建临时文件                    │
    │     /tmp/dscli_editor_xxx.md       │
    │     内容 = 问题内容                 │
    │                                    │
    │  2. exec dscli-tui client          │
    │     /tmp/dscli_editor_xxx.md       │
    │     (因为 EDITOR=dscli-tui client) │
    │                                    │
    │  ┌─ 客户端 (子进程) ──────────┐    │
    │  │ 3. 读取文件内容            │    │
    │  │ 4. 连接 Unix socket        │    │
    │  │ 5. 发送 request            │────│──── 6. 收到消息
    │  │    {question, file}        │    │     发送 SocketAskUserMsg
    │  │                            │    │     等待响应
    │  │                            │    │
    │  │                            │    │     ┌─ TUI ───────────┐
    │  │                            │    │     │ 7. ScreenAskUser │
    │  │                            │    │     │    模态框        │
    │  │                            │    │     │ 8. 用户输入      │
    │  │                            │    │     │ 9. Enter → 响应  │
    │  │                            │    │     └─────────────────┘
    │  │                            │◄───│──── 10. 返回 response
    │  │ 11. 追加响应到文件         │    │
    │  │ 12. 退出（exit 0）         │    │
    │  │ ──────────────────────────┘    │
    │                                    │
    │  13. dscli 读取文件内容            │
    │     作为 ask_user 结果             │
    │                                    │
    │  14. 继续 ChatRound                │
```

### 12.4 Socket 协议

#### Socket 文件位置

```
<project-root>/.dscli/dscli-tui.sock
```

通过 `findGitRoot(cwd)` 或 `os.Getwd()` 确定项目根目录（与 dscli 的 `GetProjectRoot` 逻辑一致）。

#### Request 格式（纯文本，两行）

发送方（socket client，即 EDITOR 子进程）按以下格式写入连接：

```text
<question 文本>
<file path>
```

- 第一行：dscli 的 `ask_user` 工具写入临时文件的内容，即待回答的问题
- 第二行：临时文件的绝对路径，用于后续服务端将用户响应写回文件

两行均以 `\n` 结尾。服务端使用 `bufio.Scanner` 或类似方式读取两行。

#### Response 格式（纯文本，剩余所有内容）

服务端将用户通过 TUI 模态框输入的内容原样写回连接（可能多行），客户端在连接关闭后读取全部内容：

```text
<用户输入的响应内容，可能包含多行>
```

服务端写入完成后关闭连接。客户端通过 `io.ReadAll(conn)` 或 `bufio.Reader` 读取全部内容。


### 12.5 包结构

```
internal/
├── socket/                    # 新增包：Unix socket 通信
│   ├── service.go            # Socket 服务端（在 TUI 启动时运行）
│   │   ├── Start(ctx, projectRoot) → listener
│   │   │   - 创建 .dscli/ 目录（如不存在）
│   │   │   - 清理旧 socket 文件（如有），防上次崩溃残留
│   │   │   - 监听 Unix socket
│   │   │   - 接收连接 → 按行读取两行纯文本（question + file path）→ 发送到 channel
│   │   │   - 等待 response → 将内容写回连接（纯文本，可能多行）
│   │   └── Stop() → cleanup
│   │       - 关闭 listener
│   │       - 删除 socket 文件
│   │
│   └── client.go             # Socket 客户端（作为 EDITOR 子进程）
│       └── Run(args []string) exitCode
│           - 解析参数（文件名）
│           - 验证文件存在
│           - 读取文件内容
│           - 连接 socket
│           - 发送 request（两行纯文本：question + file path）
│           - 接收 response（纯文本，读至连接关闭）
│           - 追加 response 到文件
│           - 退出
│
└── tui/
    ├── model.go              # 增加 socket 相关字段
    ├── update.go             # 处理 SocketAskUserMsg
    └── commands.go           # cmdStartSocketService / cmdWaitSocketResponse
```

### 12.6 TUI Model 新增字段

```go
type RootModel struct {
    // ... 现有字段 ...

    // ── Socket Service ────────────────────────────────────────
    socketService  *socket.Service   // nil 表示未启动
    socketListener net.Listener      // Unix socket 监听器

    // ── Socket AskUser ────────────────────────────────────────
    socketAskReq   *socket.AskRequest  // 当前待处理的请求
    socketAskResp  chan string         // 用于接收用户响应的 channel
}
```

### 12.7 Update 路由变化

```go
// 新增消息类型
type socketAskUserMsg struct {
    Request  *socket.AskRequest
    Respond  chan<- string  // 用于写回响应
}

// Update 中处理
func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case socketAskUserMsg:
        // 保存 request 和 respond channel
        m.socketAskReq = msg.Request
        m.socketAskResp = msg.Respond
        // 设置 askQuestion 供模态框显示
        m.askQuestion = msg.Request.Question
        m.askInput.SetValue("")
        m.askInput.Focus()
        m.prevScreen = m.screen  // 保存当前屏幕
        m.screen = ScreenAskUser
        return m, nil
    }
}

// resumeFromAskUser 中检测 socket 路径
func (m *RootModel) resumeFromAskUser() (tea.Model, tea.Cmd) {
    if m.socketAskResp != nil && m.askResponse != nil {
        // 通过 socket 返回响应
        m.socketAskResp <- m.askResponse.Value
        m.socketAskResp = nil
        m.socketAskReq = nil
        screen := m.prevScreen
        m.prevScreen = ScreenMainMenu
        m.screen = screen
        return m, nil
    }
    // ... 原有逻辑 ...
}
```

### 12.8 ChatSession 环境变量

在 `cmdStartChat` 中设置 `EDITOR` 环境变量：

```go
func cmdStartChat(agent aiagent.AIAgent, history []ChatLine) tea.Cmd {
    return func() tea.Msg {
        opts := aiagent.ChatSessionOptions{
            Model: "deepseek-chat",
            // 传递 EDITOR 环境变量给 dscli 子进程
            Env: []string{
                "EDITOR=dscli-tui client",
            },
        }
        session, err := agent.NewChatSession(context.Background(), opts)
        // ...
    }
}
```

对应的 `ChatSessionOptions` 和 `NewChatSession` 增加 env 支持：

```go
type ChatSessionOptions struct {
    Model      string
    Role       string
    HistSize   int
    DscliPath  string
    ProjectDir string
    Env        []string // 额外环境变量，传递给 dscli 子进程
}
```

在 `NewChatSession` 中将 `opts.Env` 应用到 `cmd.Env`。

### 12.9 Socket 客户端作为子命令

main.go 中注册：

```go
func main() {
    if len(os.Args) >= 2 && os.Args[1] == "client" {
        // Socket 客户端模式
        os.Exit(socket.RunClient(os.Args[2:]))
    }
    // 正常启动 TUI
    // ... 启动 Socket Service goroutine ...
}
```

`socket.RunClient` 实现要点：

```go
func RunClient(args []string) int {
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "usage: dscli-tui client <file>")
        return 1
    }
    filePath := args[0]

    // 1. 验证文件存在
    content, err := os.ReadFile(filePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", filePath, err)
        return 1
    }

    // 2. 查找 socket 文件
    socketPath := findSocketPath() // 根据 CWD 查找 .dscli/dscli-tui.sock
    if socketPath == "" {
        fmt.Fprintln(os.Stderr, "error: dscli-tui service not running (socket not found)")
        return 1
    }

    // 3. 连接 socket
    conn, err := net.Dial("unix", socketPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: cannot connect to dscli-tui: %v\n", err)
        return 1
    }
    defer conn.Close()

    // 4. 发送请求（纯文本两行）
    question := strings.TrimSpace(string(content))
    fmt.Fprintf(conn, "%s\n%s\n", question, filePath)

    // 5. 接收响应（纯文本，读至连接关闭）
    respBytes, _ := io.ReadAll(conn)
    respContent := string(respBytes)

    // 6. 追加响应到文件
    f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return 1
    }
    defer f.Close()
    f.WriteString("\n" + respContent)

    return 0
}
```

### 12.10 查找 Socket 路径

客户端需要从当前工作目录向上查找 `.dscli/dscli-tui.sock`：

```go
func findSocketPath() string {
    cwd, _ := os.Getwd()
    dir := cwd
    for {
        socketPath := filepath.Join(dir, ".dscli", "dscli-tui.sock")
        if _, err := os.Stat(socketPath); err == nil {
            return socketPath
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            break
        }
        dir = parent
    }
    return ""
}
```

### 12.11 边界情况

| 场景 | 处理方式 |
|------|----------|
| Socket 文件不存在 | 客户端报错退出，dscli 认为编辑器失败，ask_user 返回 error |
| Socket 文件残留（上次崩溃） | Start 中 `os.Remove(socketPath)` 在 Listen 前清理，确保重启后正常绑定 |
| Socket 服务未启动 | 客户端连接失败，同上 |
| 用户按 Esc 取消 | 追加空内容到文件，dscli 读回空字符串 |
| Socket 超时 | 服务端设置读取超时（如 300s），超时后返回空响应 |
| 多个 ask_user 同时 | 设计上不支持——串行执行，同一时间只有一个待处理请求 |
| 客户端崩溃 | dscli 的 cmd.Wait() 检测到非零退出，ask_user 返回 error |

## 13. 实施阶段

### 13.1 Phase 6: AskUser Socket 桥接 ✓

- [x] 新建 `internal/socket/` 包（service.go + client.go）
- [x] 实现 `socket.Service`：启动 Unix socket 监听（含 stale socket 清理）、请求/响应循环
- [x] 实现 `socket.Client`：连接 socket、发送/接收、写回文件
- [x] 实现 `findSocketPath()` 向上查找 socket 文件
- [x] main.go 增加 `client` 子命令路由
- [x] `ChatSessionOptions` 增加 `Env []string` 字段
- [x] `exec.go:NewChatSession` 应用中 `opts.Env` 到 `cmd.Env`
- [x] `commands.go:cmdStartChat` 设置 `EDITOR=dscli-tui client`
- [x] Model 增加 socket 相关字段
- [x] update.go 处理 `SocketAskUserMsg`
- [x] `resumeFromAskUser` 增加 socket 响应路径
- [x] TUI 启动时启动 Socket Service goroutine
- [x] TUI 退出时关闭 Socket Service（删除 socket 文件）
- [x] Socket 包单元测试（3 个测试：Start/Stop、请求/响应、stale socket 清理）
- [ ] 手动验证 ask_user 从 dscli chat → socket → TUI 模态框 → 返回的全链路
