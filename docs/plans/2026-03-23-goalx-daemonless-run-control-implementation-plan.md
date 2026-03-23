# GoalX Daemonless Run Control Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace GoalX's heartbeat-centric tmux control path with a daemonless, run-scoped control architecture built on explicit run identity, durable events, leases, reminder delivery state, and canonical completion proof.

**Architecture:** Introduce protocol v2 with local-first run resolution, per-run durable control objects, a run-scoped sidecar process, transport adapters that update delivery state instead of scraping panes, and a proof manifest that becomes the closeout gate. Keep legacy v1 runs readable in explicit compatibility mode, but do not preserve old control semantics as a long-term dual path.

**Tech Stack:** Go 1.21, tmux, JSON/JSONL durable state under run dirs, Go CLI tests, HTTP serve tests, protocol templates.

---

### Task 1: Freeze The Protocol V2 Surface

**Files:**
- Create: `docs/plans/2026-03-23-goalx-daemonless-run-control-design.md`
- Modify: `cli/run_metadata.go`
- Test: `cli/runctx_test.go`

**Step 1: Write failing tests for protocol v2 metadata expectations**

Add tests for:
- new runs writing `protocol_version = 2`
- legacy runs with `protocol_version = 1` remaining readable
- no implicit metadata rewrite during read-only commands

**Step 2: Run the targeted tests to confirm they fail**

Run: `go test ./cli -run 'TestRunMetadata|TestResolveRun' -count=1`

Expected: FAIL because protocol v2 metadata and legacy-read behavior do not exist yet.

**Step 3: Implement protocol version and run identity scaffolding**

Define the v2 metadata contract in `cli/run_metadata.go`:
- bump protocol version
- add `run_id`
- add `epoch`
- keep file-schema version separate from protocol version

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestRunMetadata|TestResolveRun' -count=1`

Expected: PASS for the new metadata behavior.

**Step 5: Commit**

```bash
git add cli/run_metadata.go cli/runctx_test.go docs/plans/2026-03-23-goalx-daemonless-run-control-design.md
git commit -m "refactor: introduce goalx protocol v2 metadata identity"
```

### Task 2: Make Run Resolution Local-First

**Files:**
- Modify: `cli/runctx.go`
- Modify: `cli/global_run_registry.go`
- Modify: `cli/project_registry.go`
- Modify: `cli/serve.go`
- Test: `cli/runctx_test.go`
- Test: `cli/focus_test.go`

**Step 1: Write failing tests for selector behavior**

Cover:
- bare `--run NAME` resolves local project first
- cross-project same-name run requires `project-id/run` or `run_id`
- mutating commands reject ambiguous global lookup

**Step 2: Run the selector tests**

Run: `go test ./cli -run 'TestResolveRun|TestFocus' -count=1`

Expected: FAIL on local-first behavior and explicit cross-project selector rules.

**Step 3: Change resolver behavior**

Implement:
- local-first `ResolveRun`
- explicit global selector path
- `run_id` lookup helpers in registry code
- shared resolver path for CLI and serve

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestResolveRun|TestFocus' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/runctx.go cli/global_run_registry.go cli/project_registry.go cli/serve.go cli/runctx_test.go cli/focus_test.go
git commit -m "refactor: make goalx run resolution local-first"
```

### Task 3: Introduce Control V2 Objects And Read-Only Loaders

**Files:**
- Create: `cli/control_v2.go`
- Create: `cli/control_v2_test.go`
- Modify: `cli/control.go`
- Modify: `cli/storage_paths.go`
- Test: `cli/read_only_test.go`

**Step 1: Write failing tests for v2 control object initialization**

Cover:
- `control/run-identity.json`
- `control/run-state.json`
- `control/events.jsonl`
- `control/leases/*.json`
- `control/reminders.json`
- `control/deliveries.json`

**Step 2: Run the new control object tests**

Run: `go test ./cli -run 'TestControlV2|TestReadOnly' -count=1`

Expected: FAIL because these objects and loaders do not exist.

**Step 3: Implement control v2 schemas and loaders**

Add new types and helpers for:
- run identity
- run lifecycle state
- event append/read
- lease state
- reminder queue
- delivery state

Ensure loaders can read legacy state without rewriting it.

**Step 4: Re-run the tests**

Run: `go test ./cli -run 'TestControlV2|TestReadOnly' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/control_v2.go cli/control_v2_test.go cli/control.go cli/storage_paths.go cli/read_only_test.go
git commit -m "refactor: add goalx control v2 durable objects"
```

### Task 4: Add The Run-Scoped Sidecar Command

**Files:**
- Create: `cli/sidecar.go`
- Create: `cli/sidecar_test.go`
- Modify: `cmd/goalx/main.go`
- Modify: `cli/start.go`
- Modify: `cli/stop.go`
- Test: `cli/start_test.go`
- Test: `cli/lifecycle_test.go`

**Step 1: Write failing tests for sidecar lifecycle**

Cover:
- `start` launches sidecar instead of heartbeat window
- sidecar writes and renews its own lease
- `stop` terminates sidecar and marks terminal run state

**Step 2: Run the sidecar lifecycle tests**

Run: `go test ./cli -run 'TestStart|TestLifecycle|TestSidecar' -count=1`

Expected: FAIL because sidecar command and lifecycle do not exist.

**Step 3: Implement the sidecar command**

Implement a run-scoped loop that:
- renews `sidecar` lease
- checks master/session leases
- appends system events
- never sends keystrokes directly

**Step 4: Wire `start` and `stop`**

Replace heartbeat tmux-window startup with sidecar startup and stop-sidecar cleanup.

**Step 5: Re-run tests**

Run: `go test ./cli -run 'TestStart|TestLifecycle|TestSidecar' -count=1`

Expected: PASS.

**Step 6: Commit**

```bash
git add cli/sidecar.go cli/sidecar_test.go cmd/goalx/main.go cli/start.go cli/stop.go cli/start_test.go cli/lifecycle_test.go
git commit -m "refactor: add run-scoped goalx sidecar"
```

### Task 5: Remove Heartbeat As Wake Delivery

**Files:**
- Modify: `cli/pulse.go`
- Modify: `cli/heartbeat.go`
- Modify: `cli/control.go`
- Test: `cli/control_test.go`
- Test: `cli/heartbeat_test.go`

**Step 1: Write failing tests for heartbeat semantics**

Cover:
- heartbeat only updates lease or legacy compatibility state
- heartbeat does not request raw keystroke wake delivery
- no Codex bare-`Enter` path remains

**Step 2: Run the heartbeat tests**

Run: `go test ./cli -run 'TestHeartbeat|TestPulse|TestControl' -count=1`

Expected: FAIL under the current nudge behavior.

**Step 3: Implement the semantic split**

Change behavior so:
- `pulse` is legacy-only or compatibility-only
- sidecar handles scheduling
- `planAgentNudge` and wake-message scraping logic are removed or isolated behind explicit transport tests

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestHeartbeat|TestPulse|TestControl' -count=1`

Expected: PASS with no unsafe wake delivery path.

**Step 5: Commit**

```bash
git add cli/pulse.go cli/heartbeat.go cli/control.go cli/control_test.go cli/heartbeat_test.go
git commit -m "refactor: decouple goalx heartbeat from wake delivery"
```

### Task 6: Implement Inbox, Reminder, And Delivery State

**Files:**
- Create: `cli/delivery.go`
- Create: `cli/reminder.go`
- Create: `cli/delivery_test.go`
- Create: `cli/reminder_test.go`
- Modify: `cli/tell.go`
- Modify: `cli/guidance_state.go`
- Modify: `cli/add.go`
- Modify: `cli/lifecycle.go`
- Modify: `cli/tmux.go`

**Step 1: Write failing tests for durable delivery**

Cover:
- enqueue message to master/session inbox
- create delivery request with `dedupe_key`
- retry and cooldown behavior
- ack and failure semantics
- no screen-scrape fallback

**Step 2: Run delivery and reminder tests**

Run: `go test ./cli -run 'TestTell|TestDelivery|TestReminder|TestAdd|TestLifecycle' -count=1`

Expected: FAIL because durable delivery state does not exist.

**Step 3: Implement delivery state and reminder queue**

Add:
- inbox append helpers
- delivery records
- reminder scheduler
- explicit transport adapter result updates

Keep session guidance durable, but route delivery through inbox and delivery state rather than immediate `send-keys`.

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestTell|TestDelivery|TestReminder|TestAdd|TestLifecycle' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/delivery.go cli/reminder.go cli/delivery_test.go cli/reminder_test.go cli/tell.go cli/guidance_state.go cli/add.go cli/lifecycle.go cli/tmux.go
git commit -m "refactor: add durable goalx reminder and delivery state"
```

### Task 7: Rework Runtime State And Status Cache Discipline

**Files:**
- Modify: `cli/runtime_state.go`
- Modify: `cli/status.go`
- Modify: `cli/project_registry.go`
- Modify: `cli/global_run_registry.go`
- Test: `cli/status_test.go`
- Test: `cli/storage_model_test.go`
- Test: `cli/read_only_test.go`

**Step 1: Write failing tests for derived-status discipline**

Cover:
- inactive/stopped runs do not show stale wake state
- `status.json` is derived-only
- no reverse sync from project status cache into run state

**Step 2: Run the status tests**

Run: `go test ./cli -run 'TestStatus|TestStorageModel|TestReadOnly' -count=1`

Expected: FAIL because `syncRunStateFromProjectStatus()` still backwrites run state.

**Step 3: Remove split-brain behavior**

Implement:
- derived-only project status cache
- no reverse sync into run runtime state
- lease/event-based control summary
- terminal-state rendering for stopped runs

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestStatus|TestStorageModel|TestReadOnly' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/runtime_state.go cli/status.go cli/project_registry.go cli/global_run_registry.go cli/status_test.go cli/storage_model_test.go cli/read_only_test.go
git commit -m "refactor: make goalx status cache derived-only"
```

### Task 8: Make Observe And Serve Transport-Agnostic

**Files:**
- Modify: `cli/observe.go`
- Modify: `cli/serve.go`
- Test: `cli/serve_test.go`
- Test: `cli/read_only_test.go`

**Step 1: Write failing tests for degraded transport**

Cover:
- `observe` works without tmux by showing control/journal state
- `serve /runs` derives activity from run state plus leases
- `serve tell/status/observe` share the same resolver semantics as CLI

**Step 2: Run the transport-agnostic observer tests**

Run: `go test ./cli -run 'TestServe|TestReadOnly|TestObserve' -count=1`

Expected: FAIL because current logic is tmux-first.

**Step 3: Implement transport-agnostic readers**

Change:
- `observe` to durable-state-first
- `serve` run listing and actions to use shared resolver + derived state
- error and output semantics for degraded transport

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestServe|TestReadOnly|TestObserve' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/observe.go cli/serve.go cli/serve_test.go cli/read_only_test.go
git commit -m "refactor: make goalx observe and serve control-first"
```

### Task 9: Introduce Canonical Proof Manifest

**Files:**
- Create: `cli/proof.go`
- Create: `cli/proof_test.go`
- Modify: `cli/completion.go`
- Modify: `cli/goal_contract.go`
- Modify: `cli/acceptance.go`
- Modify: `cli/verify.go`
- Test: `cli/goal_contract_test.go`
- Test: `cli/verify_test.go`

**Step 1: Write failing proof tests**

Cover:
- required item cannot close without semantic proof entry
- proof manifest must carry evidence class and counter-evidence fields
- `verify` fails when proof and contract versions are inconsistent

**Step 2: Run proof tests**

Run: `go test ./cli -run 'TestGoalContract|TestVerify|TestProof' -count=1`

Expected: FAIL because no proof manifest exists.

**Step 3: Implement `proof/completion.json` and validation**

Add:
- proof manifest schema
- mapper from contract + acceptance + completion into proof checks
- semantic evidence enforcement in `verify`

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestGoalContract|TestVerify|TestProof' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/proof.go cli/proof_test.go cli/completion.go cli/goal_contract.go cli/acceptance.go cli/verify.go cli/goal_contract_test.go cli/verify_test.go
git commit -m "refactor: add canonical goalx completion proof manifest"
```

### Task 10: Upgrade Master And Subagent Protocol Templates

**Files:**
- Modify: `templates/master.md.tmpl`
- Modify: `templates/program.md.tmpl`
- Modify: `cli/protocol.go`
- Test: `cli/protocol_test.go`

**Step 1: Write failing protocol template tests**

Cover:
- protocol text references leases/events/inbox/delivery state
- no `goalx-hb` or heartbeat-as-wake instructions remain
- completion instructions require proof manifest semantics

**Step 2: Run protocol tests**

Run: `go test ./cli -run 'TestProtocol' -count=1`

Expected: FAIL because templates still reference old heartbeat/control semantics.

**Step 3: Update templates and protocol data**

Implement:
- new control object paths in protocol data
- new sidecar/reminder/proof instructions
- removal of stale wake-message narration

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestProtocol' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add templates/master.md.tmpl templates/program.md.tmpl cli/protocol.go cli/protocol_test.go
git commit -m "refactor: align goalx agent protocols with control v2"
```

### Task 11: Add Legacy Compatibility And Migration Guardrails

**Files:**
- Create: `cli/migrate.go`
- Create: `cli/migrate_test.go`
- Modify: `cli/status.go`
- Modify: `cli/observe.go`
- Modify: `cli/verify.go`
- Modify: `cli/save.go`
- Modify: `cli/report.go`
- Test: `cli/read_only_test.go`

**Step 1: Write failing tests for legacy compatibility**

Cover:
- v1 runs remain observable in read-only mode
- no implicit migration during `status`, `observe`, `save`, `report`, `verify`
- explicit migration command or helper, if added, is the only writer

**Step 2: Run compatibility tests**

Run: `go test ./cli -run 'TestReadOnly|TestMigrate|TestSave|TestReport|TestVerify' -count=1`

Expected: FAIL because v2 compatibility boundaries are not enforced.

**Step 3: Implement migration guardrails**

Implement:
- legacy detection
- read-only compatibility path
- explicit migrate command or internal migrator entry point
- old heartbeat path retirement rules

**Step 4: Re-run tests**

Run: `go test ./cli -run 'TestReadOnly|TestMigrate|TestSave|TestReport|TestVerify' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/migrate.go cli/migrate_test.go cli/status.go cli/observe.go cli/verify.go cli/save.go cli/report.go cli/read_only_test.go
git commit -m "refactor: add goalx legacy compatibility guardrails"
```

### Task 12: Run Full Regression And Remove Dead Paths

**Files:**
- Modify: `cli/pulse.go`
- Modify: `cli/heartbeat.go`
- Modify: `cli/start.go`
- Modify: `cli/status.go`
- Modify: `README.md`
- Modify: `deploy/README.md`
- Test: `cli/*_test.go`

**Step 1: Write final regression assertions where coverage is still missing**

Cover:
- no heartbeat tmux window startup
- no stale control summary after stop
- no unsafe Codex wake transport
- local-first resolution documented and tested

**Step 2: Run focused CLI regression**

Run: `go test ./cli -count=1`

Expected: PASS.

**Step 3: Run full repository regression**

Run: `go test ./... -count=1`

Expected: PASS, or only documented pre-existing failures if they already exist outside this slice.

**Step 4: Update docs**

Update CLI and deploy docs to describe:
- control v2
- sidecar lifecycle
- local-first run selectors
- legacy compatibility rules

**Step 5: Commit**

```bash
git add cli/pulse.go cli/heartbeat.go cli/start.go cli/status.go README.md deploy/README.md
git add cli/*_test.go
git commit -m "docs: finalize goalx daemonless control v2 rollout"
```

### Task 13: Final Verification

**Files:**
- Modify: none unless regressions are found

**Step 1: Run the full verification suite**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 2: Spot-check command semantics**

Run:
- `goalx status --help`
- `goalx observe --help`
- `goalx tell --help`

Expected: commands render usage and no legacy heartbeat guidance remains in docs or output.

**Step 3: Review git diff for dead compatibility leftovers**

Run: `git diff --stat HEAD~12..HEAD`

Expected: control v2 files present, no accidental reintroduction of global-first or raw tmux wake semantics.

**Step 4: Final commit if needed**

```bash
git add -A
git commit -m "test: verify goalx control v2 rollout"
```

## Cross-Cutting Review Checklist

- Every mutating command now resolves runs local-first.
- Every observer command is transport-agnostic.
- Every active run has identity, state, lease, inbox, reminder, and delivery objects.
- Sidecar is per-run and stoppable; no hidden global daemon exists.
- `status.json` is derived-only.
- Legacy v1 runs are readable without implicit migration.
- Proof manifest is the canonical closeout gate.
- Protocol templates and Go implementation ship in the same batch.

## Risks To Reject During Implementation

- Adding sidecar without deleting heartbeat-window semantics.
- Keeping pane scraping as a “fallback”.
- Allowing `status.json` to write back into run state.
- Leaving serve and CLI on different resolver logic.
- Tightening proof late, after control changes ship.

## Execution Notes

- Prefer small commits exactly as listed.
- Do not merge partially-complete protocol v2 behavior.
- If a task exposes a missing contract boundary, add the boundary before continuing. Do not paper over it with compatibility code.
