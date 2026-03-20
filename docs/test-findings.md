# GoalX 完整链路测试发现 (2026-03-20)

测试路径: Research (Round 1) → Debate (Round 2) → Develop → Keep

---

## A. Protocol 问题

### A1. Master 只能被动等 heartbeat，无法主动干预 subagent
- **现象**: subagent 做完在 idle，master 写了 guidance 但 subagent 不会去读 → 死锁
- **根因**: heartbeat 是 shell `while sleep N` 循环，只触发 master。subagent 没有被动接收机制
- **影响**: develop 模式下 subagent 几分钟做完，master 要等 3 个 heartbeat (15min) 才能确认
- **方案**: master 写完 guidance 后，直接 `tmux send-keys` 给 subagent 发一条提示（如 "Master guidance available — read it now"），而不是等 subagent 下次自发检查

### A2. Develop 模式 heartbeat 间隔过长
- **现象**: subagent 3 分钟做完，master 等 5 分钟才 check
- **方案**: `goalx init` 根据 mode 设不同默认值：research=5m, develop=2m

### A3. Subagent 做完所有任务后不检查 guidance 就 idle
- **现象**: subagent 完成任务列表后停在 `❯` prompt，不再检查 guidance
- **方案**: program.md.tmpl 加指令："完成所有已知任务后，检查一次 guidance，如果为空则报告 done 并等待"

### A4. Master 写文件偶尔失败 (Error writing file)
- **现象**: master 第一次 Write acceptance.md 报错，重试后成功
- **根因**: 可能是 `~` 路径解析问题，Claude Code 有时不认 `~`
- **方案**: master.md.tmpl 中所有路径用绝对路径（已通过模板变量实现），但 master agent 自己写文件时也会用 `~`。考虑在 protocol 中提醒 "use absolute paths, not ~"

### A5. Journal 字段不匹配（已修复）
- research protocol 让 agent 写 `"question"` + `"confidence"`，但 JournalEntry 只有 `"desc"`
- **已修复**: P0-1 in develop run

---

## B. Skill 问题

### B1. Skill 命令应该更智能
- **现象**: `/goalx status --run NAME` 以前不工作（已修 U1），但 skill 里的示例还写 `--run`
- **方案**: skill 统一用位置参数 `goalx status NAME`

### B2. Skill 缺少观测能力
- **现象**: 用户反复说"看看状态"，每次我都要手动跑 status + tmux capture-pane
- **方案**: skill 加一个 `/goalx watch` 或 `/goalx observe` 命令，自动跑 status + capture 所有 pane 的最近输出 + 返回格式化摘要

### B3. Skill 缺少完整链路命令
- **现象**: research→debate→develop→keep 每步都要手动操作
- **方案**: skill 文档里加 workflow 示例，或加 `/goalx next` 建议下一步操作

### B4. tmux attach vs switch-client（已修复）
- skill 已输出两种命令让用户选

---

## C. Code/UX 问题

### C1. `goalx keep` 只支持 ff-only merge
- **现象**: main 有新 commit 后 worktree 分支分叉，keep 失败
- **方案**: 尝试 ff-only，失败时 fallback 到 `--no-ff` merge，或提示用户选择

### C2. Dirty worktree 检查过严
- **现象**: untracked files (bin/, ar) 被当 dirty，阻止 merge
- **已修复**: .gitignore

### C3. Master journal 显示格式
- **现象**: `goalx status` 的 master summary 截断过长
- **方案**: summary 截断到合理长度 + 尾部加 `...`

### C4. Subagent 触发 skill（已修复）
- program.md.tmpl 已加 "Do NOT invoke any slash commands or skills"

### C5. Binary 过期问题
- **现象**: config.go 改了但 bin/goalx 未重编译
- **方案**: CLAUDE.md 加提醒，或加 Makefile

---

## D. Protocol 设计改进

### D1. Master 主动干预能力（A1 的具体设计）
当前:
```
heartbeat → master check → write guidance → wait next heartbeat → subagent maybe reads
```
改进:
```
heartbeat → master check → write guidance → tmux send-keys "CHECK_GUIDANCE" to subagent → subagent reads immediately
```
master.md.tmpl 加指令: "写完 guidance 后，立即发送提醒:
`tmux send-keys -t {session} 'Master wrote guidance. Read {guidance_path} now and follow it.' Enter`"

### D2. Subagent 完成信号
当前: subagent 做完就 idle，master 不知道
改进: program.md.tmpl 加 "完成所有任务后:
1. 检查 guidance 一次
2. 如果为空，写 journal status=done
3. 继续等待（master 可能有新任务）"

### D3. Develop 模式的 heartbeat 默认值
cli/init.go 中根据 mode 设不同 check_interval:
- research: 5m（思考密集）
- develop: 2m（改动快）

---

## E. 优先级排序

### 立即修复（影响核心体验）
1. **D1**: Master 主动发 guidance 提醒给 subagent（消除死锁）
2. **D2**: Subagent 完成信号（消除 idle 等待）
3. **D3**: Develop heartbeat 默认 2m
4. **C1**: goalx keep 支持 non-ff merge
5. **B2**: Skill 加 observe/watch 命令

### 应该修复（改善体验）
6. **A4**: Protocol 提醒用绝对路径
7. **B1**: Skill 示例统一位置参数
8. **B3**: Skill 加 workflow 示例和 next 建议

### 可选改进
9. **C3**: Status summary 截断优化
10. **C5**: 加 Makefile 或 build 提醒
