# AutoResearch — Design Spec

> 精巧的自主研究框架。配合 AI Coding Agent 实现一键启动的自动化开发、测试、研究、优化。

## 1. Problem Statement

开发者在优化代码、调参、研究方案时，核心循环是 **改 → 跑 → 看 → 判断 → 重复**。这个循环：
- 人工执行低效（一天做不了几轮）
- AI coding agent（Claude Code, Codex, Factory Droid）天然具备执行能力
- 但缺少**结构化的协议**来驱动自主循环

Karpathy 的 autoresearch 证明了：一个精巧的 `program.md` + 固定评估 harness + 实验日志，
就能让 AI agent 整夜跑 100+ 实验。

AutoResearch 将这个理念泛化为通用框架，适配任何项目、任何 AI agent。

## 2. Design Philosophy

| 原则 | 含义 |
|------|------|
| **约束即自由** | 框架只做 AI agent 不擅长的事，AI agent 做所有重活 |
| **协议即引擎** | `program.md` 是灵魂，Go 代码是脚手架 |
| **黑盒目标** | 通过文件/命令/API 交互，零耦合 |
| **精巧不复杂** | ~1200 行 Go + 声明式配置 |
| **一键启动** | `ar start` 完成 init + tmux/API + AI agent 启动 |

### 不做什么

- 不自己调 LLM — AI coding agent 本身就是 LLM engine
- 不自己写文件编辑/命令执行 — AI agent 天然具备
- 不引入复杂依赖 — 标准库 + YAML 解析，完事

## 3. Architecture

### 3.1 两层设计

```
┌───────────────────────────────────────────────────────┐
│  Level 1: CLI 开发工具 (ar)                            │
│                                                        │
│  面向开发者。配合 AI coding agent 从外部对任何项目       │
│  做自主研究/优化/升级。                                  │
│                                                        │
│  config.yaml → ar start → tmux/API → AI agent 自主循环  │
│  ar status / ar attach / ar stop / ar report            │
└───────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────┐
│  Level 2: 可嵌入 SDK                                   │
│                                                        │
│  面向系统集成。import 到 Go 项目，在进程内跑研究循环。    │
│  宿主系统提供 Artifact / Executor / Proposer 实现。      │
│                                                        │
│  Engine.Run() → Propose → Execute → Measure → Decide   │
└───────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────┐
│  Shared Core                                           │
│  Experiment / Metric / Journal / Policy / Config        │
└───────────────────────────────────────────────────────┘
```

### 3.2 Level 1 数据流

```
ar start
  ├─ 解析 ar.yaml (config)
  ├─ git worktree create (隔离实验)
  ├─ 渲染 program.md (从 config + 模板)
  ├─ 初始化 journal.tsv
  ├─ 生成 adapter 指令 (CLAUDE.md / AGENTS.md / API payload)
  ├─ 启动 engine:
  │   ├─ interactive: tmux new-session → send-keys "claude ..."
  │   └─ remote: git push → API trigger (Factory Droid)
  └─ 输出状态

AI agent 自主循环:
  读 program.md → 改 target → git commit → 跑 harness → 提取 metrics
  → 记 journal → keep/revert → repeat
```

### 3.3 Level 2 数据流

```
宿主系统调用:
  engine := ar.NewEngine(opts)
  result := engine.Run(ctx)

Engine 内部:
  baseline := executor.Execute(artifact)
  loop:
    proposal := proposer.Propose(journal, artifact)
    artifact.Apply(proposal)
    result := executor.Execute(artifact)
    verdict := policy.Decide(baseline, best, result)
    journal.Record(experiment)
    if verdict == Keep: best = result
    else: artifact.Restore(snapshot)
```

## 4. Core Abstractions

### 4.1 Experiment

```go
type Experiment struct {
    Round       int
    Timestamp   time.Time
    Commit      string       // git short hash
    Hypothesis  string       // 人类可读假设描述
    Metrics     MetricSet    // 实验结果指标
    Verdict     Verdict      // keep | revert | crash | constraint_violated
    Description string       // 简要描述
}
```

### 4.2 Metric

```go
type Direction int
const (
    Minimize Direction = iota
    Maximize
)

type Metric struct {
    Name      string
    Value     float64
    Direction Direction
}

type MetricSet struct {
    Primary     Metric
    Secondary   []Metric
    Constraints []ConstraintMetric
}

type ConstraintMetric struct {
    Metric
    Op        string  // "<=", ">=", "==", "<", ">"
    Threshold float64
}

func (ms MetricSet) Satisfies() bool // 所有 constraints 是否满足
func (ms MetricSet) BetterThan(other MetricSet) bool // primary 是否改进
```

### 4.3 Journal

TSV 格式，append-only，和 Karpathy 的 `results.tsv` 一脉相承。

```
commit	ic_mean	max_drawdown	status	description
a1b2c3d	0.062	0.15	keep	baseline
b2c3d4e	0.071	0.14	keep	add 20-day lookback window
c3d4e5f	0.068	0.22	revert	switch to exponential weighting
d4e5f6g	0.000	0.00	crash	divide by zero in normalization
```

```go
type Journal struct {
    Path    string
    Columns []string   // 从 header 解析
    Entries []Experiment
}

func LoadJournal(path string) (*Journal, error)
func (j *Journal) Append(e Experiment) error      // append-only
func (j *Journal) Best() *Experiment               // 最优 kept entry
func (j *Journal) Kept() []Experiment               // 所有 kept entries
func (j *Journal) Summary() JournalSummary          // 统计摘要
func (j *Journal) RecentN(n int) []Experiment        // 最近 N 轮（给 LLM context）
```

### 4.4 Policy

```go
type Policy interface {
    Decide(baseline, best, current *ExperimentResult) Verdict
}

// SimplePolicy: primary 改进就 keep
type SimplePolicy struct{}

// ThresholdPolicy: 改进超过 minDelta 才 keep
type ThresholdPolicy struct {
    MinDelta float64
}

// ParetoPolicy: 多指标帕累托支配才 keep
type ParetoPolicy struct {
    Objectives []string // metric names to optimize
}
```

### 4.5 Budget

```go
type Budget struct {
    MaxRounds   int            // 0 = 无限
    MaxDuration time.Duration  // 0 = 无限
    Convergence int            // 连续 N 轮无改进则停 (0 = 不检测)
    Target      string         // goal-driven 式达标停止: "ic_mean >= 0.08"
}

func (b Budget) ShouldStop(journal *Journal, elapsed time.Duration) (bool, string)
```

## 5. Config Format

### 5.1 单会话

```yaml
# ar.yaml
objective: "优化动量因子在A股的IC"
engine: claude-code

harness:
  command: "go test -run TestFactorEval -v -count=1 ./..."
  timeout: 3m
  metrics:
    primary:
      name: ic_mean
      pattern: '^ic_mean:\s+([\d.]+)'
      direction: maximize
    secondary:
      - name: ic_ir
        pattern: '^ic_ir:\s+([\d.]+)'
    constraints:
      - name: max_drawdown
        pattern: '^max_dd:\s+([\d.]+)'
        op: "<="
        threshold: 0.20

target:
  files:
    - "quant_pipeline/factor/momentum.yaml"
  readonly:
    - "quant_pipeline/factor/eval/"

budget:
  max_rounds: 100
  convergence: 10
  target: "ic_mean >= 0.08"
```

### 5.2 多会话并行

```yaml
# ar.yaml
sessions:
  - name: factor-ic
    objective: "优化动量因子IC"
    engine: claude-code
    harness:
      command: "go test -run TestFactorEval -v -count=1 ./..."
      timeout: 3m
      metrics:
        primary: { name: ic_mean, pattern: '^ic_mean:\s+([\d.]+)', direction: maximize }
    target:
      files: ["factors/momentum.yaml"]
    budget:
      convergence: 10

  - name: scheduler-perf
    objective: "优化scheduler吞吐量"
    engine: codex
    harness:
      command: "go test -bench=. ./quant_mono/scheduler/..."
      timeout: 5m
      metrics:
        primary: { name: ns_per_op, pattern: 'Benchmark\w+\s+\d+\s+([\d.]+)\s+ns/op', direction: minimize }
    target:
      files: ["quant_mono/scheduler/*.go"]

  - name: stability-check
    objective: "提升系统启动稳定性"
    engine: factory-droid
    harness:
      command: "go build ./cmd/quantos && timeout 30 ./bin/quantos serve"
      timeout: 1m
      metrics:
        primary: { name: boot_ok, pattern: '"status":"ok"', direction: match }
    target:
      files: ["configs/config.yaml"]
```

### 5.3 Harness 引用

```yaml
# 可以引用内置 harness 模板
harness: go-bench
harness_vars:
  BenchPattern: "BenchmarkScheduler"
  BenchTime: "10s"
  TestPath: "./quant_mono/scheduler/..."
```

## 6. Engine Adapters

### 6.1 抽象

所有引擎都是本地 CLI 工具，在 tmux 内运行。统一模型，无需区分 interactive/remote。

```go
type EngineConfig struct {
    Name    string   // claude-code | codex | factory-droid | aider | custom
    Command string   // CLI 命令: claude | codex | droid | aider
    Args    []string // 默认参数
    Prompt  string   // 启动 prompt 模板（{program} 占位符）
    Adapter string   // adapter 模板名称
}
```

### 6.2 内置 engines

```yaml
# ~/.config/ar/engines.yaml
engines:
  claude-code:
    command: claude
    prompt: "Read {program} and follow it exactly. Begin immediately. Never stop."
    adapter: claude-code

  codex:
    command: codex
    prompt: "Follow {program} autonomously."
    adapter: codex

  factory-droid:
    command: droid
    args: ["exec", "--auto", "high", "-f"]
    prompt: "{program}"
    adapter: factory-droid

  aider:
    command: aider
    args: ["--no-auto-commits", "--yes"]
    prompt: "/read {program}"
    adapter: generic
```

### 6.3 统一执行模型

所有引擎都是本地 CLI，统一通过 tmux 管理：

| Engine | tmux 内启动命令 | 指令文件 |
|--------|----------------|----------|
| **Claude Code** | `claude -p "Read {program} ..."` | CLAUDE.md |
| **Codex** | `codex -q "Follow {program} ..."` | AGENTS.md |
| **Factory Droid** | `droid exec --auto high -f {program}` | program.md 直传 |
| **Aider** | `aider --yes` + `/read {program}` | aider prompt |

统一生命周期：
- `ar start` → tmux new-session → send-keys "{engine command}"
- `ar attach` → tmux attach-session
- `ar stop` → tmux send-keys C-c (graceful) 或 kill-session
- `ar status` → 读 journal.tsv（所有引擎的产出格式一致）

### 6.4 Factory Droid 适配细节

Factory Droid 是本地 CLI（`droid` 命令），通过 `droid exec` 执行：

```bash
# 安装
curl -fsSL https://app.factory.ai/cli | sh

# autoresearch 启动方式
droid exec --auto high --cwd {worktree_path} -f .autoresearch/program.md
```

关键 flag：
- `--auto high`：允许修改文件和执行命令（autoresearch 必需）
- `-f program.md`：从文件读取研究协议
- `--cwd`：指定工作目录（worktree）
- `--session-id`：可选，继续已有会话

## 7. tmux Orchestration

### 7.1 单会话

```go
func launchInteractive(cfg SessionConfig, workdir string) error {
    sessionName := fmt.Sprintf("ar-%s", cfg.Name)
    // tmux new-session -d -s {sessionName} -c {workdir}
    // tmux send-keys -t {sessionName} "{engine.Command} {engine.Prompt}" Enter
}
```

### 7.2 多会话

```go
func launchMulti(sessions []SessionConfig) error {
    tmuxSession := "autoresearch"
    // tmux new-session -d -s autoresearch -c {workdir[0]}
    for i, s := range sessions {
        if i > 0 {
            // tmux new-window -t autoresearch -n {s.Name} -c {workdir[i]}
        } else {
            // tmux rename-window -t autoresearch:0 {s.Name}
        }
        // tmux send-keys -t autoresearch:{s.Name} "{cmd}" Enter
    }
}
```

每个 session 用独立 git worktree，互不干扰。

### 7.3 ar attach

```bash
ar attach                    # attach 到 autoresearch tmux session
ar attach factor-ic          # attach 到指定 window
```

实现：`tmux attach-session -t autoresearch` / `tmux select-window -t autoresearch:{name}`

## 8. SDK Interfaces (Level 2)

### 8.1 Artifact

```go
type Artifact interface {
    ID() string
    Kind() string
    Snapshot() ([]byte, error)
    Restore(snapshot []byte) error
    Describe() string                      // 给 Proposer 的 context
    Apply(patch Patch) error               // 应用修改
}
```

### 8.2 Executor

```go
type Executor interface {
    Execute(ctx context.Context, artifact Artifact, budget time.Duration) (*ExperimentResult, error)
}

type ExperimentResult struct {
    Metrics  MetricSet
    Duration time.Duration
    Crashed  bool
    CrashLog string
}
```

### 8.3 Proposer

```go
type Proposer interface {
    Propose(ctx context.Context, state ProposerState) (*Proposal, error)
}

type ProposerState struct {
    Objective string
    Artifact  string       // Artifact.Describe() output
    Journal   []Experiment // 历史实验
    Best      *Experiment
    StuckFor  int          // 连续未改进轮次（用于 stuck detection）
}

type Proposal struct {
    Hypothesis string   // 为什么这个改变可能有效
    Patch      Patch    // 对 Artifact 的修改
}

type Patch struct {
    Kind    string // "replace" | "diff" | "structured"
    Content []byte
}
```

### 8.4 Engine

```go
type Engine struct {
    artifact Artifact
    executor Executor
    proposer Proposer
    policy   Policy
    budget   Budget
    journal  *Journal
}

type EngineOpts struct {
    Artifact Artifact
    Executor Executor
    Proposer Proposer
    Policy   Policy
    Budget   Budget
    Journal  *Journal   // nil → create in-memory
}

func NewEngine(opts EngineOpts) *Engine

type RunResult struct {
    Journal     *Journal
    Best        *Experiment
    TotalRounds int
    StopReason  string
}

func (e *Engine) Run(ctx context.Context) (*RunResult, error)
```

### 8.5 Engine 主循环

```go
func (e *Engine) Run(ctx context.Context) (*RunResult, error) {
    // 1. Baseline
    snapshot := e.artifact.Snapshot()
    baseline := e.executor.Execute(ctx, e.artifact, ...)
    e.journal.Record(baselineExperiment)

    // 2. Loop
    for round := 1; ; round++ {
        if stop, reason := e.budget.ShouldStop(e.journal, elapsed); stop {
            return result(reason)
        }

        // Propose
        state := ProposerState{Journal: e.journal.RecentN(20), Best: best, StuckFor: stuckCount}
        proposal := e.proposer.Propose(ctx, state)

        // Snapshot + Apply
        snap := e.artifact.Snapshot()
        e.artifact.Apply(proposal.Patch)

        // Execute
        result := e.executor.Execute(ctx, e.artifact, budget)

        // Decide
        verdict := e.policy.Decide(baseline, best, result)
        if !result.Metrics.Satisfies() {
            verdict = ConstraintViolated
        }

        // Record
        e.journal.Record(experiment{Round: round, Verdict: verdict, ...})

        // Keep or Revert
        if verdict == Keep {
            best = result
            stuckCount = 0
        } else {
            e.artifact.Restore(snap)
            stuckCount++
        }
    }
}
```

## 9. Git Worktree Strategy

每个研究会话使用独立 git worktree：

```
ar start --name momentum
  → git worktree add .autoresearch/worktrees/momentum -b autoresearch/momentum
  → AI agent 在 worktree 内工作
  → 主仓库不受影响

ar close momentum
  → git worktree remove .autoresearch/worktrees/momentum
  → 可选: merge 或 discard branch
```

多会话并行时，每个 session 各自的 worktree，各自的 branch，互不冲突。

## 10. Report Generation

`ar report` 从 Journal 生成 Markdown 报告：

```markdown
# AutoResearch Report: 优化动量因子IC

## Summary
| Metric | Baseline | Best | Improvement |
|--------|----------|------|-------------|
| ic_mean | 0.062 | 0.083 | +33.9% |

## Statistics
- Total rounds: 47
- Kept: 8 (17.0%)
- Reverted: 36
- Crashed: 3
- Runtime: 5h 22m

## Kept Experiments (chronological)
| # | Commit | ic_mean | Delta | Description |
|---|--------|---------|-------|-------------|
| 1 | a1b2c3d | 0.062 | — | baseline |
| 5 | e5f6g7h | 0.067 | +8.1% | add 20-day lookback |
| ... | ... | ... | ... | ... |
```

## 11. 全局配置

```yaml
# ~/.config/ar/config.yaml
default_engine: claude-code
journal_format: tsv           # tsv | json
worktree_dir: .autoresearch/worktrees
log_dir: .autoresearch/logs

engines:
  claude-code:
    command: claude
    prompt: "Read {program} and follow it exactly. Begin immediately. Never stop."
    adapter: claude-code
  codex:
    command: codex
    prompt: "Follow {program} autonomously."
    adapter: codex
  factory-droid:
    command: droid
    args: ["exec", "--auto", "high", "-f"]
    prompt: "{program}"
    adapter: factory-droid
  aider:
    command: aider
    args: ["--no-auto-commits", "--yes"]
    prompt: "/read {program}"
    adapter: generic
```

## 12. 代码量估算

| 模块 | 文件 | 估计行数 |
|------|------|---------|
| **Shared Core** | experiment.go, metric.go, journal.go, policy.go, config.go | ~350 |
| **SDK Engine** | engine.go, artifact.go, executor.go, proposer.go, session.go, report.go | ~450 |
| **CLI** | start.go, stop.go, status.go, attach.go, report.go, protocol.go, harness.go, tmux.go, worktree.go | ~450 |
| **CLI entry** | cmd/ar/main.go | ~50 |
| **总 Go 代码** | | **~1300** |
| Templates | program.md.tmpl, report.md.tmpl | ~150 (Markdown) |
| Harnesses | go-test, go-bench, command, http-health | ~80 (YAML) |
| Adapters | claude-code, codex, factory-droid, generic | ~60 (Markdown) |
| **总配置/模板** | | **~290** |

## 13. QuantOS 集成 (第一个目标)

### 13.1 Level 1 (外部研究)

```bash
cd /data/dev/quantos
ar init --objective "优化动量因子IC" --harness go-test --target "factors/*.yaml" --engine claude-code
ar start
# Claude Code 在 tmux 里自主研究 QuantOS 的因子定义
```

### 13.2 Level 2 (内部研究)

```
quantos 仓库新增:
quant_agent/autoresearch/
  ├── bridge.go              # SDK → Agent 模式桥接
  ├── factor_artifact.go     # ar.Artifact 实现: 因子定义
  ├── backtest_executor.go   # ar.Executor 实现: 跑回测
  └── agent_proposer.go      # ar.Proposer 实现: 用 Agent LLM 生成假设
```

QuantOS Agent 在对话中触发 autoresearch 模式 → 调用 SDK Engine → 进程内自主循环。

## 14. 未来扩展

暂不实现，但架构预留：
- **Web Dashboard**: 实时查看 Journal + 指标趋势图
- **Webhook**: 实验完成 / 达标 时通知（Slack / Discord）
- **Cross-session Knowledge**: 多会话间共享洞察
- **Meta-research**: 研究"怎么研究更高效"（优化 propose 策略本身）
