# GoalX Master Dispatcher Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the master actively maintain required-work coverage and parallel utilization without turning GoalX into a heavy workflow engine.

**Architecture:** Keep the existing run-level `goal-contract.json` as the completion boundary. Tighten the master and subagent protocol templates so the master acts as a lightweight dispatcher/referee, and extend journal entries with minimal blocker metadata that helps the master rebalance work.

**Tech Stack:** Go, text templates, JSONL journals, GoalX protocol docs

---

### Task 1: Lock the new behavior into protocol tests

**Files:**
- Modify: `cli/protocol_test.go`
- Modify: `journal_test.go`

**Step 1: Write the failing tests**

Add assertions for:
- master identity shifting from `strategist + referee` to `dispatcher + referee`
- uncovered required work being treated as a scheduling bug
- dispatching independent required work when parallel capacity exists
- subagent blocker/dependency fields in the journal example

**Step 2: Run test to verify it fails**

Run: `go test ./cli -run 'TestRender(Subagent|Master)Protocol' -count=1 && go test . -run 'Test(LoadJournal|Summary)' -count=1`

Expected: FAIL because the current templates and journal struct do not expose the new scheduler language or blocker metadata.

### Task 2: Make the protocol templates state the new contract

**Files:**
- Modify: `templates/master.md.tmpl`
- Modify: `templates/program.md.tmpl`

**Step 1: Update master responsibilities**

Keep the prompt short while adding only these hard rules:
- every required item must be done/waived, owned, or explicitly blocked
- uncovered required work is a scheduling bug
- idle parallel capacity should be used for independent required work
- stuck required work must be reassigned, split, or taken over

**Step 2: Update subagent reporting**

Add optional blocker metadata fields to the journal example and a concise instruction for `status:"stuck"` entries.

### Task 3: Extend the journal schema with minimal blocker fields

**Files:**
- Modify: `journal.go`
- Test: `journal_test.go`

**Step 1: Add optional subagent metadata**

Add:
- `quality`
- `owner_scope`
- `blocked_by`
- `depends_on`
- `can_split`
- `suggested_next`

**Step 2: Improve summaries**

If the last entry is `stuck` and has `blocked_by`, include the blocker in `Summary(...)`.

### Task 4: Sync human-facing docs

**Files:**
- Modify: `README.md`
- Modify: `skill/SKILL.md`

**Step 1: Describe the master as a lightweight dispatcher**

Explain that the master keeps required items covered and should rebalance parallel work instead of waiting on a single blocked session.

**Step 2: Describe the new blocker handoff**

Mention that subagents report ownership and blockers through the journal so the master can redirect work quickly.

### Task 5: Verify, install, and publish

**Files:**
- Modify: `/root/.codex/skills/goalx/SKILL.md`
- Modify: `/root/.claude/skills/goalx/SKILL.md`

**Step 1: Run full verification**

Run:
- `go test ./... -count=1`
- `go build ./...`
- `go install ./cmd/goalx`
- `/root/go/bin/goalx help`

**Step 2: Install the updated skill**

Copy `skill/SKILL.md` into:
- `/root/.codex/skills/goalx/SKILL.md`
- `/root/.claude/skills/goalx/SKILL.md`

**Step 3: Commit**

Use a commit message that reflects the scheduler refactor.
