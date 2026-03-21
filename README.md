# GoalX

Autonomous research and development framework. Master/Subagent architecture powered by AI coding agents (Claude Code, Codex). GoalX starts the master, and the master orchestrates the rest through GoalX tools.

**GoalX provides tools. The master orchestrates.**

## How It Works

```
goalx auto "investigate authentication system vulnerabilities"
```

GoalX creates a run directory and launches a master agent in tmux. The master decides when to call `goalx add`, assigns work, challenges findings, rescues failed sessions, and synthesizes results.

```
┌─────────────────────────────────────────────────┐
│  goalx auto "objective"                         │
│                                                 │
│  master-led run                                 │
│                                                 │
│  tmux session:                                  │
│    master: starts first and reads goalx config   │
│    master: calls goalx add to launch workers     │
│    session-1+: created on demand by the master   │
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
# Give a goal, master handles everything
goalx auto "audit code quality and find bugs"

# Watch progress
goalx observe

# View results
goalx result
```

## Commands

| Command | Description |
|---------|-------------|
| `goalx init` | Generate config from objective |
| `goalx start` | Launch tmux session with the master only |
| `goalx auto` | Init and start one master-led run, then exit |
| `goalx observe` | Live tmux capture from all agents |
| `goalx status` | Journal-based progress summary |
| `goalx add` | Add a session to a running run |
| `goalx save` | Save artifacts to `.goalx/runs/` |
| `goalx debate` | Generate debate config from prior research |
| `goalx implement` | Generate develop config from consensus |
| `goalx keep` | Merge session branch into main |
| `goalx next` | Suggest next pipeline step |
| `goalx result` | Show saved run results (`--full` prints the full research summary) |
| `goalx review` | Compare all session outputs |
| `goalx attach` | Attach to tmux session or window |
| `goalx serve` | Start HTTP API server |
| `goalx stop` | Graceful shutdown |
| `goalx drop` | Cleanup worktrees and branches |

## Single-Run Flow

```
goalx init → start → master reads config
                    → master uses goalx add / keep / save as needed
                    → observe / status while it runs
                    → save / keep when the master finishes
```

`goalx debate` and `goalx implement` still exist as explicit commands, but `goalx auto` no longer routes between phases on the framework side.

## Goal Dimensions

Dimensions define how agents approach the objective — not what they do, but from what angle:

```bash
goalx init "objective" --research --parallel 3 --strategy depth,adversarial,evidence
```

| Dimension | Focus |
|-----------|-------|
| `depth` | Pick the most impactful area, go as deep as possible |
| `breadth` | Scan all dimensions, find blind spots |
| `creative` | Non-obvious solutions, challenge assumptions |
| `feasibility` | Implementation cost, risk, dependencies |
| `adversarial` | Find bugs, flaws, edge cases |
| `evidence` | Quantify everything with data |
| `comparative` | Compare with industry best practices |
| `user` | End-user perspective, usability |

Defaults: parallel=2 → depth+adversarial, parallel=3 → +evidence. Custom dimensions in `~/.goalx/config.yaml`.

## Agent Composition

```bash
# Explicit agent composition with --sub
goalx init "objective" --research --sub claude-code/opus:2 --sub codex/gpt-5.4:1

# Override master engine
goalx init "objective" --master codex/gpt-5.4 --parallel 2

# N workers + 1 auditor pattern
goalx init "objective" --preset claude-h --parallel 3 --auditor codex/gpt-5.4
```

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
│   ├── auto.go         # Init + start, then exit
│   ├── start.go        # Session launch + worktree setup
│   ├── observe.go      # Live tmux capture
│   └── ...
└── cmd/goalx/main.go   # Entry point
```

### Protocol Design

GoalX is a **protocol scaffolding tool**. The Go code launches the master, exposes worker-management tools, and handles git/worktree mechanics; the orchestration logic lives in the protocol templates:

**Master** (`master.md.tmpl`): Final responsible party. Writes acceptance checklist, challenges subagent findings, rescues dead sessions, synthesizes results, recommends next steps.

**Subagent** (`program.md.tmpl`): Hypothesis-driven exploration (research) or structured TDD (develop). Communicates via journal files and guidance files.

### Config Hierarchy

```
Built-in defaults → ~/.goalx/config.yaml → .goalx/config.yaml → .goalx/goalx.yaml
```

### Engine Presets

| Preset | Master | Research Sub | Develop Sub |
|--------|--------|-------------|-------------|
| hybrid | claude/opus | claude/opus | codex/gpt-5.4 |
| claude | claude/opus | claude/sonnet | codex/gpt-5.4 |
| claude-h | claude/opus | claude/opus | claude/opus |
| codex | codex/gpt-5.4 | codex/gpt-5.4 | codex/gpt-5.4 |
| mixed | codex/gpt-5.4 | claude/opus | codex/gpt-5.4 |

Custom presets in `~/.goalx/config.yaml`. Override per-run with `--master`, `--sub`, `--auditor`.

## HTTP API & Remote Management

GoalX includes a lightweight HTTP server for remote management:

```bash
goalx serve    # starts on configured bind address
```

API endpoints:
- `GET /projects` — list all configured workspaces
- `POST /projects/:name/goalx/start` — start a run
- `GET /projects/:name/goalx/observe` — check agent progress
- `POST /projects/:name/goalx/tell` — send instructions to master
- `PUT /projects/:name/goalx/config` — modify run configuration
- `POST /workspaces` — add project directory (auto git-init if needed)
- `GET /runs` — all active runs across all projects

Bearer token auth + IP binding. See [deploy/](deploy/) for config example and systemd unit.

## OpenClaw Integration

GoalX can be managed by an [OpenClaw](https://github.com/openclaw/openclaw) agent via Lark, Telegram, or Web UI:

```bash
cp -r skill/openclaw-skill /path/to/openclaw/workspace/skills/goalx
```

The OpenClaw agent calls GoalX HTTP API to start research, check progress, and manage tasks across all projects. See [deploy/README.md](deploy/README.md) for setup guide.

## Claude Code Skill

For local interactive use in Claude Code:

```bash
mkdir -p ~/.claude/skills/goalx
cp skill/SKILL.md ~/.claude/skills/goalx/SKILL.md
```

Then: `/goalx observe`, `/goalx auto "objective"`, etc.

## License

MIT
