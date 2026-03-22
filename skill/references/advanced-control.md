# Advanced Control

Use this only when the user explicitly asks for low-level GoalX control.

## Config-First Flow

Use the manual path when the user wants to inspect or edit config before launch:

```bash
goalx init "goal"
# edit .goalx/goalx.yaml if requested
# active runs use immutable run-spec.yaml snapshots; redirect them with `goalx tell`, `goalx add`, `goalx park`, or `goalx resume` instead of editing run files
goalx start
```

Do not choose this path by default. Prefer `goalx auto`.

## Mid-Run Intervention

Use these only when the user asks for manual control or the default autonomous path is not enough:

- Redirect the master with a short direction message
- Prefer `goalx tell --run NAME ...`; explicit `--run NAME` resolution is global when the name is unique, and `--run <project-id>/<run>` disambiguates collisions
- `goalx add --run NAME ...` to create a session manually
- `goalx park --run NAME session-N` to pause a session without losing its worktree
- `goalx resume --run NAME session-N` to restart a parked session
- `goalx keep --run NAME session-N` to merge a develop session branch

Do not micromanage subagents directly unless the user explicitly asks for that level of control.
