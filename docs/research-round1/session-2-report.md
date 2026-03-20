# AutoResearch Architecture Optimization Report

**Objective:** 调研 autoresearch 自身架构的优化空间
**Approach:** Breadth-first + adversarial

---

## Finding: Codebase is compact but start.go is a god function

- **Confidence**: HIGH
- **Evidence**:
  - Total: 2278 production lines + 857 test lines across 33 Go files
  - `cli/start.go` has 269 lines and **exactly 1 function** (`Start()`). This single function handles: arg parsing, config loading, auto-name generation, validation, conflict checking, dirty worktree warning, session expansion, directory creation, config snapshotting, session iteration (worktree creation, journal init, guidance init, adapter generation, trust bootstrap), master engine resolution, protocol rendering (both master and subagent), journal/acceptance init, tmux session creation, master launch, subagent launch, heartbeat launch, and status printing.
  - By contrast, other CLI files are well-decomposed: `init.go` (91 lines), `stop.go` (34 lines), `review.go` (81 lines)
- **Counter-evidence**: The function is linear (no branching logic), well-commented with numbered steps, and reads clearly from top to bottom. It's procedural orchestration, not complex logic.
- **Implication**: Despite readability, this monolithic function is untestable in isolation. Any test would need to mock tmux, git worktrees, filesystem, and config loading simultaneously. This directly explains the 28.9% CLI coverage.

---

## Finding: Test coverage is critically low in CLI package (28.9%)

- **Confidence**: HIGH
- **Evidence**:
  - Overall coverage: 35.1%
  - Root package: 60.6% (config, journal — testable pure logic)
  - CLI package: 28.9% (commands — side-effect heavy)
  - cmd/goalx: 0% (main function — expected)
  - Functions at 0% coverage: `LoadConfig`, `LoadBaseConfig`, `loadBaseConfigRaw`, `LoadRawBaseConfig`, `ApplyPreset` (public wrapper), `copyEngines`, `hasDirtyWorktree`, `TagArchive`, `main`
  - All 12 CLI commands (`Start`, `Stop`, `Review`, `Keep`, `Archive`, `Drop`, `Diff`, `Report`, `List`, `Status`, `Attach`, `Init`) depend directly on `exec.Command` for git/tmux and `os.UserHomeDir()` for path resolution
- **Counter-evidence**: The tests that DO exist are well-written. `git_test.go` (148 lines) tests worktree/merge operations using real temporary git repos. `trust_test.go` tests trust file generation. The testing approach is pragmatic, not missing due to negligence.
- **Implication**: 24 direct `exec.Command` calls in CLI code with no abstraction layer make the package fundamentally hard to test without integration-level setup. This is the #1 structural limitation.

---

## Finding: config.go violates the ≤500-line convention (513 lines)

- **Confidence**: HIGH
- **Evidence**:
  - `config.go`: 513 lines, 22 functions (13 exported, 9 unexported)
  - Convention from CLAUDE.md: "文件大小 ≤ 500 行"
  - Functions in config.go span 5 distinct responsibilities:
    1. Type definitions (lines 1-82): 8 struct types + 2 constants
    2. Presets + built-in engines (lines 84-149): 3 presets, 3 engines, defaults
    3. Config loading + merging (lines 151-246): LoadYAML, loadBaseConfig, LoadConfig, mergeConfig
    4. Validation (lines 283-387): ValidateConfig, validateInteractiveEngine
    5. Utility functions (lines 389-513): ResolvePrompt, ExpandSessions, ProjectID, RunDir, Slugify, copyEngines
  - All 22 functions are in one file. The mergeConfig function alone (lines 448-500) is 52 lines of repetitive field-by-field assignment.
- **Counter-evidence**: The file is only 13 lines over the limit. Each function is small (avg 23 lines/function). Splitting would create import cycles or artificial package boundaries.
- **Implication**: Marginal violation. The real issue isn't line count but the number of distinct responsibilities. Splitting types + presets into a separate file would be clean and zero-risk.

---

## Finding: Silent behavior changes without warning

- **Confidence**: HIGH
- **Evidence**:
  - `start.go:234-237`: Heartbeat interval silently changed from any value < 30s to 300s (5 minutes):
    ```go
    checkSec := int(cfg.Master.CheckInterval.Seconds())
    if checkSec < 30 {
        checkSec = 300
    }
    ```
    This is a 10x change from the configured 5-minute default, applied silently. If a user sets `check_interval: 10s`, they get 300s with no warning.
  - `init.go:70`: Budget uses raw nanoseconds instead of `time.Duration` arithmetic:
    ```go
    cfg.Budget = ar.BudgetConfig{MaxDuration: 6 * 3600_000_000_000}
    ```
    Should be `6 * time.Hour`. This is fragile and error-prone.
  - `start.go:86`: `filepath.Abs` error silently ignored:
    ```go
    absProjectRoot, _ := filepath.Abs(projectRoot)
    ```
  - `config.go:172,414`: `os.UserHomeDir()` errors silently ignored
- **Counter-evidence**: The heartbeat floor may be intentional to prevent accidental resource waste from very frequent checks.
- **Implication**: Silent behavior modifications are a debugging nightmare. A user troubleshooting "why is my heartbeat slow" would find no logs, no warnings, no clue.

---

## Finding: report.go still references old "ar" command name

- **Confidence**: HIGH
- **Evidence**: `cli/report.go:23`:
  ```go
  return fmt.Errorf("usage: ar report [--run NAME]")
  ```
  Should be `goalx report`. All other commands use `goalx` correctly.
- **Counter-evidence**: None. This is a straightforward rename miss.
- **Implication**: Minor bug. User-facing inconsistency after the ar→goalx rename.

---

## Finding: JournalEntry is an untyped union struct

- **Confidence**: MEDIUM
- **Evidence**: `journal.go:12-26`:
  ```go
  type JournalEntry struct {
      // Subagent fields
      Round  int    `json:"round,omitempty"`
      Commit string `json:"commit,omitempty"`
      Desc   string `json:"desc,omitempty"`
      Status string `json:"status,omitempty"`
      // Master fields
      Ts       string `json:"ts,omitempty"`
      Action   string `json:"action,omitempty"`
      Session  string `json:"session,omitempty"`
      Finding  string `json:"finding,omitempty"`
      Reason   string `json:"reason,omitempty"`
      Guidance string `json:"guidance,omitempty"`
  }
  ```
  10 fields with `omitempty`, no type discrimination. Code uses heuristics to distinguish (`if last.Round > 0` for subagent, `if last.Action != ""` for master) in `Summary()` and `report.go`.
- **Counter-evidence**: JSONL is inherently schema-less. The struct works correctly as-is. Splitting would add complexity for minimal benefit since the journal reading code is already small (68 lines).
- **Implication**: Low risk currently due to small codebase. Would become a maintenance problem if journal entries grow more fields or new entry types are added.

---

## Finding: Duplicated CLI command boilerplate pattern

- **Confidence**: MEDIUM
- **Evidence**: 8 of 12 CLI commands follow the exact same pattern:
  ```go
  func Command(projectRoot string, args []string) error {
      runName, rest, err := extractRunFlag(args)
      // ... flag parsing ...
      rc, err := ResolveRun(projectRoot, runName)
      // ... business logic ...
  }
  ```
  Seen in: `stop.go`, `review.go`, `keep.go`, `archive.go`, `drop.go`, `diff.go`, `report.go`, `status.go`

  Each duplicates: flag extraction, positional run name fallback, usage error formatting, ResolveRun call.
- **Counter-evidence**: Each command's flag parsing has minor variations (some accept session names, some don't). The duplication is 5-8 lines per command. Abstracting it might reduce readability for negligible savings.
- **Implication**: Marginal. A `withRun(fn)` wrapper could save ~50 lines total but adds indirection. Current approach is acceptable for 12 commands.

---

## Finding: mergeConfig is repetitive but correct

- **Confidence**: HIGH
- **Evidence**: `config.go:448-500` — 52 lines of field-by-field "if overlay.X != zero { base.X = overlay.X }". 17 fields merged manually.
  - Go has no built-in struct merge. Reflection-based alternatives exist (mergo library) but add a dependency.
  - The pattern is consistent and each field uses the correct zero-value check for its type (empty string, 0, nil slice, 0 duration).
- **Counter-evidence**: Manual merging catches bugs that reflection misses (e.g., `Parallel > 0` vs `!= 0`, `[]string` length check vs nil check). The explicit approach is actually safer.
- **Implication**: Not a real problem. The verbosity is the cost of correctness in Go without generics-based merging.

---

## Finding: Adapter hooks.json schema may conflict with existing hooks

- **Confidence**: MEDIUM
- **Evidence**: `adapter.go:36-41`:
  ```go
  var hooks []map[string]string
  if raw, ok := doc["hooks"]; ok {
      if err := json.Unmarshal(raw, &hooks); err != nil {
          return fmt.Errorf("parse hooks array: %w", err)
      }
  }
  hooks = append(hooks, map[string]string{...})
  ```
  This unconditionally appends a new hook on every call. If `GenerateAdapter` is called multiple times (e.g., after a restart), duplicate hooks accumulate. The hooks.json structure uses `[]map[string]string` but Claude Code's actual hooks schema uses a different format with `event`/`command` fields that may have additional required fields.

  Additionally, `adapter.go:43-45` uses `\n` in the echo command which is a Go string escape, not a shell escape — the actual output will contain a literal backslash-n, not a newline.
- **Counter-evidence**: The `--assume-unchanged` git flag prevents this file from showing in diffs, and worktrees are ephemeral. Duplicate hooks would just print the same warning twice.
- **Implication**: The `\n` in the shell command is a potential bug. Should be `\\n` for shell or removed entirely.

---

## Finding: No cleanup on Start() failure (partial state leak)

- **Confidence**: HIGH
- **Evidence**: `start.go:14-269` creates multiple side effects in sequence:
  1. Directories created (line 70-74)
  2. Config snapshot written (line 81)
  3. Worktrees created (line 125)
  4. Journal/guidance files created (lines 130-137)
  5. Adapter files generated (line 140)
  6. Trust bootstrapped (line 144)
  7. Master protocol rendered (line 177)
  8. Subagent protocols rendered (lines 181-200)
  9. Master journal/acceptance created (lines 203-208)
  10. Tmux session created (line 211)
  11. Master launched (lines 216-219)
  12. Subagents launched (lines 222-231)
  13. Heartbeat launched (lines 239-244)

  If step 7 fails (e.g., template rendering error), steps 1-6 leave orphaned directories, worktrees, and branches with no cleanup. Running `goalx start` again fails with "run directory already exists." The user must manually `goalx drop` first.
- **Counter-evidence**: `goalx drop` exists and works correctly (verified by `TestDropRemovesRunDirectoryAndBranch`). The failure mode is annoying but recoverable.
- **Implication**: A `defer cleanup` pattern or transactional approach would prevent orphaned state. Common in CLI tools that create multiple resources.

---

---

## Finding: Start() testability barrier is interface absence, not exec.Command

- **Confidence**: HIGH
- **Evidence**:
  - `start.go` has **0 direct exec.Command calls**. It delegates to named functions: `SessionExists`, `NewSession`, `SendKeys`, `NewWindow` (tmux), `CreateWorktree`, `hasDirtyWorktree` (git), `GenerateAdapter`, `EnsureEngineTrusted`, `RenderMasterProtocol`, `RenderSubagentProtocol` (filesystem).
  - All 24 exec.Command calls are concentrated in just 3 files: `tmux.go` (9), `worktree.go` (9), `adapter.go` (2), `review.go` (2), `diff.go` (2).
  - `start.go`'s dependencies categorize as:
    - 4 tmux functions (require tmux binary)
    - 2 git functions (require real git repo)
    - 5 filesystem functions (testable with temp dirs)
    - 3 pure functions (testable as-is)
  - If tmux.go and worktree.go functions were behind a Go interface, `Start()` could be tested with a mock — no tmux or git required. The filesystem operations (protocol rendering, trust, adapter) already work with temp directories (proven by `trust_test.go` and `protocol_test.go`).
- **Counter-evidence**: Introducing an interface for 9 tmux functions + 8 git functions is overhead. The project is a personal tool with ~30 tests that all pass. The pragmatic cost of the interface may exceed its value.
- **Implication**: The testability barrier is architectural (no interfaces), not technical (exec.Command). A single `Runner` or `Platform` interface would unlock testing the entire Start() flow, but the ROI depends on how often Start() bugs occur in practice.

---

## Finding: report.md.tmpl is dead code with undefined fields

- **Confidence**: HIGH
- **Evidence**:
  - `report.md.tmpl` references `{{.JournalSummary}}` — a field that does NOT exist on `SessionData`:
    ```go
    type SessionData struct {
        Name, WindowName, WorktreePath, JournalPath, GuidancePath, EngineCommand, Prompt string
    }
    ```
  - `report.md.tmpl` also references `{{.Name}}` at the top level — but `ProtocolData` has no `Name` field (it has `SessionName`).
  - No Go code calls `renderTemplate("templates/report.md.tmpl", ...)`. The `Report` command (`cli/report.go`) generates output directly from journal data without using the template.
  - The template is embedded via `//go:embed templates/*.tmpl` in `templates.go`, so it increases binary size despite being unused.
  - Verified: `grep -rn 'report.md.tmpl' --include='*.go'` returns no results.
- **Counter-evidence**: It could be intended for future use. However, its current field references are wrong for any existing struct.
- **Implication**: Dead code that would produce wrong output if ever used. Should be either fixed to match actual structs or removed.

---

## Finding: Security posture is adequate (no injection risks)

- **Confidence**: HIGH
- **Evidence**:
  - All user-controlled input (objective text) flows through `Slugify()` before reaching shell contexts. Slugify strips everything except `[a-z0-9]` and hyphens.
  - Tmux session names: `TmuxSessionName()` → `ProjectID()` → `slugify()` → safe chars only.
  - Launch commands use Go's `%q` formatting for prompt strings: `fmt.Sprintf("%s %q", masterCmd, masterPrompt)` — this escapes special characters.
  - `exec.Command` is used with argument arrays (not shell strings), preventing shell injection.
  - Heartbeat command uses backtick Go string with `%s` for session name, but session name is slugified.
- **Counter-evidence**: `HeartbeatCommand()` builds a shell one-liner that's executed via `SendKeys()`. If the tmux session name somehow contained a `'` or `;`, it could break the shell command. However, slugify guarantees this won't happen.
- **Implication**: No actionable security issues found. The design is safe-by-construction through the slugify sanitization layer.

---

## Finding: No TOCTOU race in conflict detection but no atomicity either

- **Confidence**: MEDIUM
- **Evidence**: `start.go:45-49`:
  ```go
  if _, err := os.Stat(runDir); err == nil {
      return "run directory already exists"
  }
  if SessionExists(tmuxSess) {
      return "tmux session already exists"
  }
  ```
  Between the check and the creation at line 70 (`os.MkdirAll`), another process could create the same directory. However:
  - The tool is single-user, single-invocation
  - `os.MkdirAll` is idempotent — it won't fail if the dir exists
  - The real protection is the tmux session name uniqueness (tmux itself rejects duplicates)
- **Counter-evidence**: In practice, two `goalx start` commands would need to race within milliseconds, and even then, the second tmux session creation would fail harmlessly.
- **Implication**: Theoretical concern. Not worth fixing for a personal tool.

---

---

## Finding: Master restart protocol has missing C-c step

- **Confidence**: HIGH
- **Evidence**: In `master.md.tmpl`, the restart instruction for dead/unresponsive subagents says:
  ```
  **Dead or unresponsive (2+ heartbeats with no progress):**
  - Write accumulated context to guidance file.
  - Restart the subagent.

  Restart command: `tmux send-keys -t SESSION:WINDOW "ENGINE_CMD 'PROMPT'" Enter`
  ```

  But there's no `C-c` step to kill the existing process before sending the restart command. Compare with the "objective achieved" case which correctly includes:
  ```
  Stop the relevant subagent: `tmux send-keys -t SESSION:WINDOW C-c`
  ```

  If the subagent is stuck (alive but unresponsive), the restart command gets typed into the stuck Claude CLI's input prompt, producing garbled output instead of starting a fresh instance.
- **Counter-evidence**: The condition "dead or unresponsive" may imply the process has already exited (shell prompt visible). The master AI should be smart enough to C-c first if needed. However, the template doesn't make this explicit, and a less capable model could miss it.
- **Implication**: The restart command should prepend `C-c` to ensure clean restart regardless of process state.

---

## Finding: File protection in develop mode is honor-system only

- **Confidence**: HIGH
- **Evidence**:
  - Protocol template says `**You CANNOT modify:** {readonly files}` but this is text instruction only
  - Subagent engines run with full write access: `--permission-mode auto` (Claude) and `--full-auto` (Codex)
  - No git pre-commit hooks are generated to enforce readonly constraints
  - No filesystem permissions are set on readonly files in worktrees
- **Counter-evidence**: Worktree isolation provides defense-in-depth:
  - Each subagent works on an isolated branch
  - Merge uses `--ff-only` (safe, no force)
  - User reviews via `goalx review` / `goalx diff` before `goalx keep`
  - Even if a subagent modifies readonly files, the damage is confined to a disposable branch
- **Implication**: The worktree-as-sandbox design makes this acceptable for a personal tool. Adding a pre-commit hook for readonly enforcement would be a low-effort hardening option.

---

## Finding: Heartbeat pileup is possible but unlikely

- **Confidence**: MEDIUM
- **Evidence**: Heartbeat sends "Heartbeat: execute check cycle now." to master window every `check_interval` seconds. If master takes longer than `check_interval` to process a check cycle, heartbeat messages queue in tmux's input buffer. When master finishes, it processes queued heartbeats back-to-back, causing rapid re-checks.
  - Default check_interval: 5 minutes
  - Minimum enforced: 300 seconds (same as default — the floor code is actually a no-op for default config)
  - A check cycle involves: reading journal, git log, running harness, AI evaluation
  - With AI response time, this likely takes 1-3 minutes
- **Counter-evidence**: At 5-minute intervals, pileup requires the master to take >5 minutes per check. This is unlikely for the typical check cycle. Even if pileup occurs, the worst case is extra re-checks, which is annoying but not dangerous.
- **Implication**: Not a problem at default settings. Would become an issue if check_interval were shortened (but the silent 300s floor prevents this).

---

---

## Finding: Templates are well-designed — engine-agnostic with proper parameterization

- **Confidence**: HIGH
- **Evidence**:
  - 306 total template lines (master: 127, program: 161, report: 18)
  - 55 template variable references (`{{...}}`): 28 in master, 27 in program
  - Templates contain ZERO engine-specific references (no "claude", "codex", "aider")
  - All engine-specific behavior is resolved at render time via `.EngineCommand` and `.Prompt`
  - Adding a new engine requires NO template changes — only Go code in:
    1. `config.go` BuiltinEngines (5 lines)
    2. `cli/adapter.go` GenerateAdapter (guidance notification — the ONLY engine-specific hook point)
    3. `cli/trust.go` EnsureEngineTrusted (workspace trust bootstrap)
  - Template conditionals handle mode variations cleanly: `{{if eq .Mode "research"}}` / `{{else}}`
  - Budget/context/diversity sections use `{{if}}` guards to skip when empty
- **Counter-evidence**:
  - `program.md.tmpl:34` hardcodes "Run tests/benchmarks: `go test`, `go build`" — this is Go-specific guidance that won't apply to non-Go projects. However, it's in the "suggested methods" section, not a directive.
  - No `missingkey=error` template option set, so undefined fields silently render as empty strings rather than erroring. This is a trade-off: resilient but harder to debug.
- **Implication**: The template design follows the "协议即引擎" principle well. The templates are the framework's best architectural achievement.

---

## Finding: No signal handling — SIGINT during Start() causes same orphan state as error

- **Confidence**: HIGH
- **Evidence**:
  - Zero signal handling code in entire codebase: `grep -rn 'os.Signal\|SIGTERM\|Interrupt' --include='*.go'` returns nothing
  - Ctrl+C during `goalx start` causes immediate process termination
  - Any side effects created before the signal (dirs, worktrees, tmux sessions) are orphaned
  - This is the same symptom as the "no cleanup on Start() failure" finding but with a different trigger
  - The tmux session is created with `-d` (detached), so it survives terminal death — this is correct for normal operation but means orphaned tmux sessions persist after crashes
- **Counter-evidence**:
  - `goalx drop` cleanly recovers from orphaned state (tested by `TestDropRemovesRunDirectoryAndBranch`)
  - For a personal CLI tool, "run goalx drop" is a reasonable recovery procedure
  - Go CLI tools commonly skip signal handling when `defer` cleanup is sufficient
- **Implication**: Combined with the partial-failure orphan issue, this argues for a `defer cleanup` pattern in `Start()` that runs when the function exits abnormally.

---

## Finding: Multi-run is fully supported and conflict-free by design

- **Confidence**: HIGH
- **Evidence**:
  - Each run gets unique paths via `RunDir(projectRoot, name)` → `~/.autoresearch/runs/{projectID}/{name}/`
  - Each run gets unique tmux session via `TmuxSessionName(projectRoot, name)` → `goalx-{projectID}-{name}`
  - Each run gets unique branches via `goalx/{name}/{N}`
  - Worktree paths are under the run directory, preventing overlap
  - `goalx list` correctly shows all runs for the current project
  - Same-name conflict is caught at start.go:45 (`os.Stat(runDir)`)
  - Cross-project isolation is guaranteed by `ProjectID()` which includes the full project path
- **Counter-evidence**: None found. The naming scheme is well-designed.
- **Implication**: This is a strength. The framework correctly supports parallel exploration runs.

---

## Finding: Heartbeat is a shell loop, not a Go goroutine — correct design

- **Confidence**: HIGH
- **Evidence**:
  - CLAUDE.md says "heartbeat goroutine" but the actual implementation is a shell `while sleep` loop in a tmux window, NOT a Go goroutine
  - `heartbeat.go:9`: `while sleep %d; do tmux send-keys -t %s:master 'Heartbeat: ...' Enter; done`
  - This runs as a shell process in the tmux "heartbeat" window
  - Zero goroutines, zero channels, zero sync primitives in the entire codebase
  - The `while sleep N` pattern guarantees first heartbeat fires N seconds after start — no race with master startup
- **Counter-evidence**: CLAUDE.md line "heartbeat goroutine: 每 check_interval send-keys → master" is misleading documentation — it's not a goroutine.
- **Implication**: The implementation is actually simpler and more robust than a goroutine approach. Documentation should be updated to say "heartbeat process" instead of "goroutine."

---

---

## Finding: Template-struct field coverage is 100% for active templates, broken for dead template

- **Confidence**: HIGH
- **Evidence**: Exhaustive field-by-field comparison:
  - **master.md.tmpl**: All 15 template variables map to valid ProtocolData/SessionData fields. No mismatches.
    - Top-level: `.Objective`, `.TmuxSession`, `.SummaryPath`, `.AcceptancePath`, `.MasterJournalPath`, `.EngineCommand`, `.Budget.MaxDuration` — all on ProtocolData
    - Inside `{{range .Sessions}}`: `.Name`, `.WindowName`, `.WorktreePath`, `.JournalPath`, `.GuidancePath`, `.EngineCommand`, `.Prompt` — all on SessionData
    - `$.TmuxSession`, `$.Harness.Command` — parent context access, correct
  - **program.md.tmpl**: All 12 template variables map correctly. `.Objective`, `.Mode`, `.JournalPath`, `.GuidancePath`, `.Target.Files`, `.Target.Readonly`, `.Harness.Command`, `.Context.Files`, `.DiversityHint`, `.Budget.MaxRounds`, `.Budget.MaxDuration` — all on ProtocolData
  - **report.md.tmpl**: 2 undefined fields:
    - `{{.Name}}` — ProtocolData has `SessionName`, not `Name`
    - `{{.JournalSummary}}` — does not exist on SessionData
  - All paths are fully parameterized — zero hardcoded absolute paths
  - Zero engine-specific references in any template (no "claude", "codex", "aider")
- **Counter-evidence**: report.md.tmpl is dead code (never rendered), so the mismatches have zero runtime impact.
- **Implication**: Active templates are correct. Dead template should be fixed or removed.

---

## Finding: No template content duplication — master and program are complementary

- **Confidence**: HIGH
- **Evidence**:
  - master.md.tmpl describes the **master's view**: how to read journals, write guidance, evaluate progress, decide actions
  - program.md.tmpl describes the **subagent's view**: how to read guidance, write journals, do research/development
  - Journal format is defined DIFFERENTLY in each:
    - Master: `{"ts":"...","action":"...","session":"...",...}` (master.md.tmpl:126)
    - Subagent: `{"round":1,"commit":"...","question":"...",...}` (program.md.tmpl:120)
  - Guidance protocol is described from opposite perspectives:
    - Master: "write specific guidance to the session's guidance file" (line 95)
    - Subagent: "check for guidance... read it carefully... follow it" (lines 136-156)
  - Status values `ack-guidance` appears in both (master checks for it, subagent writes it) — this is protocol coherence, not duplication
- **Counter-evidence**: None. The two-party protocol design is clean.
- **Implication**: No refactoring needed. The templates are correctly structured as complementary protocol definitions.

---

## Finding: Adapter `\n` escape bug verified with precision

- **Confidence**: HIGH
- **Evidence**: Traced the exact data flow:
  1. **Go source** (`adapter.go:43-45`): Uses backtick string (raw literal). The `\n` is two characters: `\` + `n`
  2. **JSON output** (hooks.json): `json.Marshal` produces `\\n` (JSON-escaped backslash + n). Additionally, `>` becomes `\u003e` and `&&` becomes `\u0026\u0026` due to Go's default HTML-safe JSON encoding
  3. **Shell execution**: Claude Code reads hooks.json, decodes JSON, executes: `echo '\n⚠️ MASTER GUIDANCE PENDING...'`. In bash, single-quoted strings are literal, so `echo '\n...'` outputs the literal characters `\n` followed by the message
  4. **Verified by running**: `echo '\n⚠️ TEST'` outputs `\n⚠️ TEST` — confirmed literal `\n` in output
  - The JSON HTML-encoding (`\u003e`, `\u0026`) is correctly decoded by `json.Unmarshal` and is NOT a bug
  - Only the `\n` literal is a bug — intent was a newline for visual separation
- **Counter-evidence**: None. The bug is definitively confirmed through the complete data flow.
- **Implication**: Fix by using `printf '\\n...'` (printf interprets `\n`), `echo '' && echo '...'`, or simply removing the `\n`.

---

## Consolidated Summary: All Findings by Priority

### Bugs (should fix)

| # | Issue | File:Line | Confidence | Effort |
|---|-------|-----------|-----------|--------|
| B1 | No cleanup on partial `Start()` failure — orphans dirs/worktrees/tmux | cli/start.go:14-269 | HIGH | Low |
| B2 | `report.go:23` says "ar" not "goalx" (rename miss) | cli/report.go:23 | HIGH | Trivial |
| B3 | Adapter hook echo has literal `\n` instead of newline | cli/adapter.go:44 | HIGH | Trivial |
| B4 | Master restart command missing `C-c` kill step before restart | templates/master.md.tmpl:106 | HIGH | Trivial |
| B5 | `init.go:70` uses raw nanoseconds (6 * 3600_000_000_000), not `6 * time.Hour` | cli/init.go:70 | HIGH | Trivial |
| B6 | No signal handling — SIGINT during Start() orphans resources (same root cause as B1) | cli/start.go | HIGH | Low |

### Dead Code

| # | Issue | File | Confidence | Effort |
|---|-------|------|-----------|--------|
| D1 | `report.md.tmpl` never rendered, references undefined fields (`.Name`, `.JournalSummary`) | templates/report.md.tmpl | HIGH | Trivial |

### Design Issues (consider fixing)

| # | Issue | File:Line | Confidence | Effort |
|---|-------|-----------|-----------|--------|
| A1 | `start.go` god function: 269 lines, 1 func, 15 sequential side effects, 0 exec.Command (delegates to tmux.go/worktree.go) | cli/start.go | HIGH | Medium |
| A2 | CLI test coverage 28.9% — 24 exec.Command calls in tmux.go (9) + worktree.go (9) + 6 others, no interface abstraction | cli/ | HIGH | High |
| A3 | Silent heartbeat floor: `< 30s` silently becomes `300s` with no warning | cli/start.go:234-237 | HIGH | Low |
| A4 | `config.go` at 513 lines (convention ≤ 500), 22 functions spanning 5 responsibilities | config.go | HIGH | Low |
| A5 | Suppressed errors: `os.UserHomeDir()` (config.go:172,414), `filepath.Abs()` (start.go:86) | config.go, cli/start.go | HIGH | Low |
| A6 | File protection is advisory only — worktree isolation provides defense-in-depth | templates/program.md.tmpl | HIGH | Low |
| A7 | CLAUDE.md says "heartbeat goroutine" but it's a shell process in tmux | CLAUDE.md | HIGH | Trivial |
| A8 | `program.md.tmpl:34` hardcodes `go test` / `go build` as examples | templates/program.md.tmpl:34 | LOW | Trivial |

### Strengths (well-designed, keep as-is)

| # | Aspect | Evidence |
|---|--------|---------|
| S1 | Templates are fully engine-agnostic | 0 engine refs in 306 template lines; new engine = 0 template changes |
| S2 | Template-struct field coverage is 100% for active templates | 27 fields checked, all match ProtocolData/SessionData |
| S3 | No template duplication | Master and program define complementary protocol views |
| S4 | Multi-run is conflict-free by design | ProjectID + RunDir + TmuxSessionName = unique per project+name |
| S5 | Security via slugify sanitization | All shell-facing input passes through `[a-z0-9-]` filter |
| S6 | Heartbeat as shell loop (not goroutine) | Simpler, more robust, no race with master startup |
| S7 | Worktree-as-sandbox isolation model | ff-only merge + user review = defense-in-depth |
| S8 | mergeConfig explicit field-by-field | Catches type-specific zero-value bugs that reflection misses |
