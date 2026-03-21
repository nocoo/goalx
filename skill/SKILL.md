---
name: goalx
description: Use when the user wants to launch goal-driven autonomous research or development, monitor agent progress, or manage GoalX runs. GoalX starts a master agent that self-orchestrates subagents to achieve the user's goal. Even if the user just says "research this" or "investigate that" or "look into X", this skill likely applies.
allowed-tools: Bash, Read, Glob, Grep, Write, Edit
user-invocable: true
---

# GoalX

GoalX launches a master agent that orchestrates everything. The framework provides tools and infrastructure; the master decides what to do.

## Core Concept

```
goalx auto "goal"  →  master starts  →  master creates subagents  →  master evaluates  →  done
```

The master is a **strategist + referee**: it decomposes the goal, selects engines, launches subagents via `goalx add`, checks progress on each heartbeat, restarts dead sessions, and stops when criteria are met. Subagents are fully autonomous workers.

## Operating Rules
1. Write the objective as a simple goal, not a task checklist. The master figures out the details.
2. Route direction changes through the master via `tmux send-keys`, not directly to subagent panes.
3. Interpret `goalx observe` output — report what matters, don't dump raw tmux noise.
4. Keep git hygiene invisible. Handle dirty worktrees silently before `start` or `keep`.

## Quick Start

```bash
goalx auto "goal"
```

That's it. The master starts in tmux, creates subagents as needed, and runs until done. Use `goalx observe` to check progress, `goalx result` to see the outcome.

Options only when the user wants control:
- `--develop` — hint that code changes are the primary goal
- `--research` — hint that reports/analysis are the primary goal (default)
- `--context /path/a,/path/b` — external reference files
- `--name NAME` — custom run name
For explicit control over config: `goalx init "goal" → edit .goalx/goalx.yaml → goalx start`

## Scenario Guide
- Research, investigate, audit: `goalx auto "goal"`
- Fix, implement, refactor: `goalx auto "goal" --develop`
- Reference another repo: `goalx auto "goal" --context /path/to/other-project`
- Check progress: `goalx observe`, `goalx status`, `goalx attach`
- Redirect mid-run: `tmux send-keys -t <session>:master "new direction" Enter`
- View results: `goalx result` or `goalx result --full`

## Commands

| Command | Purpose |
|---------|---------|
| `goalx auto "goal"` | Init + start master, then exit. Master runs in tmux. |
| `goalx init "goal"` | Generate config only |
| `goalx start` | Launch master from existing config |
| `goalx observe [NAME]` | Live capture from all tmux windows |
| `goalx status [NAME]` | Journal-based progress |
| `goalx result [NAME]` | Show summary (`--full` for raw report) |
| `goalx add "direction"` | Add a subagent session (master does this itself) |
| `goalx keep [NAME] <session>` | Merge session branch into main |
| `goalx save [NAME]` | Save artifacts to `.goalx/runs/` |
| `goalx stop [NAME]` | Graceful shutdown |
| `goalx drop [NAME]` | Cleanup worktrees and branches |
| `goalx attach [NAME]` | Attach to tmux session |
| `goalx list` | List all runs |
| `goalx debate` | Generate debate config from prior research |
| `goalx implement` | Generate develop config from consensus |

## Observe and React
- Healthy: summarize progress, wait.
- Stuck 2+ heartbeats: redirect the master.
- Wrong direction: steer the master, not subagents.
- Complete: `goalx save` then `goalx result` to review.

