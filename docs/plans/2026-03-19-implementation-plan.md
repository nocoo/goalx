# AutoResearch — Implementation Plan

> Spec: `docs/specs/2026-03-19-autoresearch-design.md`

## Phase 1: Shared Core

最先实现的基础类型，Level 1 和 Level 2 都依赖。

### 1.1 types + verdict (~50 lines)
- `verdict.go`: `Verdict` enum (Keep, Revert, Crash, ConstraintViolated)
- `Direction` enum (Minimize, Maximize)
- **测试**: enum string 方法

### 1.2 metric (~80 lines)
- `metric.go`: `Metric`, `MetricSet`, `ConstraintMetric`
- `MetricSet.BetterThan(other)` — primary 比较
- `MetricSet.Satisfies()` — constraints 检查
- **测试**: 比较逻辑、constraints 满足/违反

### 1.3 experiment (~40 lines)
- `experiment.go`: `Experiment` struct (Round, Timestamp, Commit, Hypothesis, Metrics, Verdict, Description)
- `ExperimentResult` struct (Metrics, Duration, Crashed, CrashLog)
- **测试**: 构造 + 序列化

### 1.4 journal (~120 lines)
- `journal.go`: TSV 读写
- `LoadJournal(path)` — 从 TSV 文件加载
- `Journal.Append(e)` — append-only 写入
- `Journal.Best()` — 最优 kept entry
- `Journal.Kept()` — 所有 kept entries
- `Journal.RecentN(n)` — 最近 N 轮
- `Journal.Summary()` — 统计 (total, kept, reverted, crashed, keep rate)
- **测试**: 写→读 round-trip、Best 查询、空 journal 处理

### 1.5 policy (~60 lines)
- `policy.go`: `Policy` interface + `SimplePolicy` + `ThresholdPolicy`
- `SimplePolicy.Decide()` — primary 改进即 keep
- `ThresholdPolicy.Decide()` — 改进超过 minDelta
- `ParetoPolicy` 可后续加，Phase 1 先不实现
- **测试**: 各 policy 的 keep/revert 判定

### 1.6 config (~100 lines)
- `config.go`: `Config` struct (单会话 + 多会话 sessions)
- `LoadConfig(path)` — 解析 ar.yaml
- 支持单会话 (flat) 和多会话 (sessions array) 两种格式
- harness 模板引用 (`harness: go-bench` + `harness_vars:`)
- **依赖**: `gopkg.in/yaml.v3`
- **测试**: 两种格式解析、harness 引用

**Phase 1 完成标志**: `go test ./...` 全绿，core 类型全部可用。

---

## Phase 2: SDK Engine (Level 2)

可嵌入的进程内研究循环。

### 2.1 interfaces (~60 lines)
- `artifact.go`: `Artifact` interface (ID, Kind, Snapshot, Restore, Describe, Apply)
- `executor.go`: `Executor` interface (Execute)
- `proposer.go`: `Proposer` interface (Propose) + `ProposerState` + `Proposal` + `Patch`
- **测试**: mock 实现验证接口可用

### 2.2 session (~70 lines)
- `session.go`: `Session` struct + 状态机
- States: Init → Baseline → Running → Converged | Stopped
- `Session.Transition(to)` — 状态转换 + 合法性校验
- **测试**: 状态转换合法路径 + 非法路径 panic

### 2.3 budget (~50 lines)
- 放入 `config.go` 或独立 `budget.go`
- `Budget.ShouldStop(journal, elapsed)` — 检查所有停止条件
- 条件: max_rounds / max_duration / convergence / target metric
- `parseTarget("ic_mean >= 0.08")` — 简单表达式解析
- **测试**: 各停止条件触发

### 2.4 engine (~150 lines)
- `engine.go`: `Engine` struct + `NewEngine(opts)` + `Run(ctx) (*RunResult, error)`
- 主循环: baseline → loop{propose → snapshot → apply → execute → decide → record → keep/restore}
- stuck detection: 连续 N 轮未改进时在 ProposerState.StuckFor 传递信号
- context cancellation 支持 (graceful stop)
- **测试**: mock artifact + executor + proposer 跑完整循环
  - case 1: 3 轮，2 keep 1 revert
  - case 2: convergence 触发停止
  - case 3: target 达标停止
  - case 4: ctx cancel 停止

### 2.5 report (~60 lines)
- `report.go`: `GenerateReport(journal, config) (string, error)`
- 从 Journal 生成 markdown 报告
- 使用 `templates/report.md.tmpl`
- **测试**: 验证报告包含 summary table + kept experiments

**Phase 2 完成标志**: Engine 能用 mock 实现跑完整研究循环，journal 正确记录。

---

## Phase 3: CLI (Level 1)

tmux 编排 + protocol 生成 + AI agent 启动。

### 3.1 CLI 框架 (~50 lines)
- `cmd/ar/main.go`: 子命令路由 (start, stop, status, attach, report, close)
- 参数解析用标准库 `flag` (每个子命令一个 FlagSet)
- 不引入第三方 CLI 框架

### 3.2 protocol 渲染 (~60 lines)
- `cli/protocol.go`: `RenderProtocol(config, tmplPath) (string, error)`
- 从 `templates/program.md.tmpl` + config 渲染 program.md
- 模板变量: Objective, Target, Harness, Budget, JournalPath, Baseline
- **测试**: 渲染结果包含关键字段

### 3.3 harness 解析 (~40 lines)
- `cli/harness.go`: `LoadHarness(name) (*HarnessConfig, error)`
- 从 `harnesses/` 目录加载 YAML
- 支持模板变量替换 (harness_vars)
- **测试**: 加载 go-test.yaml + 变量替换

### 3.4 worktree 管理 (~50 lines)
- `cli/worktree.go`: `CreateWorktree(name) (path, error)` / `RemoveWorktree(name)`
- `git worktree add .autoresearch/worktrees/{name} -b autoresearch/{name}`
- 清理: `git worktree remove` + `git branch -D`
- **测试**: 手动验证（依赖 git）

### 3.5 tmux 管理 (~70 lines)
- `cli/tmux.go`:
  - `NewSession(name, workdir)` — `tmux new-session -d -s {name} -c {workdir}`
  - `NewWindow(session, name, workdir)` — 多会话用
  - `SendKeys(session, keys)` — 发送启动命令
  - `AttachSession(session)` — `tmux attach-session -t {session}`
  - `KillSession(session)` — graceful stop
  - `SessionExists(name) bool` — 检查 session 是否存在
- **测试**: 手动验证（依赖 tmux）

### 3.6 ar start (~80 lines)
- `cli/start.go`: 完整启动流程
  1. 加载 config (ar.yaml)
  2. 解析 engine 配置 (~/.config/ar/engines.yaml 或内置默认)
  3. 创建 git worktree
  4. 渲染 program.md → `.autoresearch/program.md`
  5. 初始化 journal.tsv (header only)
  6. 生成 adapter 文件 (CLAUDE.md / AGENTS.md)
  7. 创建 tmux session
  8. send-keys 启动 AI agent
  9. 打印状态
- 多会话: 循环上述步骤，每个 session 一个 tmux window

### 3.7 ar status (~40 lines)
- `cli/status.go`: 读 journal → 格式化输出
- 显示: round count, best metric, keep rate, runtime, status
- 多会话: 所有 session 的表格

### 3.8 ar attach (~15 lines)
- `cli/attach.go`: tmux attach + 可选 window 选择

### 3.9 ar stop (~20 lines)
- `cli/stop.go`: tmux send-keys C-c 或 kill-session

### 3.10 ar report (~30 lines)
- `cli/report.go`: 调用 `GenerateReport()` → 写文件

### 3.11 ar close (~30 lines)
- `cli/close.go`: stop + remove worktree + 可选 delete branch

**Phase 3 完成标志**: `ar start` 能完整创建 worktree + 渲染 protocol + 启动 tmux + Claude Code。

---

## Phase 4: Templates & Adapters

### 4.1 program.md.tmpl 打磨
- 已有初版，根据实际测试调整措辞
- 确保 Karpathy 的 "never stop" 精神完整传达
- 加入 stuck detection 指令

### 4.2 adapter 模板完善
- claude-code.md.tmpl — 已有
- codex.md.tmpl — 已有
- factory-droid.md.tmpl — 已有
- generic.md.tmpl — 通用适配器（给 aider 等）

### 4.3 内置 engines.yaml
- 嵌入到 binary (embed) 作为默认配置
- 用户可覆盖 (~/.config/ar/engines.yaml)

### 4.4 harness 模板完善
- go-test.yaml, go-bench.yaml, command.yaml, http-health.yaml — 已有
- 根据实际使用调整

**Phase 4 完成标志**: 模板渲染出的 program.md 可直接被 Claude Code 执行。

---

## Phase 5: QuantOS 集成测试

### 5.1 Level 1 集成
- 在 `/data/dev/quantos/` 创建 `ar.yaml`
- 选择一个真实目标（如 go-bench harness）
- `ar start` 启动 Claude Code → 验证自主循环可工作

### 5.2 Level 2 集成（QuantOS bridge）
- 在 `/data/dev/quantos/quant_agent/autoresearch/` 创建 bridge
- 实现 Artifact / Executor / Proposer for QuantOS
- Agent 模式触发 SDK Engine
- 此阶段可能在 QuantOS 侧另开 plan

**Phase 5 完成标志**: QuantOS 能通过 Level 1 外部 + Level 2 内部两种方式使用 autoresearch。

---

## 实施顺序总结

```
Phase 1 (Core)       ████████░░░░  ~450 lines   ← 先做，所有后续依赖
Phase 2 (SDK)        ░░░░████░░░░  ~390 lines   ← Core 完成后
Phase 3 (CLI)        ░░░░░░░░████  ~435 lines   ← 可与 Phase 2 并行
Phase 4 (Templates)  ░░░░░░░░░░██  ~150 lines   ← 穿插进行
Phase 5 (QuantOS)    ░░░░░░░░░░░█  集成测试     ← 最后验证
```

Phase 1 → Phase 2 + Phase 3 (并行) → Phase 4 → Phase 5

## 依赖

```
go 1.23+
gopkg.in/yaml.v3    # config/harness YAML 解析
```

仅此。标准库 + 一个 YAML 库。
