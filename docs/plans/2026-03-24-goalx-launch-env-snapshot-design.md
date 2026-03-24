# GoalX Launch Env Snapshot Design

**Date:** 2026-03-24

## Goal

Eliminate environment drift between the shell that starts a run and the tmux-hosted actors that GoalX launches later. New runs should carry one durable launch environment snapshot, and every actor launch in that run should use that snapshot instead of whatever environment happens to exist in the current tmux pane or the later `goalx add` caller.

## Problem

Today GoalX launches master/session actors through tmux and injects a curated environment allowlist from the current process:

- `start` captures the environment of the current `goalx start` caller.
- `add` and `resume` capture the environment of the later `goalx add` / `goalx resume` caller.
- The injected environment is allowlist-based, so arbitrary project-specific variables are silently dropped.
- Initial actor launch still depends on `send-keys`, so GoalX creates a shell first and then types a command into it.

This creates three failure modes:

1. A run started from one shell and resumed from another gets inconsistent actor environments.
2. Project-specific tooling variables vanish because they are not on the allowlist.
3. tmux shell startup state can still interfere with launch behavior.

## Requirements

- One run-scoped environment authority for each live run.
- New runs capture the caller environment once at run creation.
- `add` and `resume` must reuse the stored run environment, not the current caller environment.
- Actor launch should not depend on typing the initial command via `send-keys`.
- The snapshot should preserve most caller environment variables.
- We must exclude variables known to be shell-/tmux-/engine-session-specific and unsafe to reuse.
- No compatibility layer: new protocol only.

## Options

### Option 1: Keep allowlist and expand it

Add more variables to `launchEnvKeys` / `launchEnvPrefixes`.

Pros:
- Smallest code change.

Cons:
- Still reactive and incomplete.
- Keeps drift between `start` and later `add` / `resume`.
- Still relies on tmux pane shell for initial launch.

### Option 2: Run-scoped full snapshot minus denylist, keep `send-keys`

Capture a durable env snapshot at run start and use it for future launches, but keep current tmux launch flow.

Pros:
- Fixes cross-command drift.
- Simpler than changing tmux launch path.

Cons:
- Still depends on shell-in-pane bootstrap behavior.
- Leaves one unnecessary source of launch fragility.

### Option 3: Run-scoped full snapshot minus denylist + direct tmux spawn

Capture a durable env snapshot at run start, persist it, and launch master/session windows directly with tmux shell-command arguments instead of `send-keys`.

Pros:
- Single environment authority per run.
- Removes pane bootstrap race for initial actor launch.
- Matches the desired mental model: a run uses the environment it was born with.

Cons:
- Touches tmux helpers and launch tests.

## Decision

Choose **Option 3**.

This is the clean cutover that actually fixes the environment model instead of extending the current allowlist workaround.

## Design

### Durable file

Add `control/launch-env.json`.

This file is immutable for a run after creation and stores:

- captured key/value pairs
- snapshot version
- created timestamp

It is run-scoped and becomes the only source for actor launch environment.

### Snapshot policy

Replace the current allowlist with:

- capture all non-empty environment variables from `os.Environ()`
- remove a strict denylist of volatile or unsafe variables

Initial denylist:

- `TMUX`
- `TMUX_PANE`
- `PWD`
- `OLDPWD`
- `SHLVL`
- `PS1`
- `PROMPT_COMMAND`
- `TERM_PROGRAM`
- `TERM_PROGRAM_VERSION`
- `CODEX_THREAD_ID`
- `CODEX_SESSION_ID`
- `CLAUDE_SESSION_ID`

Also drop obvious shell bookkeeping variables if present, but do not aggressively strip generic project-specific variables.

### Launch flow

#### `start`

- capture current caller env
- write `control/launch-env.json`
- create tmux master session with cwd and shell-command in one step
- launch command uses stored snapshot, not ad hoc `os.Environ()`

#### `add`

- load `control/launch-env.json`
- create session window with cwd and shell-command in one step
- no initial `send-keys` for launch

#### `resume`

- same as `add`

### tmux helpers

Introduce direct-launch helpers instead of relying on `send-keys`:

- `NewSessionWithCommand(name, firstWindow, cwd, command)`
- `NewWindowWithCommand(session, window, cwd, command)`

`SendKeys` remains for reminders/wake nudges only, not initial actor launch.

### Serve behavior

`goalx serve` keeps current behavior: it launches runs using the environment of the serve process itself. That remains a distinct source by design. This refactor only guarantees per-run consistency after launch, not equivalence to arbitrary client terminals.

### Docs

Update README / deploy / skill docs to say:

- actor launch environment is captured once when the run starts
- later session launches reuse that run-scoped snapshot
- serve-triggered runs inherit the serve process environment

## Testing

Add coverage for:

- snapshot creation at start
- denylist stripping of volatile env
- arbitrary custom env survives snapshot
- `add` reuses stored run snapshot even if caller env changes later
- `resume` reuses stored run snapshot even if caller env changes later
- tmux initial launch now uses `new-session` / `new-window` shell-command path, not `send-keys`

## Non-goals

- No project-level service discovery.
- No remote env synchronization.
- No runtime mutation of a run's captured launch env.
