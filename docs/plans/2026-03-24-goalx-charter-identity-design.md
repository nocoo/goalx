# GoalX Charter And Identity Hardening Design

## Status

Drafted on 2026-03-24 after local code review plus parallel subagent review focused on:

- immutable charter shape
- low-disturbance identity refresh
- protocol integration and hidden coupling surfaces

## Decision

Harden GoalX's long-running autonomy with three new protocol objects:

- immutable `run-charter.json`
- immutable `sessions/session-N/identity.json`
- derived `control/identity-fence.json`

Keep the current lightweight goal-boundary model:

- `goal.json` remains the only mutable completion boundary
- `acceptance.json` remains the only mutable verification gate
- `goal-log.jsonl` remains audit-only
- `proof/completion.json` remains the only canonical closeout proof written by `goalx verify`

This is a hard cutover. Do not add compatibility write paths, dual-read logic, or fallback protocol semantics for live runs.

## Problem

GoalX now preserves mutable goal boundary better than before, but long-running master/session behavior still has a structural weakness:

1. original user objective and exploration doctrine still live mainly in rendered prompts
2. `run-metadata.json` is strong enough for run identity, but too thin for immutable behavioral contract
3. `goal.json.version` and `coordination.json.version` are not reliable freshness fences
4. `goalx add` and `goalx resume` still recompute session identity from current config, so dynamically added workers can drift after compaction, park/resume, or later config changes
5. `acceptance.json` is mutable, so if completion doctrine is not anchored elsewhere, the master can converge on the smallest passable gate instead of the full user goal

The failure mode is not "forgot file paths." The failure mode is that after enough cycles, compaction, or restarts, the system can still remember the current repo state but lose the durable identity that says:

- what the original user asked for
- what the master is allowed to reinterpret versus not narrow
- that non-trivial goals require path comparison and autonomous path switching
- that partial progress is not equivalent to `done`

## Goals

- Preserve the original user objective as a durable, immutable anchor.
- Preserve exploration doctrine and non-narrowing rules outside prompt-only state.
- Preserve durable worker identity across add, park, resume, and relaunch.
- Avoid frequent prompt bloat or repeated rereads of large files on every control cycle.
- Prevent mutable acceptance changes from quietly shrinking completion scope.
- Keep completion truth small and singular.
- Cut over all related code, templates, docs, saved artifacts, and tests together.

## Non-Goals

- Do not introduce a second immutable goal contract that competes with `goal.json`.
- Do not turn GoalX into an event-sourced system.
- Do not add a global daemon.
- Do not preserve live compatibility for pre-cutover runs.
- Do not add high-frequency "identity reminder" spam to master or subagents.

## Protocol Model

### 1. `run-charter.json`

`run-charter.json` is immutable and semantic. It is the durable constitution for a single run.

It is not:

- the current mutable goal boundary
- an execution status file
- a second closeout proof file

It records facts and rules that must survive compaction and must not be silently narrowed.

Proposed shape:

```json
{
  "version": 1,
  "charter_id": "charter_abc123",
  "run_id": "run_abc123",
  "root_run_id": "run_root123",
  "run_name": "corpus-land",
  "project_id": "data-dev-autoresearch",
  "mode": "research|develop",
  "phase_kind": "research|debate|implement|explore",
  "original_user_objective": "...",
  "run_launch_objective": "...",
  "source_run": "saved-run-name",
  "source_phase": "research",
  "parent_run": "parent-run-name",
  "completion_standard": "full_goal",
  "partial_completion_requires_user_approval": true,
  "narrow_scope_requires_user_approval": true,
  "required_outcomes_may_expand_but_not_shrink": true,
  "acceptance_is_verification_only": true,
  "exploration_doctrine": {
    "compare_paths_before_commit": true,
    "minimum_paths": 3,
    "allow_autonomous_path_switch": true,
    "architecture_path_required_for_non_trivial": true
  },
  "role_contracts": {
    "master": {
      "must_dispatch": true,
      "must_verify_before_done": true,
      "must_finish_if_workers_fail": true
    },
    "research_subagent": {
      "read_only": true,
      "may_recommend_better_path": true,
      "may_not_self_close_run": true
    },
    "develop_subagent": {
      "may_change_code": true,
      "may_recommend_better_path": true,
      "may_not_self_close_run": true
    }
  },
  "paths": {
    "goal": "/abs/path/goal.json",
    "acceptance": "/abs/path/acceptance.json",
    "proof": "/abs/path/proof/completion.json"
  },
  "created_at": "RFC3339"
}
```

Rules:

- once written, it is immutable for that run
- it defines doctrine and completion principles, not current work allocation
- it does not list current required items; that remains in `goal.json`
- it does not define the current gate command; that remains in `acceptance.json`

### 2. `sessions/session-N/identity.json`

Each durable worker gets one immutable identity file.

Proposed shape:

```json
{
  "version": 1,
  "session_name": "session-2",
  "role_kind": "master-derived-research|master-derived-develop",
  "mode": "research|develop",
  "engine": "claude-code|codex",
  "model": "opus|gpt-5.4",
  "target": {
    "files": ["report.md"],
    "readonly": ["."]
  },
  "harness": {
    "command": "go test ./..."
  },
  "origin_charter_id": "charter_abc123",
  "created_at": "RFC3339"
}
```

Rules:

- `resume` must use this file, not recompute identity from current config
- this file is worker identity only; it is not where current assignment lives
- current assignment remains in durable inbox entries

### 3. `control/identity-fence.json`

`control/identity-fence.json` is derived and operational. It is not a source of truth.

Proposed shape:

```json
{
  "version": 1,
  "run_id": "run_abc123",
  "epoch": 2,
  "charter_hash": "sha256:...",
  "goal_hash": "sha256:...",
  "acceptance_hash": "sha256:...",
  "coordination_hash": "sha256:...",
  "updated_at": "RFC3339"
}
```

Rules:

- it exists only to let master/sidecar cheaply detect meaningful identity changes
- it uses content hashes, not schema version counters
- it is safe to regenerate at any time

## Completion Authority

Completion authority is intentionally split:

- `run-charter.json` defines immutable completion principles
- `goal.json` defines the current required outcomes
- `acceptance.json` defines the current verification gate
- `proof/completion.json` records the verified closeout result

This split is important.

`acceptance.json` may change, but it is never allowed to redefine the completion boundary. It only changes how the current goal is tested.

## Completion Rules

The charter makes these rules explicit:

- `done` means the current run goal is fully satisfied
- partial progress is not `done`
- milestone completion is not `done`
- budget exhaustion is not `done`
- acceptance success without required goal satisfaction is not `done`

`goalx verify` must therefore distinguish:

- `done`
- `incomplete`
- `phase_complete_but_goal_incomplete`

The last state is allowed for phase transitions or saved outputs, but it must not be reported as final completion.

## Mutable Surfaces

### `goal.json`

`goal.json` remains the only mutable completion boundary.

It may:

- add `source=master` required items
- split user items into smaller user-locked items
- add/remove optional improvements
- move required items between `open`, `claimed`, and `waived`

It may not:

- delete or narrow `source=user` required scope without user approval
- treat optional work as a substitute for unfinished required work

### `acceptance.json`

`acceptance.json` remains the only mutable gate.

It may:

- keep the same gate
- expand the gate
- rewrite the gate when quality improves

It may not:

- narrow the gate without explicit user approval
- act as scope authority

## Identity Refresh Model

The refresh mechanism must be low-disturbance.

### Master

On every control cycle:

1. read `control/identity-fence.json`
2. compare to last-seen fence
3. only reread heavy identity files when:
   - fence changed
   - control cycle follows resume/compaction
   - a user `tell` arrived
   - before dispatch decisions
   - before `goalx verify`
   - before phase transition

Heavy reread set:

- `run-charter.json`
- `run-metadata.json`
- `goal.json`
- `acceptance.json`
- `coordination.json`

### Sidecar

The sidecar continues as the lightweight supervisor. It does not become a second planner.

New behavior:

- recompute `identity-fence.json`
- when the fence changes, enqueue a deduped `refresh-context` reminder/inbox event for master

It does not:

- force rereads every tick
- emit high-frequency subagent reminders by default

### Subagents

Subagents reread durable identity only at natural boundaries:

- session launch
- resume after park/restart/compaction
- after unread inbox changes
- before declaring `idle`
- before any closeout-sensitive claim

Required reread order on resume:

1. `sessions/session-N/identity.json`
2. `run-charter.json`
3. unread session inbox
4. recent journal tail
5. worktree state

## Review Matrix

This refactor is only valid if all of these surfaces cut over together.

### Storage / Identity

- `cli/run_metadata.go`
- `cli/control_state.go`
- new charter and session-identity storage helpers
- new identity-fence helpers

### Launch / Lifecycle

- `cli/start.go`
- `cli/add.go`
- `cli/lifecycle.go`
- `cli/phase_run.go`
- `cli/debate.go`
- `cli/implement.go`
- `cli/explore.go`

### Protocol Rendering

- `cli/protocol.go`
- `templates/master.md.tmpl`
- `templates/program.md.tmpl`

### Control / Refresh

- `cli/sidecar.go`
- `cli/reminder.go`
- `cli/control.go`

### Verification / Closeout

- `cli/verify.go`
- `cli/proof.go`
- `cli/completion.go`
- `cli/adapter.go`

### Read Models / UX

- `cli/status.go`
- `cli/observe.go`
- `cli/derived_run.go`
- `cli/serve.go`
- `cli/report.go`

### Export / Saved Runs

- `cli/save.go`
- `cli/result.go`
- saved artifact manifests

### Duplicate Semantics To Remove Or Downgrade

- `run-metadata.objective` as protocol doctrine source
- `coordination.objective` as authoritative boundary/objective source
- `runtime_state.objective` as anything beyond display cache
- any prompt-only statement that is not anchored in charter

### Documentation / Skills

- `README.md`
- `deploy/README.md`
- `skill/SKILL.md`
- `skill/references/advanced-control.md`
- `skill/openclaw-skill/SKILL.md`
- `skill/agents/openai.yaml`

### Tests

- start/add/resume/park/stop/drop
- protocol render tests
- verify/proof/adapter tests
- status/observe/serve/read-only tests
- save/result tests
- phase transition tests

## Hard Cutover Rules

- Live runs without `run-charter.json` are unsupported.
- Live sessions without `sessions/session-N/identity.json` are unsupported.
- Do not dual-read old and new identity semantics.
- Do not derive session identity from current config once identity files exist.
- Do not keep behavioral doctrine only in prompts.
- Do not add a compatibility writer for old saved artifacts.

## Why This Is The Best Trade-Off

This is the smallest design that closes the real gap:

- lighter than root/phase charter chains
- stronger than prompt-only doctrine
- stronger than run-metadata-only anchoring
- safer than a second immutable goal baseline

It preserves one mutable completion boundary while making identity and doctrine durable enough for long-running autonomous execution.
