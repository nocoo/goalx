---
name: goalx
description: Use when Mochi needs to manage GoalX runs over the Tailscale-only HTTP API instead of local CLI access.
allowed-tools: "Bash"
metadata.openclaw: true
---

# GoalX HTTP for Mochi

GoalX runs on the `dev` host and is exposed over Tailscale only.
Use the HTTP API, not SSH.
Every request needs a Bearer token.

## Working Rules

- Keep replies short and Lark-friendly.
- Summarize results as `状态 / 关键信息 / 下一步`.
- Do not dump large JSON unless the user asks for raw output.
- Before mutating a run, say what you are about to change.

## Setup

```bash
export GOALX_PORT="18790"
export GOALX_BASE="http://100.110.196.103:${GOALX_PORT}"
export GOALX_TOKEN="REPLACE_ME"

goalx_get() {
  curl -fsS \
    -H "Authorization: Bearer $GOALX_TOKEN" \
    "$@"
}

goalx_post() {
  curl -fsS -X POST \
    -H "Authorization: Bearer $GOALX_TOKEN" \
    -H "Content-Type: application/json" \
    "$@"
}

goalx_put() {
  curl -fsS -X PUT \
    -H "Authorization: Bearer $GOALX_TOKEN" \
    -H "Content-Type: application/json" \
    "$@"
}

pretty() {
  python3 -m json.tool
}
```

If a request returns `401` or `403`, check the Bearer token first.
If a request times out, confirm you are on Tailscale and the server is bound to `100.110.196.103`, not `0.0.0.0`.

## Fast Flows

### 1. Browse projects

```bash
goalx_get "$GOALX_BASE/projects" | pretty
goalx_get "$GOALX_BASE/projects/my-project" | pretty
goalx_get "$GOALX_BASE/runs" | pretty
```

### 2. Start a new research or develop run

```bash
goalx_post "$GOALX_BASE/projects/my-project/goalx/auto" \
  -d '{
    "objective": "Investigate the regression in the sync pipeline",
    "mode": "research",
    "parallel": 2,
    "name": "sync-regression"
  }' | pretty
```

For a develop run, switch `mode` to `develop`.

```bash
goalx_post "$GOALX_BASE/projects/my-project/goalx/init" \
  -d '{
    "objective": "Fix the regression and add tests",
    "mode": "develop",
    "parallel": 2,
    "name": "sync-fix"
  }' | pretty

goalx_put "$GOALX_BASE/projects/my-project/goalx/config" \
  -d '{
    "target": {
      "files": ["cli/", "config.go", "cmd/goalx/main.go", "skill/"]
    },
    "harness": {
      "command": "go build ./... && go test ./... -count=1 && go vet ./..."
    }
  }' | pretty

goalx_post "$GOALX_BASE/projects/my-project/goalx/start" \
  -d '{"run":"sync-fix"}' | pretty
```

Use `auto` when Mochi should run the whole flow.
Use `init` + `config` + `start`
when the run needs a manual config pass first.

### 3. Watch progress

```bash
goalx_get "$GOALX_BASE/projects/my-project/goalx/status?run=sync-regression" | pretty
goalx_get "$GOALX_BASE/projects/my-project/goalx/observe?run=sync-regression" | pretty
```

Use `status` for a compact view.
Use `observe`
when you need the live master/session picture.

### 4. Change direction mid-run

```bash
goalx_post "$GOALX_BASE/projects/my-project/goalx/tell" \
  -d '{
    "run": "sync-regression",
    "message": "Focus on API retries first, not database indexing.",
    "session": "master"
  }' | pretty

goalx_post "$GOALX_BASE/projects/my-project/goalx/add" \
  -d '{
    "run": "sync-regression",
    "direction": "Investigate whether rate limiting changed last week."
  }' | pretty
```

Use `tell` to redirect an existing run.
Use `add`
to add a new angle without stopping the current work.

### 5. Read or update config

```bash
goalx_get "$GOALX_BASE/projects/my-project/goalx/config" | pretty

goalx_put "$GOALX_BASE/projects/my-project/goalx/config" \
  -d '{
    "objective": "Investigate retry failures in production",
    "parallel": 3,
    "target": {
      "files": ["cli/", "config.go", "cmd/goalx/main.go", "skill/"]
    }
  }' | pretty
```

Keep config edits minimal.
Change only the fields you intend to override.

### 6. Stop, save, keep, or clean up

```bash
goalx_post "$GOALX_BASE/projects/my-project/goalx/save" \
  -d '{"run":"sync-regression"}' | pretty

goalx_post "$GOALX_BASE/projects/my-project/goalx/keep" \
  -d '{"run":"sync-regression","session":"session-1"}' | pretty

goalx_post "$GOALX_BASE/projects/my-project/goalx/stop" \
  -d '{"run":"sync-regression"}' | pretty

goalx_post "$GOALX_BASE/projects/my-project/goalx/drop" \
  -d '{"run":"sync-regression"}' | pretty
```

Use `save` before `drop` if the results matter.
Use `keep`
when a specific session should be preserved or merged.

## Full Endpoint Reference

### Project discovery

```bash
goalx_get "$GOALX_BASE/projects"
goalx_get "$GOALX_BASE/projects/my-project"
goalx_get "$GOALX_BASE/runs"
```

### GoalX actions

```bash
goalx_post "$GOALX_BASE/projects/my-project/goalx/init" \
  -d '{
    "objective": "Investigate the regression in the sync pipeline",
    "mode": "research",
    "parallel": 2,
    "name": "sync-regression"
  }'

goalx_post "$GOALX_BASE/projects/my-project/goalx/start" \
  -d '{"run":"sync-regression"}'

goalx_post "$GOALX_BASE/projects/my-project/goalx/auto" \
  -d '{
    "objective": "Investigate the regression in the sync pipeline",
    "mode": "research",
    "parallel": 2,
    "name": "sync-regression"
  }'

goalx_get "$GOALX_BASE/projects/my-project/goalx/observe?run=sync-regression"
goalx_get "$GOALX_BASE/projects/my-project/goalx/status?run=sync-regression"

goalx_post "$GOALX_BASE/projects/my-project/goalx/tell" \
  -d '{
    "run": "sync-regression",
    "message": "Focus on API retries first, not database indexing.",
    "session": "master"
  }'

goalx_post "$GOALX_BASE/projects/my-project/goalx/add" \
  -d '{
    "run": "sync-regression",
    "direction": "Investigate whether rate limiting changed last week."
  }'

goalx_post "$GOALX_BASE/projects/my-project/goalx/stop" \
  -d '{"run":"sync-regression"}'

goalx_post "$GOALX_BASE/projects/my-project/goalx/save" \
  -d '{"run":"sync-regression"}'

goalx_post "$GOALX_BASE/projects/my-project/goalx/keep" \
  -d '{"run":"sync-regression","session":"session-1"}'

goalx_post "$GOALX_BASE/projects/my-project/goalx/drop" \
  -d '{"run":"sync-regression"}'

goalx_get "$GOALX_BASE/projects/my-project/goalx/config"

goalx_put "$GOALX_BASE/projects/my-project/goalx/config" \
  -d '{
    "objective": "Investigate retry failures in production",
    "mode": "develop",
    "parallel": 3,
    "target": {
      "files": ["cli/", "config.go", "cmd/goalx/main.go", "skill/"]
    },
    "harness": {
      "command": "go build ./... && go test ./... -count=1 && go vet ./..."
    }
  }'
```

## Lark Output Template

Use this shape after each important action:

```text
状态：已启动 / 进行中 / 已完成 / 已停止 / 失败
关键信息：run、当前阶段、是否需要人工决策
下一步：继续观察 / 改方向 / 保存 / keep / drop
```

Good example:

```text
状态：进行中
关键信息：run=sync-regression，master 已进入 research phase，2 个 session 都在工作
下一步：5-10 分钟后再看 status；如果 heartbeat 不变，再发 tell
```

## Safety Notes

- Use the Tailscale address, not a public IP.
- Always send the Bearer token.
- Prefer `status` before `observe`.
- Prefer `save` before `drop`.
- If the user asks for a redirect, use `tell` first; do not stop a healthy run unless asked.
