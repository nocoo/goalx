# AutoResearch 架构优化调研 — Summary

**Objective:** 调研 autoresearch 自身架构的优化空间
**Sessions:** 2 parallel subagents, 3 heartbeat cycles
**Total findings:** 18 unique findings across both sessions

---

## Key Architectural Insights

1. **Templates are engine-agnostic by design** — 0 engine-specific references in 306 template lines. Adding a new engine requires only config changes, zero template changes. This is the strongest design element.

2. **Zero goroutines — all process management delegated to tmux.** Heartbeat is a shell `while sleep` loop, not a Go goroutine. This is simpler and more robust than a concurrent Go approach. CLAUDE.md's "heartbeat goroutine" is a documentation error.

3. **Framework overhead is negligible — AI agent cost is the binding constraint.** 14ms per worktree, 324K per session. Scaling bottleneck is AI API cost and master context window, not framework code.

4. **Error handling architecture is clean** (wrap → return → exit at main) but has 5 critical suppressed `os.UserHomeDir()`/`os.Getwd()` sites that produce cryptic failures when HOME is unset.

5. **Session-path formulas duplicated 18 times across 7 files** — highest-ROI refactoring target. A 4-function helper (~20 lines) eliminates all duplications.

---

## Bugs Found

| # | Issue | File | Effort |
|---|-------|------|--------|
| B1 | No cleanup on partial `Start()` failure | cli/start.go | Low |
| B2 | `report.go:23` says "ar" not "goalx" | cli/report.go:23 | Trivial |
| B3 | Adapter hook echo has literal `\n` not newline | cli/adapter.go:44 | Trivial |
| B4 | Master restart command missing C-c kill step | templates/master.md.tmpl | Trivial |
| B5 | `init.go:70` uses raw nanoseconds, not `time.Hour` | cli/init.go:70 | Trivial |

## Dead Code

| # | Issue | File | Effort |
|---|-------|------|--------|
| D1 | `report.md.tmpl` never rendered, undefined fields | templates/report.md.tmpl | Trivial |
| D2 | `LoadBaseConfig` + `loadBaseConfig` — 0 callers | config.go | Trivial |
| D3 | `ResolveSubagentCommand` — pure passthrough | config.go:359 | Trivial |

## Prioritized Recommendations

| Priority | Action | Effort | Impact |
|----------|--------|--------|--------|
| P0 | Fix rename bug "ar" → "goalx" in report.go | 1 min | Bug fix |
| P1 | Extract 4 session-path helpers | 15 min | Eliminates 18 duplications |
| P2 | Remove dead code (LoadBaseConfig, passthrough, template) | 5 min | config.go under 500 lines |
| P3 | Add startup guard for UserHomeDir + Getwd in main() | 5 min | Eliminates 5 cryptic failure paths |
| P4 | Fix adapter \n bug + master restart C-c + init.go nanoseconds | 5 min | 3 trivial bugs |
| P5 | Test pure CLI functions (parseSessionIndex, etc.) | 20 min | Coverage 28.9% → ~35% |
| P6 | Add heartbeat floor warning | 2 min | Better UX |

## Not Recommended (with reasoning)

- **Refactoring start.go into sub-functions** — readable as-is, single linear function
- **Replacing mergeConfig with reflection** — loses type safety for marginal savings
- **Adding Go-level process monitoring** — violates "框架做编排，agent 做判断"
- **Adding rollback to Start()** — `goalx drop` handles cleanup adequately
- **Interface abstraction for tmux/git** — high effort, marginal value for personal tool

## Strengths Confirmed

- Engine-agnostic template design (0 engine refs)
- 100% template-struct field coverage for active templates
- Multi-run conflict-free by design (ProjectID + RunDir + TmuxSessionName)
- Security via slugify sanitization (safe-by-construction)
- Worktree-as-sandbox isolation model (ff-only merge + user review)
- Single external dependency (gopkg.in/yaml.v3)

---

*Full reports available in session worktrees: autoresearch-1/report.md (361 lines) and autoresearch-2/report.md (436+ lines)*
