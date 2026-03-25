# GoalX Config Resolution Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace GoalX's duplicated config loading and resolution chain with one pure loader plus one resolver, delete legacy config helpers and fallback logic, and prove the cutover with real E2E coverage.

**Architecture:** Introduce `ConfigLayers`, `ResolveRequest`, and `ResolvedConfig`; route every run-creation entry point through `ResolveConfig`; make phase runs inherit only source facts; then delete the old config APIs and compatibility behavior entirely.

**Tech Stack:** Go, existing GoalX CLI/runtime code, tmux-backed launch flow, Go test, installed-binary E2E with fake engine executables.

---

### Task 1: Add failing tests that lock the desired resolver semantics

**Files:**
- Modify: `config_test.go`
- Create: `config_resolver_test.go`

**Step 1: Write failing tests for explicit-vs-detected preset behavior**

Cover:
- explicit `preset: codex` is preserved even when both engines are installed
- unset preset triggers auto-detect
- per-load catalogs do not mutate shared package state

**Step 2: Run focused tests to verify failure**

Run: `go test ./... -run 'TestResolveConfig|TestLoadConfig' -count=1`
Expected: FAIL on missing resolver API and current duplicated behavior

**Step 3: Add minimal test scaffolding for future resolver types**

Create placeholder test helpers and table structure for `ConfigLayers`, `ResolveRequest`, and `ResolvedConfig`.

**Step 4: Re-run focused tests**

Run: `go test ./... -run 'TestResolveConfig|TestLoadConfig' -count=1`
Expected: FAIL for the same reasons, with stable test names in place

**Step 5: Commit**

```bash
git add config_test.go config_resolver_test.go
git commit -m "test: lock config resolver target semantics"
```

### Task 2: Introduce pure config-layer loading

**Files:**
- Create: `config_layers.go`
- Modify: `config.go`
- Modify: `dimensions.go`
- Test: `config_test.go`
- Test: `config_resolver_test.go`

**Step 1: Write failing tests for pure layer loading**

Add tests that load two different project configs sequentially and assert:
- one project's presets do not leak into the next
- one project's dimensions do not leak into the next
- engine catalog overrides remain local to the returned result

**Step 2: Run focused tests to verify failure**

Run: `go test ./... -run 'TestLoadConfigLayers|TestLoadConfigProjectEnvelopeOverridesUserEnvelope' -count=1`
Expected: FAIL because current code mutates global catalogs

**Step 3: Implement `ConfigLayers` and local catalog cloning**

Add:
- `ConfigLayers`
- pure layer load helper(s)
- local copies of engines, presets, dimensions

Do not resolve preset or infer launch defaults here.

**Step 4: Re-run focused tests**

Run: `go test ./... -run 'TestLoadConfigLayers|TestLoadConfigProjectEnvelopeOverridesUserEnvelope' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add config_layers.go config.go dimensions.go config_test.go config_resolver_test.go
git commit -m "refactor: make goalx config loading pure"
```

### Task 3: Introduce request and resolved-config types

**Files:**
- Create: `config_request.go`
- Create: `config_resolver.go`
- Test: `config_resolver_test.go`

**Step 1: Write failing tests for request precedence**

Cover:
- shared config baseline
- manual draft overlay
- CLI override precedence
- explicit preset precedence over auto-detect

**Step 2: Run focused tests to verify failure**

Run: `go test ./... -run 'TestResolveConfig' -count=1`
Expected: FAIL because resolver types/functions do not exist or do not yet honor precedence

**Step 3: Implement `ResolveRequest`, `ResolvedConfig`, and `ResolveConfig`**

Resolver rules:
- detect exactly once
- explicit preset wins
- apply preset defaults once
- validate final resolved result

**Step 4: Re-run focused tests**

Run: `go test ./... -run 'TestResolveConfig' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add config_request.go config_resolver.go config_resolver_test.go
git commit -m "refactor: add unified goalx config resolver"
```

### Task 4: Cut direct launch commands to the new resolver

**Files:**
- Modify: `cli/launch_config.go`
- Modify: `cli/start.go`
- Modify: `cli/auto.go`
- Modify: `cli/research.go`
- Modify: `cli/develop.go`
- Test: `cli/launch_config_test.go`
- Test: `cli/start_test.go`
- Test: `cli/auto_test.go`

**Step 1: Write failing direct-launch regression tests**

Cover:
- `auto`
- `start`
- `research`
- `develop`

Assert they all resolve equivalent engine/model defaults for equivalent requests.

**Step 2: Run focused tests to verify failure**

Run: `go test ./cli -run 'TestBuildLaunchConfig|TestStart|TestAuto' -count=1`
Expected: FAIL because direct launch paths still use old builders and late mutation

**Step 3: Replace direct launch config building with `ResolveConfig`**

Remove preset detection and config mutation from `startWithConfig`. Make launch code consume already-resolved config only.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestBuildLaunchConfig|TestStart|TestAuto' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/launch_config.go cli/start.go cli/auto.go cli/research.go cli/develop.go cli/launch_config_test.go cli/start_test.go cli/auto_test.go
git commit -m "refactor: route direct launches through config resolver"
```

### Task 5: Replace phase config inheritance with source facts

**Files:**
- Create: `cli/phase_source.go`
- Modify: `cli/phase_run.go`
- Modify: `cli/debate.go`
- Modify: `cli/implement.go`
- Modify: `cli/explore.go`
- Test: `cli/debate_test.go`
- Test: `cli/implement_test.go`
- Test: `cli/explore_test.go`
- Test: `cli/test_fixtures_test.go`

**Step 1: Write failing tests against real saved-run specs**

Build real saved runs through actual resolution, then assert:
- phase `--preset` changes resolved roles
- `next_config.preset` changes resolved roles
- source lineage and context are still preserved

**Step 2: Run focused tests to verify failure**

Run: `go test ./cli -run 'TestDebateAppliesNextConfigPreset|TestImplementAppliesNextConfigPreset|TestExplore' -count=1`
Expected: FAIL on real resolved run specs

**Step 3: Implement `PhaseSourceFacts` and phase re-resolution**

Stop copying full saved config as the new base. Rebuild phase config from shared config plus extracted source facts.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestDebateAppliesNextConfigPreset|TestImplementAppliesNextConfigPreset|TestExplore' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/phase_source.go cli/phase_run.go cli/debate.go cli/implement.go cli/explore.go cli/debate_test.go cli/implement_test.go cli/explore_test.go cli/test_fixtures_test.go
git commit -m "refactor: rebuild goalx phase runs from source facts"
```

### Task 6: Cut `serve` to the same request/resolution path

**Files:**
- Modify: `cli/serve.go`
- Modify: `cli/serve_test.go`

**Step 1: Write failing tests for serve/CLI equivalence**

Assert equivalent HTTP payloads and CLI invocations resolve the same:
- preset
- role defaults
- effort overrides
- context and dimensions

**Step 2: Run focused tests to verify failure**

Run: `go test ./cli -run 'TestServe' -count=1`
Expected: FAIL because `serve` still rebuilds argument semantics separately

**Step 3: Refactor `serve` to build `ResolveRequest` instead of duplicating config behavior**

Keep request parsing, but remove duplicate resolution logic from the HTTP path.

**Step 4: Re-run focused tests**

Run: `go test ./cli -run 'TestServe' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/serve.go cli/serve_test.go
git commit -m "refactor: make goalx serve use unified config resolution"
```

### Task 7: Delete obsolete config APIs and fallback helpers

**Files:**
- Modify: `config.go`
- Modify: `cli/start.go`
- Modify: `cli/launch_config.go`
- Modify: `cli/phase_run.go`
- Modify: any remaining call sites found by search
- Test: `config_test.go`
- Test: `cli/*_test.go`

**Step 1: Write a deletion checklist from current references**

Search for and list remaining references to:
- `LoadRawBaseConfig`
- `LoadConfig`
- `LoadConfigWithManualDraft`
- `ApplyPreset`
- `ForceApplyPreset`
- `finalizeLoadedConfig`

**Step 2: Run the search and confirm references still exist**

Run: `rg -n 'LoadRawBaseConfig|LoadConfigWithManualDraft|LoadConfig\\(|ApplyPreset|ForceApplyPreset|finalizeLoadedConfig'`
Expected: matches remain

**Step 3: Delete old APIs and remove remaining call sites**

Do not wrap or deprecate. Delete dead logic and update callers to the resolver or pure layer loader.

**Step 4: Re-run the search**

Run: `rg -n 'LoadRawBaseConfig|LoadConfigWithManualDraft|LoadConfig\\(|ApplyPreset|ForceApplyPreset|finalizeLoadedConfig'`
Expected: no matches for deleted APIs

**Step 5: Commit**

```bash
git add config.go cli/start.go cli/launch_config.go cli/phase_run.go
git commit -m "refactor: delete legacy goalx config APIs"
```

### Task 8: Update docs to match the clean-cut model

**Files:**
- Modify: `README.md`
- Modify: `skill/SKILL.md`
- Modify: `skill/references/advanced-control.md`
- Modify: any config-related docs found by search

**Step 1: Write doc assertions before editing**

Docs must state:
- one resolver path
- no legacy config helper path
- phase config is rebuilt, not copied
- explicit preset beats detect

**Step 2: Run a doc search to find stale wording**

Run: `rg -n 'LoadConfig|LoadRawBaseConfig|goalx.yaml|auto-detect|preset' README.md skill docs`
Expected: multiple stale references to old flow

**Step 3: Update docs to reflect the new architecture**

Remove wording that implies multiple config resolution paths or legacy compatibility.

**Step 4: Re-run the doc search**

Run: `rg -n 'LoadConfig|LoadRawBaseConfig|ForceApplyPreset|finalizeLoadedConfig' README.md skill docs`
Expected: no stale references

**Step 5: Commit**

```bash
git add README.md skill/SKILL.md skill/references/advanced-control.md docs/plans
git commit -m "docs: describe unified goalx config resolution"
```

### Task 9: Add installed-binary E2E coverage for the refactor

**Files:**
- Create/Modify: `cli/e2e_test.go`
- Create/Modify: `cli/test_fixtures_test.go`
- Modify: any helper files needed for fake engine executables

**Step 1: Write failing E2E tests**

Use temp repos and fake `codex` / `claude` binaries to cover:
- zero-config detect
- explicit preset preservation
- phase preset override on real saved run
- serve/CLI equivalence

**Step 2: Run focused E2E tests to verify failure**

Run: `go test ./cli -run 'TestE2EConfigResolution' -count=1`
Expected: FAIL until resolver cutover is complete

**Step 3: Implement E2E helpers and fix any missing wiring**

Add helper(s) that install fake engines on `PATH`, create temp repos, build the binary behavior under test, and assert observable run-spec/output results.

**Step 4: Re-run focused E2E tests**

Run: `go test ./cli -run 'TestE2EConfigResolution' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/e2e_test.go cli/test_fixtures_test.go
git commit -m "test: add goalx config resolution e2e coverage"
```

### Task 10: Full verification and final cleanup

**Files:**
- Verify only

**Step 1: Run the full test suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: Build the CLI from source**

Run: `go build ./...`
Expected: PASS

**Step 3: Run vet**

Run: `go vet ./...`
Expected: PASS

**Step 4: Run one final search for banned leftovers**

Run: `rg -n 'ForceApplyPreset|finalizeLoadedConfig|compat|legacy fallback|keep for now|TODO: remove old path'`
Expected: no matches related to removed config logic

**Step 5: Commit if verification required any last cleanup**

```bash
git add .
git commit -m "test: verify unified goalx config resolution cleanup"
```

Plan complete and saved to `docs/plans/2026-03-25-goalx-config-resolution-refactor-implementation-plan.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
