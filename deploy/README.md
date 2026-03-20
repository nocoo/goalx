# GoalX Deployment Guide

## 1. Install GoalX

```bash
go install github.com/vonbai/goalx/cmd/goalx@latest
# or build from source:
git clone https://github.com/vonbai/goalx.git && cd goalx
go build -o /usr/local/bin/goalx ./cmd/goalx
```

## 2. Configure

```bash
mkdir -p ~/.goalx
cp deploy/config.example.yaml ~/.goalx/config.yaml
# Edit: set bind IP, token, workspaces
```

## 3. Run as CLI (local use)

```bash
cd /your/project
goalx init "objective" --research --parallel 2
goalx start
goalx observe
```

## 4. Run as HTTP Server (remote management)

### Start manually:
```bash
goalx serve
```

### Start as systemd service:
```bash
sudo cp deploy/goalx-serve.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now goalx-serve
```

### Verify:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://YOUR_IP:9800/projects
```

## 5. OpenClaw Integration

GoalX can be managed by an OpenClaw agent (e.g. via Lark, Telegram, or Web UI).

### Install the GoalX skill on OpenClaw:

```bash
# Copy to OpenClaw workspace skills directory
cp -r skill/openclaw-skill /path/to/openclaw/workspace/skills/goalx
```

The skill teaches the OpenClaw agent to:
- Browse projects: `GET /projects`
- Start research: `POST /projects/:name/goalx/start`
- Check progress: `GET /projects/:name/goalx/observe`
- Change direction: `POST /projects/:name/goalx/tell`
- Modify config: `PUT /projects/:name/goalx/config`
- Add workspaces: `POST /workspaces`

### Network Requirements:

- GoalX server must be reachable from the OpenClaw host
- Recommended: bind to internal/private IP only
- GoalX should not bind to 0.0.0.0 (use internal IP)
- Bearer token authentication required

### Notification (optional):

Set `notification_url` in config to receive webhook when `goalx auto` completes:

```yaml
serve:
  notification_url: "https://your-openclaw-host/hooks/wake"
```

## 6. Claude Code Skill (local use)

```bash
mkdir -p ~/.claude/skills/goalx
cp skill/SKILL.md ~/.claude/skills/goalx/SKILL.md
```

Then use `/goalx` commands in Claude Code sessions.
