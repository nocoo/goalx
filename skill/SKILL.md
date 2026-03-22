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

The master is a **dispatcher + referee**: it decomposes the goal, selects engines, launches subagents via `goalx add`, can spin up temporary `--mode research` sessions inside a develop run, keeps required items covered, rebalances work when one session is stuck, and stops only when the current goal contract is satisfied. Subagents are fully autonomous workers.

## Operating Rules
1. Write the objective as a simple goal, not a task checklist. The master figures out the details.
2. State the current direction, not a long history recap. GoalX persists durable run state; the master and subagents resume from current files plus run metadata.
3. The master may decompose, reorder, and improve, but its plan is not the completion standard. GoalX tracks a run-level `goal-contract.json`; required items can move within the current goal, but the goal is not complete until those required items are done or explicitly waived by the user.
4. The master should actively use available parallel capacity. If one session is blocked and other independent required work remains, it should rebalance or start another session instead of waiting.
5. Subagents should report blockers concisely through the journal: what scope they own, what blocks them, what they depend on, and the next smallest useful split.
6. Route direction changes through the master via `tmux send-keys`, not directly to subagent panes.
7. Interpret `goalx observe` output — report what matters, don't dump raw tmux noise.
8. Keep git hygiene invisible. Handle dirty worktrees silently before `start` or `keep`.

## Quick Start

```bash
goalx auto "goal"
```

That's it. The master starts in tmux, creates subagents as needed, and runs until done. Use `goalx observe` to check progress, `goalx result` to see the outcome.
You usually do not need to restate long background context after compaction; give the current goal or redirect and let GoalX resume from durable state.

Options only when the user wants control:
- `--develop` — hint that code changes are the primary goal
- `--research` — hint that reports/analysis are the primary goal (default)
- `--context /path/a,/path/b` — external reference files
- `--name NAME` — custom run name
For explicit control over config: `goalx init "goal" → edit .goalx/goalx.yaml → goalx start`

Runtime state lives under `~/.goalx/runs/...`; durable saved artifacts live under `<project>/.goalx/runs/...` after `goalx save`. Runs maintain both `goal-contract.json` and acceptance state so completion is tied to required goal items, not just the master's current plan. Journals can also carry blocker/dependency hints so the master can rebalance work without rereading long context. GoalX also adds `.goalx/` to `.git/info/exclude` for local repos so saved run state does not get staged by default.

## Scenario Guide
- Research, investigate, audit: `goalx auto "goal"`
- Fix, implement, refactor: `goalx auto "goal" --develop`
- Reference another repo: `goalx auto "goal" --context /path/to/other-project`
- Check progress: `goalx observe`, `goalx status`, `goalx attach`
- Launch a temporary investigation inside a develop run: `goalx add --run NAME --mode research "investigate X"`
- Run the acceptance gate explicitly: `goalx verify --run NAME`
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
| `goalx add "direction"` | Add a subagent session; use `--mode research` for temporary investigation |
| `goalx keep [NAME] <session>` | Merge session branch into main |
| `goalx save [NAME]` | Save durable artifacts and `artifacts.json` to `.goalx/runs/` |
| `goalx verify [NAME]` | Run the active run's acceptance command and record the result |
| `goalx stop [NAME]` | Graceful shutdown |
| `goalx drop [NAME]` | Cleanup worktrees and branches; refuses unsaved runs until `goalx save` |
| `goalx attach [NAME]` | Attach to tmux session |
| `goalx list` | List all runs |
| `goalx debate` | Generate debate config from prior research |
| `goalx implement` | Generate develop config from consensus |

## Observe and React
- Healthy: summarize progress, wait.
- Stuck 2+ heartbeats: redirect the master; it should rebalance required work instead of just waiting on one session.
- Wrong direction: steer the master, not subagents.
- Need an explicit acceptance check: run `goalx verify` before treating the run as done.
- Complete: `goalx save` then `goalx result` to review. Saved reports are indexed through `artifacts.json`.
