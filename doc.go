// Package goalx provides an autonomous research and development framework.
//
// Master/Subagent architecture. Claude does brain (judgment, research, guidance),
// Codex does hands (coding, review). The framework only orchestrates
// (worktree, tmux, journal, heartbeat); all intelligence is in the protocol templates.
//
// # Design Philosophy
//
//   - Framework orchestrates, AI judges: Go code manages worktree/tmux/journal, AI decides if objective is met
//   - Protocol is the engine: master.md + program.md are the soul, not Go code
//   - Guide first, restart last: file-based guidance preserves subagent context
//   - Heartbeat-driven: framework timer triggers master checks, not AI self-loop
//
// # Inspiration
//
//   - lidangzzz/goal-driven: master/subagent + criteria verification + persistence
//   - OpenClaw: framework-driven heartbeat for periodic agent checks
package goalx
