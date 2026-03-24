# GoalX Launch Env Snapshot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make GoalX actor launches use one durable run-scoped environment snapshot and direct tmux command spawning, eliminating launch-time environment drift across `start`, `add`, and `resume`.

**Architecture:** Capture a full env snapshot minus denylist at run creation, persist it under `control/launch-env.json`, and make every actor launch read from that file. Replace initial tmux `send-keys` launch with direct `new-session/new-window` shell-command launch helpers.

**Tech Stack:** Go, tmux CLI, existing GoalX control-store runtime.

---

### Task 1: Add failing tests for launch env snapshot creation

**Files:**
- Modify: `cli/start_test.go`
- Create/Modify: `cli/launch_env_test.go`

**Step 1: Write a failing test for durable snapshot creation**

Add a test that starts a run and expects:
- `control/launch-env.json` exists
- custom env like `FOO_TOOLCHAIN_ROOT=/opt/tools` is present
- denylisted env like `TMUX_PANE` is absent

**Step 2: Run the focused tests and verify they fail**

Run: `go test ./cli -run 'TestStartCapturesRunLaunchEnv|TestLaunchEnv' -count=1`

**Step 3: Implement minimal launch env snapshot storage**

Add a new launch-env state file and persistence helpers.

**Step 4: Re-run focused tests and verify they pass**

Run: `go test ./cli -run 'TestStartCapturesRunLaunchEnv|TestLaunchEnv' -count=1`

**Step 5: Commit**

```bash
git add cli/start_test.go cli/launch_env.go cli/launch_env_test.go
git commit -m "refactor: persist run launch env snapshots"
```

### Task 2: Replace allowlist with full snapshot minus denylist

**Files:**
- Modify: `cli/launch_env.go`
- Test: `cli/launch_env_test.go`

**Step 1: Write failing denylist-focused tests**

Cover:
- preserve arbitrary env key
- strip volatile env keys
- preserve common path/auth vars

**Step 2: Run focused tests and verify failure**

Run: `go test ./cli -run 'TestLaunchEnv' -count=1`

**Step 3: Implement denylist-based snapshot filtering**

Move from allowlist/prefix policy to full capture minus denylist.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestLaunchEnv' -count=1`

**Step 5: Commit**

```bash
git add cli/launch_env.go cli/launch_env_test.go
git commit -m "refactor: switch goalx launch env to denylist snapshots"
```

### Task 3: Add direct tmux launch helpers

**Files:**
- Modify: `cli/tmux.go`
- Test: `cli/start_test.go`
- Test: `cli/lifecycle_test.go`

**Step 1: Write failing tmux helper tests**

Update fake tmux log expectations to require direct shell-command launch via `new-session` / `new-window`, not initial `send-keys`.

**Step 2: Run focused tests and verify failure**

Run: `go test ./cli -run 'TestStartLaunchesOnlyMaster|TestResumeLaunchesParkedSession' -count=1`

**Step 3: Implement direct tmux launch helpers**

Add helper(s) that create master/session windows with cwd and shell-command in one call.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestStartLaunchesOnlyMaster|TestResumeLaunchesParkedSession' -count=1`

**Step 5: Commit**

```bash
git add cli/tmux.go cli/start_test.go cli/lifecycle_test.go
git commit -m "refactor: launch goalx actors directly from tmux"
```

### Task 4: Wire `start` to capture and use run-scoped snapshot

**Files:**
- Modify: `cli/start.go`
- Modify: `cli/launch_env.go`
- Test: `cli/start_test.go`

**Step 1: Write failing start-flow tests**

Assert:
- snapshot file written
- master launch command consumes stored snapshot
- no initial master launch `send-keys` command is used

**Step 2: Run focused tests and verify failure**

Run: `go test ./cli -run 'TestStart' -count=1`

**Step 3: Implement start wiring**

Capture snapshot once at run creation and use it for master launch.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestStart' -count=1`

**Step 5: Commit**

```bash
git add cli/start.go cli/launch_env.go cli/start_test.go
git commit -m "refactor: launch master from durable run env snapshot"
```

### Task 5: Wire `add` to reuse stored run snapshot

**Files:**
- Modify: `cli/add.go`
- Test: `cli/add_test.go`

**Step 1: Write failing regression test**

Start a run with env `A=one`, then call `Add` with env `A=two`, and assert launched session still sees `A=one`.

**Step 2: Run focused test and verify failure**

Run: `go test ./cli -run 'TestAddLaunchesSessionWithRunLaunchEnv' -count=1`

**Step 3: Implement `add` snapshot loading**

Load `control/launch-env.json` and use it for session launch.

**Step 4: Re-run focused test**

Run: `go test ./cli -run 'TestAddLaunchesSessionWithRunLaunchEnv' -count=1`

**Step 5: Commit**

```bash
git add cli/add.go cli/add_test.go
git commit -m "refactor: reuse run launch env for added sessions"
```

### Task 6: Wire `resume` to reuse stored run snapshot

**Files:**
- Modify: `cli/lifecycle.go`
- Test: `cli/lifecycle_test.go`

**Step 1: Write failing resume regression test**

Start/run fixture with stored snapshot, mutate caller env before `Resume`, and assert resumed session launch still uses stored values.

**Step 2: Run focused test and verify failure**

Run: `go test ./cli -run 'TestResumeUsesRunLaunchEnv' -count=1`

**Step 3: Implement resume snapshot loading**

Use stored snapshot for resumed session launch.

**Step 4: Re-run focused test**

Run: `go test ./cli -run 'TestResumeUsesRunLaunchEnv' -count=1`

**Step 5: Commit**

```bash
git add cli/lifecycle.go cli/lifecycle_test.go
git commit -m "refactor: reuse run launch env for resumed sessions"
```

### Task 7: Cover serve-triggered behavior and docs

**Files:**
- Modify: `README.md`
- Modify: `deploy/README.md`
- Modify: `skill/SKILL.md`
- Modify: `skill/openclaw-skill/SKILL.md`
- Modify: `skill/references/advanced-control.md`
- Test: `cli/serve_test.go`

**Step 1: Write/update serve docs/tests**

Clarify that serve-triggered runs use the serve process environment, while later actors remain stable within that run.

**Step 2: Run focused tests**

Run: `go test ./cli -run 'TestServe' -count=1`

**Step 3: Update docs**

Sync launch-env semantics across README and skills.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestServe' -count=1`

**Step 5: Commit**

```bash
git add README.md deploy/README.md skill/SKILL.md skill/openclaw-skill/SKILL.md skill/references/advanced-control.md cli/serve_test.go
git commit -m "docs: describe goalx run-scoped launch env snapshots"
```

### Task 8: Full verification and smoke

**Files:**
- Verify only

**Step 1: Run full unit/integration test suite**

Run: `go test ./... -count=1`

**Step 2: Install updated binary**

Run: `go install ./cmd/goalx`

**Step 3: Sync skill surface**

Run the existing local skill copy commands for Codex and Claude skill directories.

**Step 4: Run installed-binary smoke**

Use a temp repo and fake `codex` / `claude` executables to verify:
- start captures snapshot
- add/resume reuse stored env even after caller env changes
- mixed engine launch still routes correctly
- stop/save still behave

**Step 5: Commit if any verification-only fixes were needed**

```bash
git add .
git commit -m "test: verify goalx launch env snapshot rollout"
```
