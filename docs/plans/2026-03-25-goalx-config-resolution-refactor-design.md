# GoalX Config Resolution Refactor Design

**Date:** 2026-03-25

## Goal

Replace GoalX's fragmented config-loading and config-resolution flow with one clean, side-effect-free pipeline. The new model must remove duplicate entry points, remove runtime mutation of shared catalogs, remove compatibility shims and fallback glue, and make every launch path resolve configuration through the same code path.

## Problem

Today config behavior is split across too many places:

- `config.go` has multiple public load helpers with overlapping semantics.
- `cli/launch_config.go` and `cli/phase_run.go` each perform partial config resolution.
- `cli/start.go` mutates already-built config again during launch.
- phase commands inherit full saved run config, which carries stale resolved defaults into new runs.
- user/project config loading mutates package-level `Presets` and `BuiltinDimensions`.
- tests pass against synthetic sparse fixtures that do not match real saved run state.

This creates five concrete failures:

1. Config loading is not pure. Cross-project or long-lived processes can leak catalog state.
2. Auto-detect is applied in more than one layer, so explicit user intent is not reliably preserved.
3. `auto`, `start`, `research`, `develop`, phase commands, and `serve` do not truly share one resolution path.
4. Phase `--preset` and `next_config.preset` can silently fail on real saved runs.
5. The codebase carries legacy seams whose only job is to patch over earlier partial refactors.

## Requirements

- One config resolution authority for every run-creation path.
- Pure config loading with no package-level mutation.
- No compatibility layer, no dual-path rollout, no legacy fallback helpers kept alive "just in case".
- Explicit user intent must win over auto-detect.
- Phase runs must inherit source facts, not stale fully resolved engine/model defaults.
- CLI and HTTP entry points must resolve equivalent requests to equivalent configs.
- Tests must include real end-to-end coverage, not only fixture-level unit tests.

## Constraints

- The end state must be smaller and simpler than today's code.
- Public behavior can change if the current behavior is wrong.
- Internal APIs may be deleted aggressively.
- E2E verification is required before calling the refactor complete.

## Options

### Option 1: Keep current API surface, refactor internals behind helpers

Preserve `LoadRawBaseConfig`, `LoadConfig`, `LoadConfigWithManualDraft`, `ApplyPreset`, `ForceApplyPreset`, and current launch builders, but route them through shared internals.

Pros:
- Lowest migration cost.
- Least test churn.

Cons:
- Keeps the same conceptual clutter.
- Preserves redundant public APIs and encourages new call sites.
- Leaves compatibility logic in place, which conflicts with the agreed clean-cut requirement.

### Option 2: Replace config loading and resolution with a two-stage model

Introduce:

- `ConfigLayers`: raw loaded layers and catalogs
- `ResolveRequest`: normalized command intent
- `ResolvedConfig`: final launchable run config

Every run-creation path resolves through one function. Old public helpers are deleted.

Pros:
- Clear separation of concerns.
- Small enough to execute safely.
- Removes duplicate resolution logic without inventing a large framework.

Cons:
- Requires broad call-site migration.
- Forces tests to be rewritten around the new model.

### Option 3: Full command-spec orchestration framework

Unify command parsing, resolution, and launch into a generic command graph/state machine.

Pros:
- Theoretically the cleanest end state.

Cons:
- Too large for this refactor.
- High risk of over-design.
- Solves more than the actual problem.

## Decision

Choose **Option 2**.

It is the smallest design that fully deletes the current duplication instead of rearranging it. It also supports the clean-cut requirement: we can remove the old helpers entirely once all entry points use the resolver.

## Design

### 1. Split config into load-time data and resolved run config

Add three new types:

- `ConfigLayers`
- `ResolveRequest`
- `ResolvedConfig`

`ConfigLayers` contains only raw loaded facts:

- builtin defaults
- user config
- project config
- optional manual draft
- engine catalog
- preset catalog
- dimension catalog

`ResolveRequest` contains only the current command's intent:

- run mode
- objective
- parallel override
- preset override
- role engine/model overrides
- effort overrides
- context and dimension overrides
- source metadata for phase continuation
- whether resolution is for direct launch or draft emission

`ResolvedConfig` is the only structure allowed to flow into launch code. By the time launch sees it, preset choice, engine/model defaults, target, harness, context, and session definitions are fully decided.

### 2. Make config loading pure

Add a new loader module that does only this:

- read builtin/user/project/manual-draft files
- merge config structs
- clone engine/preset/dimension catalogs locally
- return local catalogs with no package-level writes

The following package-level mutation must be removed:

- writes to `Presets`
- writes to `BuiltinDimensions`

The following public helpers must be removed after migration:

- `LoadRawBaseConfig`
- `LoadConfig`
- `LoadConfigWithManualDraft`

The new loader is allowed to use existing low-level helpers like `mergeConfig` and `copyEngines`, but those helpers become internal implementation details rather than part of the behavioral API.

### 3. Make resolution happen exactly once

Add one resolver entry point:

`ResolveConfig(projectRoot string, layers ConfigLayers, req ResolveRequest) (ResolvedConfig, error)`

Resolution order:

1. Start from merged shared config facts.
2. Apply manual draft if the request uses one.
3. Apply request overrides.
4. Determine preset provenance.
5. If preset is explicit, keep it.
6. If preset is unset, auto-detect once.
7. Apply preset-derived engine/model defaults once.
8. Apply role-specific overrides and effort overrides.
9. Infer target/harness/context/session layout.
10. Validate final resolved config.

There is no later mutation in launcher code. Launch code receives final resolved values and only launches.

### 4. Track preset provenance explicitly

Today `codex` is overloaded to mean both "explicit codex" and "fallback codex". That ambiguity caused the current bug.

The resolver must track provenance explicitly:

- `explicit`
- `detected`
- `unset`

Auto-detect is only allowed when provenance is `unset`.

This removes the need for:

- `ApplyPreset`
- `ForceApplyPreset`
- `finalizeLoadedConfig`

### 5. Stop inheriting full saved run config for phase runs

Phase runs should not use the prior run's already-resolved engine/model defaults as their new base config.

Replace that behavior with `PhaseSourceFacts`, containing only:

- source run identity and lineage
- source objective
- collected report/context artifacts
- inherited target hint
- inherited harness hint
- inherited parallel suggestion if relevant

Phase resolution then uses:

- fresh shared config from current project/user config
- request overrides such as `--preset`, `--master`, `--research-role`, `--develop-role`
- source facts only where inheritance is semantically correct

This ensures phase preset changes are real and predictable.

### 6. Make every command entry point thin

After the refactor:

- `auto`
- `start`
- `research`
- `develop`
- `debate`
- `implement`
- `explore`
- `serve`

must all follow the same shape:

1. parse args / request
2. build `ResolveRequest`
3. load `ConfigLayers`
4. call `ResolveConfig`
5. call launcher

No command-specific config resolution logic should remain in:

- `cli/launch_config.go`
- `cli/phase_run.go`
- `cli/start.go`
- `cli/serve.go`

Any code there should be argument parsing, source-fact extraction, or launch orchestration only.

### 7. Make launch code launch-only

`startWithConfig` can remain as a launcher helper, but it must stop doing any config mutation.

It may:

- generate run name if still absent
- validate immutable launch preconditions
- create run directories/worktrees
- persist run spec
- launch tmux/session actors

It may not:

- detect preset
- rewrite engine/model defaults
- patch role configs

### 8. Delete old logic instead of wrapping it

This refactor explicitly does not allow:

- deprecated wrapper helpers that call the new resolver
- temporary compatibility flags
- old code paths guarded by comments like "keep for now"
- duplicate config builders for CLI vs phase vs serve

The cutover is complete only when the old resolution path is removed.

## File-Level Changes

### New files

- `config_layers.go`
- `config_request.go`
- `config_resolver.go`
- `config_resolver_test.go`
- `cli/phase_source.go`

### Existing files to simplify

- `config.go`
- `dimensions.go`
- `cli/launch_config.go`
- `cli/phase_run.go`
- `cli/start.go`
- `cli/auto.go`
- `cli/research.go`
- `cli/develop.go`
- `cli/debate.go`
- `cli/implement.go`
- `cli/explore.go`
- `cli/serve.go`

### Existing files to rewrite tests around real behavior

- `config_test.go`
- `cli/launch_config_test.go`
- `cli/debate_test.go`
- `cli/implement_test.go`
- `cli/explore_test.go`
- `cli/serve_test.go`
- `cli/test_fixtures_test.go`

## Migration Plan

### Phase 1: Build the new model in parallel

Create pure loaders, request types, resolver, and focused resolver tests. Do not cut over launch paths yet.

### Phase 2: Migrate direct launch paths

Move `auto`, `start`, `research`, and `develop` to the resolver.

### Phase 3: Migrate phase paths

Replace full-config inheritance with source-fact inheritance in `debate`, `implement`, and `explore`.

### Phase 4: Migrate HTTP path

Make `serve` construct the same `ResolveRequest` as CLI and remove duplicated argument-to-config behavior.

### Phase 5: Delete obsolete APIs

Remove old public config loaders and preset helpers, simplify tests, and update docs.

## Testing

### Unit coverage

Add focused tests for:

- config loading purity
- local catalog cloning
- explicit preset beats detect
- unset preset allows detect
- request override precedence
- phase resolution with source facts only

### Integration coverage

Add integration tests for:

- `auto`, `start`, `research`, `develop` resolving the same request shape consistently
- `serve` producing the same resolved config as CLI
- manual draft resolution obeying the same precedence model

### E2E coverage

Add installed-binary E2E tests using temp repos and fake `codex` / `claude` executables to verify:

- zero-config auto-detect chooses the correct preset
- explicit preset is not overridden
- phase `--preset` and `next_config.preset` actually change resolved roles on real saved runs
- `serve` and direct CLI paths are behaviorally equivalent
- no old config path remains reachable

E2E is a release gate for this refactor, not an optional smoke test.

## Non-goals

- No redesign of routing semantics beyond making routing pass through the new resolver.
- No new user-facing configuration features.
- No generic command framework.
- No backward-compatibility wrappers for deleted config APIs.
