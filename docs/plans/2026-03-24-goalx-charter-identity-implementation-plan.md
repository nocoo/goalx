# GoalX Charter And Identity Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add immutable run charter, immutable session identity, and low-disturbance identity refresh so GoalX retains original objective, exploration doctrine, and completion rules across compaction, park/resume, relaunch, and long-running control cycles.

**Architecture:** Introduce `run-charter.json` as the immutable semantic anchor, `sessions/session-N/identity.json` as the immutable worker identity source, and `control/identity-fence.json` as a derived refresh fence. Keep `goal.json` as the only mutable completion boundary, `acceptance.json` as the only mutable gate, and `proof/completion.json` as the only canonical closeout proof. Cut over all live run commands and protocol templates in one hard path with no compatibility layer for pre-cutover runs.

**Tech Stack:** Go 1.21, tmux, JSON/JSONL run artifacts, GoalX CLI, protocol templates, README and skill docs, Go test suite.

---

### Task 1: Freeze Charter Vocabulary And Hard-Cutover Rules

**Files:**
- Create: `docs/plans/2026-03-24-goalx-charter-identity-design.md`
- Modify: `cli/protocol_test.go`
- Modify: `cli/start_test.go`
- Modify: `cli/lifecycle_test.go`
- Modify: `README.md`
- Modify: `skill/SKILL.md`

**Step 1: Write the failing tests**

Add or update tests to assert:
- rendered protocols reference `run-charter.json`
- subagent protocols reference session identity
- live startup produces charter-aware paths
- no test assumes prompt-only doctrine is authoritative

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestRenderMasterProtocol|TestRenderSubagentProtocol|TestStart|TestResume' -count=1`

Expected: FAIL because charter and session identity do not exist yet.

**Step 3: Freeze the public vocabulary**

Update docs/tests so these are explicit:
- immutable `run-charter.json`
- immutable `sessions/session-N/identity.json`
- derived `control/identity-fence.json`
- hard cutover, no live compatibility

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestRenderMasterProtocol|TestRenderSubagentProtocol|TestStart|TestResume' -count=1`

Expected: still FAIL until implementation lands, but expectations now point at the new protocol.

**Step 5: Commit**

```bash
git add docs/plans/2026-03-24-goalx-charter-identity-design.md cli/protocol_test.go cli/start_test.go cli/lifecycle_test.go README.md skill/SKILL.md
git commit -m "test: freeze goalx charter identity vocabulary"
```

### Task 2: Add Charter, Session Identity, And Identity-Fence Storage

**Files:**
- Create: `cli/charter.go`
- Create: `cli/session_identity.go`
- Create: `cli/identity_fence.go`
- Create: `cli/charter_test.go`
- Create: `cli/session_identity_test.go`
- Create: `cli/identity_fence_test.go`
- Modify: `cli/control_state.go`
- Modify: `cli/run_metadata.go`

**Step 1: Write the failing tests**

Cover:
- `RunCharterPath(runDir)` returns `run-charter.json`
- `SessionIdentityPath(runDir, "session-1")` returns `sessions/session-1/identity.json`
- `IdentityFencePath(runDir)` returns `control/identity-fence.json`
- charter creation is immutable and includes doctrine/completion rules
- session identity creation is immutable and contains resolved engine/model/target/harness
- control run identity includes `charter_id` and `charter_hash`

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestCharter|TestSessionIdentity|TestIdentityFence|TestEnsureControlState' -count=1`

Expected: FAIL because the new storage and control identity fields do not exist.

**Step 3: Implement the storage layer**

Add:
- `RunCharter`
- `SessionIdentity`
- `IdentityFence`
- path helpers, loaders, savers, constructors, hash helpers

Extend:
- `RunMetadata` with `RootRunID`, `CharterID`, `CharterHash`
- `ControlRunIdentity` with `CharterPath`, `CharterHash`, `Mode`, `PhaseKind`

**Step 4: Make cutover explicit**

Add strict validation helpers so live protocol commands can reject runs missing:
- `run-charter.json`
- `run-metadata.charter_id`
- `control/run-identity.json` charter linkage

Do not add fallback parsing for missing charter on live runs.

**Step 5: Re-run the targeted tests**

Run: `go test ./cli -run 'TestCharter|TestSessionIdentity|TestIdentityFence|TestEnsureControlState' -count=1`

Expected: PASS.

**Step 6: Commit**

```bash
git add cli/charter.go cli/session_identity.go cli/identity_fence.go cli/charter_test.go cli/session_identity_test.go cli/identity_fence_test.go cli/control_state.go cli/run_metadata.go
git commit -m "refactor: add goalx charter and identity storage"
```

### Task 3: Wire Start, Add, Resume, And Phase Creation To Durable Identity

**Files:**
- Modify: `cli/start.go`
- Modify: `cli/add.go`
- Modify: `cli/lifecycle.go`
- Modify: `cli/phase_run.go`
- Modify: `cli/debate.go`
- Modify: `cli/implement.go`
- Modify: `cli/explore.go`
- Modify: `cli/start_test.go`
- Modify: `cli/add_test.go`
- Modify: `cli/implement_test.go`
- Modify: `cli/debate_test.go`
- Modify: `cli/explore_test.go`

**Step 1: Write the failing tests**

Cover:
- `start` writes charter, metadata linkage, control identity linkage, and initial fence
- `add` writes immutable session identity using resolved effective session config
- `resume` reads session identity instead of recomputing from current config
- phase transitions generate a fresh charter while preserving root lineage

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestStart|TestAdd|TestResume|TestImplement|TestDebate|TestExplore' -count=1`

Expected: FAIL because launch and phase code still rely on current config/objective only.

**Step 3: Implement the launch cutover**

Change:
- `start` to create charter before rendering protocols
- `add` to persist session identity before rendering subagent protocol
- `resume` to load session identity and refuse missing identity files
- phase-run builders to derive new charter from source lineage and new launch objective

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestStart|TestAdd|TestResume|TestImplement|TestDebate|TestExplore' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/start.go cli/add.go cli/lifecycle.go cli/phase_run.go cli/debate.go cli/implement.go cli/explore.go cli/start_test.go cli/add_test.go cli/implement_test.go cli/debate_test.go cli/explore_test.go
git commit -m "refactor: wire goalx launch lifecycle to charter identity"
```

### Task 4: Cut Over Master And Subagent Protocol Rendering

**Files:**
- Modify: `cli/protocol.go`
- Modify: `templates/master.md.tmpl`
- Modify: `templates/program.md.tmpl`
- Modify: `cli/protocol_test.go`

**Step 1: Write the failing tests**

Cover:
- master protocol is charter-first
- master protocol requires path comparison for non-trivial work
- master protocol states acceptance is verification-only, not scope authority
- subagent protocol resumes from session identity + charter first
- protocol data includes `CharterPath`, `SessionIdentityPath`, and control identity path for subagents

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestRenderMasterProtocol|TestRenderSubagentProtocol' -count=1`

Expected: FAIL because template data and prompts still rely on prompt-only doctrine.

**Step 3: Update protocol data and templates**

Make templates:
- read charter before mutable boundary files on relevant cycles
- teach the immutable/completion split cleanly
- remove any implication that prompt text is the only source of doctrine
- keep prompts bootstrap-sized; do not duplicate the entire charter body in markdown prose

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestRenderMasterProtocol|TestRenderSubagentProtocol' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/protocol.go templates/master.md.tmpl templates/program.md.tmpl cli/protocol_test.go
git commit -m "refactor: render goalx protocols from durable charter identity"
```

### Task 5: Add Identity-Fence Refresh And Sidecar Integration

**Files:**
- Modify: `cli/identity_fence.go`
- Modify: `cli/sidecar.go`
- Modify: `cli/control.go`
- Modify: `cli/reminder.go`
- Modify: `cli/control_state_test.go`
- Modify: `cli/lease_loop_test.go`

**Step 1: Write the failing tests**

Cover:
- identity fence hashes charter/goal/acceptance/coordination content
- sidecar updates the fence
- sidecar queues a deduped `refresh-context` reminder only when fence changes
- no reminder is emitted on unchanged fence

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestIdentityFence|TestSidecar|TestPulse|TestDeliverDueControlReminders' -count=1`

Expected: FAIL because no fence-based refresh exists.

**Step 3: Implement low-disturbance refresh**

Use content hashes, not mutable version counters.

Keep the mechanism narrow:
- recompute fence
- compare to previous fence
- queue durable refresh only on change

Do not add unconditional every-tick refresh messages.

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestIdentityFence|TestSidecar|TestPulse|TestDeliverDueControlReminders' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/identity_fence.go cli/sidecar.go cli/control.go cli/reminder.go cli/control_state_test.go cli/lease_loop_test.go
git commit -m "refactor: add goalx identity fence refresh flow"
```

### Task 6: Harden Verify And Closeout Against Scope Drift

**Files:**
- Modify: `cli/verify.go`
- Modify: `cli/proof.go`
- Modify: `cli/completion.go`
- Modify: `cli/adapter.go`
- Modify: `cli/verify_test.go`
- Modify: `cli/proof_test.go`
- Modify: `cli/adapter_test.go`

**Step 1: Write the failing tests**

Cover:
- charter provenance mismatch fails verification
- acceptance pass does not imply `done` when required goal items remain open
- verification emits `done`, `incomplete`, or `phase_complete_but_goal_incomplete`
- stop-hook respects the stricter completion result

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestVerify|TestProof|TestGenerateMasterAdapter' -count=1`

Expected: FAIL because current verify path does not enforce charter-based completion doctrine.

**Step 3: Implement the hardening**

Add charter-aware checks for:
- run identity consistency
- immutable completion rules
- goal boundary completeness
- acceptance gate result

Do not turn charter into a second goal item store.

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestVerify|TestProof|TestGenerateMasterAdapter' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/verify.go cli/proof.go cli/completion.go cli/adapter.go cli/verify_test.go cli/proof_test.go cli/adapter_test.go
git commit -m "refactor: harden goalx closeout with charter provenance"
```

### Task 7: Cut Over Status, Observe, Serve, Save, And Result Together

**Files:**
- Modify: `cli/status.go`
- Modify: `cli/observe.go`
- Modify: `cli/derived_run.go`
- Modify: `cli/serve.go`
- Modify: `cli/save.go`
- Modify: `cli/result.go`
- Modify: `cli/status_test.go`
- Modify: `cli/serve_test.go`
- Modify: `cli/save_test.go`

**Step 1: Write the failing tests**

Cover:
- status prints `run_id`, `epoch`, and `charter=ok|missing|mismatch`
- observe shows identity health through the shared status summary
- serve run listings expose the same identity-backed status
- save exports charter and session identity files
- result does not rely on deprecated objective semantics

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestStatus|TestObserve|TestServe|TestSave|TestResult' -count=1`

Expected: FAIL because read/export surfaces are not charter-aware.

**Step 3: Implement the cutover**

Make these surfaces consume the same live semantics:
- identity health from metadata + control identity + charter linkage
- saved provenance includes charter and session identity
- no ephemeral control files in saved artifacts

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestStatus|TestObserve|TestServe|TestSave|TestResult' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/status.go cli/observe.go cli/derived_run.go cli/serve.go cli/save.go cli/result.go cli/status_test.go cli/serve_test.go cli/save_test.go
git commit -m "refactor: expose goalx charter identity across read and export surfaces"
```

### Task 8: Remove Duplicate Objective And Doctrine Semantics

**Files:**
- Modify: `cli/coordination.go`
- Modify: `cli/runtime_state.go`
- Modify: `cli/derived_run.go`
- Modify: `cli/project_registry.go`
- Modify: `cli/global_run_registry.go`
- Modify: `cli/report.go`
- Modify: `cli/coordination_test.go`

**Step 1: Write the failing tests**

Cover:
- `coordination.objective` is no longer treated as authoritative
- runtime/registry/report surfaces source immutable objective from charter or metadata display only
- no live protocol logic depends on duplicated objective doctrine fields

**Step 2: Run the targeted tests**

Run: `go test ./cli -run 'TestCoordination|TestRunRegistry|TestDerivedRunState' -count=1`

Expected: FAIL because objective semantics still live in multiple files.

**Step 3: Remove or downgrade duplicates**

Keep objective in caches only where needed for UI/reporting.

Ensure live protocol decisions no longer depend on:
- `coordination.objective`
- `runtime_state.objective`
- prompt-only doctrine copies

**Step 4: Re-run the targeted tests**

Run: `go test ./cli -run 'TestCoordination|TestRunRegistry|TestDerivedRunState' -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add cli/coordination.go cli/runtime_state.go cli/derived_run.go cli/project_registry.go cli/global_run_registry.go cli/report.go cli/coordination_test.go
git commit -m "refactor: remove duplicated goalx objective semantics"
```

### Task 9: Sync Docs, Skills, And Saved-Artifact Guidance In The Same Batch

**Files:**
- Modify: `README.md`
- Modify: `deploy/README.md`
- Modify: `skill/SKILL.md`
- Modify: `skill/references/advanced-control.md`
- Modify: `skill/openclaw-skill/SKILL.md`
- Modify: `skill/agents/openai.yaml`
- Modify: `report.md`

**Step 1: Write the failing checks**

Add grep-based checks or tests that fail if docs still claim:
- prompt-only exploration authority
- mutable acceptance defines scope
- no charter/session identity exists
- old save/status semantics remain

**Step 2: Run the checks**

Run: `rg -n "goal-contract|prompt-only|contract is the boundary|acceptance defines scope" README.md deploy/README.md skill report.md`

Expected: FAIL until docs are updated.

**Step 3: Update all operator-facing surfaces**

Keep code and docs aligned in the same batch:
- charter is immutable anchor
- goal is mutable boundary
- acceptance is verification-only
- saved runs include charter/identity provenance
- pre-cutover live runs are unsupported

**Step 4: Re-run the checks**

Run: `rg -n "goal-contract|prompt-only|contract is the boundary|acceptance defines scope" README.md deploy/README.md skill report.md`

Expected: no matches, or only historical-plan references outside active docs.

**Step 5: Commit**

```bash
git add README.md deploy/README.md skill/SKILL.md skill/references/advanced-control.md skill/openclaw-skill/SKILL.md skill/agents/openai.yaml report.md
git commit -m "docs: sync goalx charter identity protocol"
```

### Task 10: Final Audit Sweep And Full Verification

**Files:**
- Modify: any files uncovered by final grep/test audit

**Step 1: Run full test suite**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 2: Run protocol grep audit**

Run:

```bash
rg -n "run-charter|session-N/identity|identity-fence|goal.json|acceptance.json|completion.json" cli templates README.md skill
rg -n "goal-contract|compatibility path|fallback identity|derive.*session identity" cli templates README.md skill
```

Expected:
- first grep shows the new surfaces wired across code/templates/docs
- second grep shows no active legacy identity semantics in live code paths

**Step 3: Run smoke checklist**

Verify manually with installed `goalx`:
- start run
- add session
- park/resume session
- status/observe show identity health
- verify blocks partial completion
- save exports charter and session identity

**Step 4: Fix any uncovered drift**

Patch any final mismatches immediately. Do not defer to a follow-up task.

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: finalize goalx charter identity hard cutover"
```

## Execution Notes

- Treat this as a non-compatible live protocol cutover.
- If a live run lacks charter/identity files, fail fast instead of trying to recover it.
- Do not stage docs as a later cleanup pass; update them in the same slice that changes operator-visible behavior.
- Do not reintroduce prompt-only doctrine as a hidden second source of truth.
