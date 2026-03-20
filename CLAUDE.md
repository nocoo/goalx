<coding_guidelines>
# CLAUDE.md

## What Is This

**AutoResearch** — 自主研究框架（Go）。Master/Subagent 架构，配合 AI Coding Agent 实现一键启动的无人值守调研和开发。

灵感来源：
- [karpathy/autoresearch](https://github.com/karpathy/autoresearch)：protocol + journal + 无人值守
- [lidangzzz/goal-driven](https://github.com/lidangzzz/goal-driven)：master/subagent + 持续运行 + AI 判断目标达成

核心区别：框架只做编排（worktree/tmux/journal），所有判断都交给 AI agent。

## Principle

个人开发工具，唯一用户是作者本人。

### 核心理念

| 原则 | 含义 |
|------|------|
| 框架做编排，agent 做判断 | Go 代码管 worktree/tmux/journal，AI 判断目标是否达成 |
| 协议即引擎 | master.md + program.md 是灵魂 |
| 有目标有终点 | master agent 判断目标达成即停 |
| 持久运行 | subagent 偏了 → 引导；崩溃/卡住 → 重启 |
| 并行探索 | 同一目标 N 个 subagent，master 监督 |
| 精巧不复杂 | ~700 行 Go + 声明式配置 + protocol 模板 |

### 禁止事项

- **不自己写 LLM 调用**
- **不自己写 criteria 验证逻辑**：objective 就是 criteria，master agent 理解和判断
- **不引入复杂抽象**
- **不留 TODO/FIXME/HACK**

## Architecture

### Master / Subagent

```
┌──────────────────────────────────────────────────────────┐
│  goalx — CLI                                                 │
│                                                           │
│  goalx start → tmux (1 master + N subagent)                 │
│  goalx status / goalx attach / goalx stop                          │
│  goalx review / goalx keep / goalx archive / goalx drop               │
└──────────────────────────────────────────────────────────┘

tmux session "autoresearch":
  window "master":    Claude opus 交互模式，framework heartbeat 驱动检查循环
  window "session-1": AI subagent 干活（Claude/Codex 交互模式）
  window "session-2": AI subagent 干活（另一条路）
  + heartbeat goroutine: 每 check_interval send-keys → master
```

### 两种模式

| mode | subagent 输出 | 改代码？ |
|------|--------------|---------|
| research | 方案报告 | 不改，代码全 readonly |
| develop | 可工作的代码 | 改 |

### 数据流

```
goalx init "objective" [--research|--develop] [--parallel N]
  → 继承 config → 生成 goalx.yaml

goalx start [或 goalx start "objective" 合一]
  → ~/.autoresearch/runs/{project-id}/{name}/ (运行状态)
  → N worktree (branch: goalx/{name}/{i}) + engine adapter + tmux
  → heartbeat goroutine 定时触发 master 检查
  → master 监督 subagent → guidance 引导/重启/停止
  → 人类醒来: goalx review → keep / archive / drop
```

## Package Layout

```
autoresearch/
├── go.mod
├── config.go              # Config / Preset / Engine / Validate / Merge / Slugify
├── journal.go             # Journal (JSONL)
├── templates.go           # embed templates/*.tmpl
├── cmd/goalx/main.go      # CLI entry (12 commands)
├── cli/
│   ├── start.go           # goalx start (full workflow)
│   ├── init.go            # goalx init (scaffold goalx.yaml)
│   ├── args.go            # 参数解析（--run, --research, --parallel 等）
│   ├── runctx.go          # ResolveRun（run 上下文解析）
│   ├── tmux.go            # tmux 操作封装
│   ├── worktree.go        # git worktree + merge (ff-only, dirty check)
│   ├── adapter.go         # engine adapter (claude hooks)
│   ├── trust.go           # workspace trust 引导 (claude .claude.json / codex config.toml)
│   ├── heartbeat.go       # heartbeat 命令生成
│   ├── protocol.go        # 模板渲染 (embedded)
│   ├── list.go / status.go / attach.go / stop.go
│   ├── review.go / diff.go / keep.go / archive.go / drop.go / report.go
│   └── harness.go
└── templates/
    ├── master.md.tmpl     # master protocol (heartbeat + acceptance checklist)
    ├── program.md.tmpl    # subagent protocol (resume-first + guidance)
    └── report.md.tmpl
```

## Config Hierarchy

```
Built-in defaults → ~/.autoresearch/config.yaml → .autoresearch/config.yaml → goalx.yaml → CLI flags
```

### Presets — Claude 做大脑，Codex 做双手

| preset | master | research sub | develop sub |
|--------|--------|-------------|-------------|
| default | claude/opus | claude/sonnet | codex/gpt-5.4 |
| turbo | claude/sonnet | claude/haiku | codex/gpt-5.4-mini |
| deep | claude/opus | claude/opus | codex/gpt-5.4 |

gpt-5.3-codex/gpt-5.2 在 codex CLI 触发交互迁移提示，不可用。统一 gpt-5.4。

```yaml
# .autoresearch/config.yaml — 项目级（配一次，可选 commit）
harness:
  command: "go build ./quant_agent/... && go test ./quant_agent/... -count=1"
target: { readonly: ["pkg/", "cmd/"] }

# goalx.yaml — run 级（只写本次独有的）
name: event-sourcing
mode: research
objective: "调研 event sourcing"
parallel: 3
# preset/engine/model 全部继承
```

运行状态全在 `~/.autoresearch/runs/`，项目目录零侵入。

## Commands

```bash
go build ./...                    # build
go test ./... -v                  # test
go vet ./...                      # lint
go build -o bin/goalx ./cmd/goalx  # build CLI binary
```

## Conventions

- **Go 100%**
- Commits: `feat(core|cli):` / `fix|test|docs|refactor:`
- 文件大小 ≤ 500 行
- 零技术债务

## Integration: QuantOS (第一客户)

用 `goalx start` 驱动 AI agent 研发 QuantOS：
- research: master 监督 subagent 调研架构方案
- develop: master 监督 subagent 实施代码改造
</coding_guidelines>
