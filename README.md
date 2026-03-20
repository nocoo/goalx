# GoalX

Autonomous research and development framework. Master/Subagent architecture powered by AI coding agents (Claude Code, Codex). One command to launch unattended research, debate, and implementation.

**Framework orchestrates. Agents judge.**

## How It Works

```
goalx auto "investigate authentication system vulnerabilities" --research --parallel 3
```

GoalX creates isolated git worktrees, launches parallel AI agents in tmux, and a master agent supervises — challenging findings, pushing for depth, rescuing failed sessions, and synthesizing results.

```
┌─────────────────────────────────────────────────┐
│  goalx auto "objective"                         │
│                                                 │
│  research → debate (optional) → implement → keep│
│                                                 │
│  tmux session:                                  │
│    master:    opus — supervises, challenges      │
│    session-1: sonnet — depth-first exploration   │
│    session-2: sonnet — adversarial bug hunting   │
│    session-3: sonnet — quantitative analysis     │
│    heartbeat: triggers master check every 2m     │
└─────────────────────────────────────────────────┘
```

## Install

```bash
go install github.com/vonbai/goalx/cmd/goalx@latest
```

Or build from source:

```bash
git clone https://github.com/vonbai/goalx.git
cd goalx
go build -o bin/goalx ./cmd/goalx
```

### Requirements

- Go 1.21+
- tmux
- One of: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [Codex CLI](https://github.com/openai/codex)

## Quick Start

```bash
# Research mode — parallel AI agents investigate your codebase
goalx init "audit code quality and find bugs" --research --parallel 2
goalx start

# Watch progress
goalx observe

# Full auto — research → debate → implement → keep
goalx auto "refactor the config system" --research --parallel 2
```

## Commands

| Command | Description |
|---------|-------------|
| `goalx init` | Generate config from objective |
| `goalx start` | Launch tmux session with master + subagents |
| `goalx auto` | Full pipeline: research → debate → implement |
| `goalx observe` | Live tmux capture from all agents |
| `goalx status` | Journal-based progress summary |
| `goalx add` | Add subagent to running session |
| `goalx save` | Save artifacts to `.goalx/runs/` |
| `goalx debate` | Generate debate config from prior research |
| `goalx implement` | Generate develop config from consensus |
| `goalx keep` | Merge session branch into main |
| `goalx next` | Suggest next pipeline step |
| `goalx review` | Compare all session outputs |
| `goalx stop` | Graceful shutdown |
| `goalx drop` | Cleanup worktrees and branches |

## Pipeline

```
goalx init → start → [observe...] → save
                                       ↓
goalx debate → start → [observe...] → save    (optional)
                                       ↓
goalx implement → start → [observe...] → keep
```

Each stage: CLI generates config, master supervises agents, framework handles git.

## Research Strategies

```bash
goalx init "objective" --research --parallel 3 --strategy depth,adversarial,experimental
```

| Strategy | Focus |
|----------|-------|
| `depth` | Deep-dive the most impactful area |
| `breadth` | Scan all dimensions, prioritize surprises |
| `adversarial` | Find bugs, flaws, security issues |
| `experimental` | Quantify everything with data |
| `comparative` | Compare with industry best practices |
| `web` | Internet research for external perspectives |
| `security` | Security audit from attacker's perspective |
| `performance` | Profile, benchmark, find bottlenecks |

Defaults: parallel=2 gets depth+adversarial, parallel=3 adds experimental.

## Architecture

```
goalx/
├── config.go           # Config hierarchy (4 layers)
├── strategies.go       # Built-in research strategies
├── journal.go          # JSONL journal format
├── templates/
│   ├── master.md.tmpl  # Master agent protocol
│   └── program.md.tmpl # Subagent protocol
├── cli/                # All CLI commands
│   ├── auto.go         # Full pipeline orchestration
│   ├── start.go        # Session launch + worktree setup
│   ├── observe.go      # Live tmux capture
│   └── ...
└── cmd/goalx/main.go   # Entry point
```

### Protocol Design

GoalX is a **protocol scaffolding tool**. The Go code handles orchestration (worktrees, tmux, git), but all intelligence lives in two protocol templates:

**Master** (`master.md.tmpl`): Final responsible party. Writes acceptance checklist, challenges subagent findings, rescues dead sessions, synthesizes results, recommends next steps.

**Subagent** (`program.md.tmpl`): Hypothesis-driven exploration (research) or structured TDD (develop). Communicates via journal files and guidance files.

### Config Hierarchy

```
Built-in defaults → ~/.goalx/config.yaml → .goalx/config.yaml → .goalx/goalx.yaml
```

### Engine Presets

| Preset | Master | Research Sub | Develop Sub |
|--------|--------|-------------|-------------|
| default | claude/opus | claude/sonnet | codex/gpt-5.4 |
| turbo | claude/sonnet | claude/haiku | codex/gpt-5.4-mini |
| deep | claude/opus | claude/opus | codex/gpt-5.4 |

## Claude Code Skill

GoalX includes a Claude Code skill for interactive use:

```bash
mkdir -p ~/.claude/skills/goalx
cp skill/SKILL.md ~/.claude/skills/goalx/SKILL.md
```

Then in Claude Code: `/goalx observe`, `/goalx auto "objective"`, etc.

## License

MIT
