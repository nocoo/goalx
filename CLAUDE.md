# CLAUDE.md

## What Is This

**AutoResearch** — 精巧的自主研究框架（Go）。配合 AI Coding Agent（Claude Code / Codex / Factory Droid / Aider）实现一键启动的自动化开发、测试、研究、优化。

灵感来源：[karpathy/autoresearch](https://github.com/karpathy/autoresearch) 的自主实验循环 + [lidangzzz/goal-driven](https://github.com/lidangzzz/goal-driven) 的目标驱动持久性。

## Principle

个人开发工具，唯一用户是作者本人。不写兼容性代码，不留技术债务，不做无意义防御。

### 核心理念

| 原则 | 含义 |
|------|------|
| 约束即自由 | 框架只做 AI agent 不擅长的事（init/journal/tmux/report），AI agent 做所有重活 |
| 协议即引擎 | program.md 是真正的灵魂，不是 Go 代码。把正确的指令给正确的执行者 |
| 黑盒目标 | 被研究的系统是黑盒，通过文件/命令/API 交互，零耦合 |
| 精巧不复杂 | ~1000 行 Go + 声明式配置/模板，不多写一行 |

### 禁止事项

- **不自己写 LLM 调用**：AI coding agent 本身就是 LLM engine
- **不自己写文件编辑逻辑**：AI coding agent 天然具备
- **不引入复杂抽象**：接口小而专，没有框架膨胀
- **不静默回退**：harness 崩溃 = 记录 crash + 下一轮，不吞错误
- **不留 TODO/FIXME/HACK**：每次提交即最优实现

## Architecture

### 两层设计

```
┌───────────────────────────────────────────────┐
│  Level 1: CLI 开发工具 (ar)                    │
│  ──────────────────────                        │
│  配合 Claude Code / Codex，从外部对任何项目     │
│  做自主研究。tmux 编排，config 驱动，一键启动。  │
│                                                │
│  ar start → init + tmux + AI agent             │
│  ar status / ar attach / ar stop / ar report   │
└───────────────────────────────────────────────┘

┌───────────────────────────────────────────────┐
│  Level 2: 可嵌入 SDK                           │
│  ──────────────────────                        │
│  import 到任何 Go 项目，在进程内跑自主研究循环。 │
│  宿主系统提供 Artifact/Executor/Proposer 实现。  │
│                                                │
│  Engine → Propose → Execute → Measure → Decide │
└───────────────────────────────────────────────┘

┌───────────────────────────────────────────────┐
│  Shared Core                                   │
│  ──────────────────────                        │
│  Experiment / Metric / Journal / Policy        │
│  两层共享，统一数据模型                         │
└───────────────────────────────────────────────┘
```

### 数据流

```
Level 1 (外部研究):
  config.yaml → ar start → tmux session → AI agent 读 program.md → 自主循环
  AI agent: 改文件 → 跑命令 → 看结果 → keep/revert → 记 journal → repeat

Level 2 (内部研究):
  宿主系统 → Engine.Run() → Proposer.Propose() → Executor.Execute() → Policy.Decide()
  → Journal.Record() → repeat
```

## Package Layout

```
autoresearch/
├── go.mod                     # module github.com/vonbai/autoresearch
│
│  # ── Shared Core ──────────────
├── experiment.go              # Experiment / ExperimentResult
├── metric.go                  # Metric / MetricSet / Direction
├── journal.go                 # Journal (TSV read/write/query)
├── policy.go                  # Policy: Simple / Pareto / Threshold
├── config.go                  # ar.yaml 解析
│
│  # ── Level 2: SDK ─────────────
├── engine.go                  # In-process research loop
├── artifact.go                # Artifact interface
├── executor.go                # Executor interface
├── proposer.go                # Proposer interface
├── session.go                 # Session state machine
├── report.go                  # Report generation
│
│  # ── Level 1: CLI ─────────────
├── cmd/ar/main.go             # CLI entry: ar start|stop|status|attach|report
├── cli/
│   ├── start.go               # init + tmux + launch
│   ├── stop.go                # graceful shutdown
│   ├── status.go              # journal progress display
│   ├── attach.go              # tmux attach
│   ├── report.go              # markdown report generation
│   ├── protocol.go            # program.md template rendering
│   ├── harness.go             # harness YAML parsing
│   ├── tmux.go                # tmux session/window management
│   └── worktree.go            # git worktree create/cleanup
│
│  # ── 声明式配置 ────────────────
├── harnesses/                 # 内置 harness 定义
│   ├── go-test.yaml
│   ├── go-bench.yaml
│   ├── command.yaml
│   └── http-health.yaml
├── adapters/                  # AI agent 适配器模板
│   ├── claude-code.md.tmpl
│   └── codex.md.tmpl
└── templates/
    ├── program.md.tmpl        # 核心协议模板（框架灵魂）
    └── report.md.tmpl         # 报告模板
```

## Key Abstractions

### Shared Core

```
Experiment:  一次实验（hypothesis + changes + result + verdict）
Metric:      命名数值 + 方向（minimize/maximize）+ 可选约束
MetricSet:   一组指标（primary + secondary + constraints）
Journal:     append-only TSV 实验日志
Policy:      keep/revert 决策规则（Simple/Pareto/Threshold）
Verdict:     keep | revert | crash | constraint_violated
```

### Level 2 SDK Interfaces

```
Artifact:    被修改的对象 → Snapshot() / Restore() / Describe()
Executor:    实验执行器 → Execute(artifact, budget) → ExperimentResult
Proposer:    假设生成器 → Propose(journal, artifact) → Proposal
Engine:      主循环编排 → propose → execute → measure → decide → record
Session:     状态机 → INIT → BASELINE → RUNNING → CONVERGED/STOPPED
```

### Level 1 CLI Concepts

```
Config:      ar.yaml — 声明式研究定义（单会话或多会话）
Harness:     实验执行方式 — command + timeout + metric extraction patterns
Engine:      AI agent 类型 — claude-code / codex / aider / custom
Adapter:     把 program.md 翻译成 AI agent 能理解的格式
```

## Config Examples

### 单会话

```yaml
objective: "优化动量因子IC"
engine: claude-code
harness:
  command: "go test -run TestFactorEval -v -count=1 ./..."
  timeout: 3m
  metrics:
    primary: { name: ic_mean, pattern: '^ic_mean:\s+([\d.]+)', direction: maximize }
target:
  files: ["factors/momentum.yaml"]
  readonly: ["factors/eval/"]
budget:
  max_rounds: 100
  convergence: 10
  target: "ic_mean >= 0.08"
```

### 多会话并行

```yaml
sessions:
  - name: factor-ic
    objective: "优化动量因子IC"
    engine: claude-code
    harness: { ... }
  - name: scheduler-perf
    objective: "优化scheduler吞吐量"
    engine: codex
    harness: { ... }
```

## Commands

```bash
go build ./...                    # build
go test ./... -v                  # test
go vet ./...                      # lint
go build -o bin/ar ./cmd/ar       # build CLI binary
```

## Conventions

- **Go 100%**
- Commits: `feat(core|sdk|cli):` / `fix|test|docs|refactor:`
- 文件大小 ≤ 500 行
- 零技术债务：不留 TODO/FIXME/HACK

## Integration: QuantOS (第一个集成项目)

QuantOS 通过两种方式使用 autoresearch：

1. **Level 1 外部**: 开发者用 `ar start` 从外部研究 QuantOS（改代码、跑测试、看结果）
2. **Level 2 内部**: QuantOS 的 `quant_agent/autoresearch/` bridge 导入 SDK，让 Agent 在进程内做因子/策略研究

```
/data/dev/quantos/quant_agent/autoresearch/
  ├── bridge.go              # SDK → Agent 桥接
  ├── factor_artifact.go     # Artifact 实现
  ├── backtest_executor.go   # Executor 实现
  └── agent_proposer.go      # Proposer 实现（用 Agent 的 LLM）
```
