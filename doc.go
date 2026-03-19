// Package autoresearch provides an autonomous research framework.
//
// AutoResearch has two personas:
//
//   - Level 1 (CLI tool): Pairs with AI coding agents (Claude Code, Codex, Aider)
//     to run autonomous experiments on any project. Manages tmux sessions,
//     generates research protocols, tracks results via journals.
//
//   - Level 2 (Embeddable SDK): Import into any Go project to add in-process
//     autonomous research capabilities. Host provides Artifact/Executor/Proposer
//     implementations; the SDK drives the propose→execute→measure→decide loop.
//
// Both levels share core abstractions: Experiment, Metric, Journal, Policy.
//
// # Design Philosophy
//
//   - Constraints enable autonomy: the framework only does what AI agents can't
//   - Protocol is the engine: program.md is the soul, not Go code
//   - Black-box targets: interact via files/commands/APIs, zero coupling
//   - Elegant, not complex: ~1000 lines of Go + declarative configs/templates
//
// # Inspiration
//
//   - karpathy/autoresearch: autonomous experiment loop (propose→train→measure→keep/revert)
//   - lidangzzz/goal-driven: goal-driven persistence with clear success criteria
package autoresearch
