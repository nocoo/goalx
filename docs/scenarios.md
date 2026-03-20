# AutoResearch — 场景示例

> Master/Subagent 架构。框架做编排，AI 判断一切。
> 基于 QuantOS（quant_agent: 21 包, 352 源文件）。

---

## 场景 1: "quant_agent 要不要引入 event sourcing"

### Step 1: Research

3 个 subagent 从不同角度调研，master 监督报告质量。

```yaml
mode: research
objective: |
  调研 quant_agent 是否应该引入 event sourcing 架构：
  1. 分析现有数据流（react → memory → tools 的调用链）
  2. 评估 event sourcing 的适用性
  3. 给出 2-3 个具体方案，各自改动范围和风险
  4. 推荐一个方案并说明理由
engine: claude-code
parallel: 3

target:
  files: [".autoresearch/report.md"]
  readonly: ["quant_agent/", "pkg/", "quant_mono/"]

harness:
  command: "test -s .autoresearch/report.md && echo 'ok'"
  timeout: 30s

budget:
  max_duration: 6h

master:
  check_interval: 5m

diversity_hints:
  - "从性能角度分析，重点关注延迟和吞吐"
  - "从架构简洁性分析，重点关注复杂度和维护成本"
  - "从可扩展性分析，重点关注未来功能扩展"
```

Master 做什么：
- 每 5 分钟检查 subagent 进展
- 读报告判断是否全面（有没有遗漏关键方面？方案是否具体可操作？）
- subagent 崩溃 → 重启
- subagent 写了个水报告声称 done → master 判断不合格，重启并指出不足
- 三份报告都足够好 → 写 summary，停止

```bash
ar review   # 对比 3 份报告
ar keep session-1
ar drop session-2 session-3
```

### Step 2: Develop

基于选定方案实施。

```yaml
mode: develop
objective: |
  按照 .autoresearch/reports/session-1/report.md 的方案实施：
  实现轻量级 event bus，改造 react → memory 调用链。
engine: claude-code

context:
  files: [".autoresearch/reports/session-1/report.md"]

target:
  files: ["quant_agent/"]
  readonly: ["pkg/", "cmd/"]

harness:
  command: "go build ./quant_agent/... && go test ./quant_agent/... -count=1 -timeout=5m"
  timeout: 8m

budget:
  max_duration: 8h

master:
  check_interval: 5m
```

Master 做什么：
- 读 subagent journal + git log，了解进展
- 跑 harness 看 build/test 状态
- 判断代码是否真正实现了 event bus（不是只改了个名字就声称完成）
- subagent 卡在某个编译错误反复修不好 → master 重启并给方向
- 代码满足 objective → 写 summary，停止

---

## 场景 2: "重构精简 quant_agent 架构"

目标明确，直接 develop。Master 持续验证。

```yaml
mode: develop
objective: |
  重构精简 quant_agent/ 架构：
  - 合并重叠包，消除过深抽象，删除死代码
  - 包数量减少 30%+（当前 21 包→14 以下）
  - 全部现有测试通过
engine: claude-code
parallel: 2

target:
  files: ["quant_agent/"]
  readonly: ["pkg/", "cmd/", "quant_mono/", "quant_kernel/"]

harness:
  command: "go build ./quant_agent/... && go test ./quant_agent/... -count=1 -timeout=5m"
  timeout: 8m

budget:
  max_duration: 12h

master:
  check_interval: 5m

diversity_hints:
  - "激进方案：大幅合并包，目标 12 个以内"
  - "保守方案：最小改动，只消除明显重复"
```

Master 的价值：
- subagent 可能在第 3 轮就说"done"但只删了 2 个文件 → master 判断不合格
- subagent 可能改太猛把一半测试搞挂了又不知道怎么修 → master 重启并给反馈
- 两个 subagent 都真正完成 → master 写对比 summary

---

## 场景 3: "给 memory 系统加向量检索"

单 subagent，master 确保做到位。

```yaml
mode: develop
objective: |
  给 quant_agent/memory 包增加向量检索：
  1. Store 接口扩展 VectorSearch
  2. inmem.go 暴力余弦相似度实现
  3. manager_recall.go 增加向量检索路径
  4. 新增测试覆盖向量检索
engine: claude-code

target:
  files: ["quant_agent/memory/"]
  readonly: ["quant_agent/react/", "quant_agent/tools/", "pkg/"]

harness:
  command: "go build ./quant_agent/memory/... && go test ./quant_agent/memory/... -v -count=1"
  timeout: 3m

budget:
  max_duration: 4h

master:
  check_interval: 5m
```

没有 master 的话，subagent 可能只实现了接口定义就声称 done。
有 master：它会读代码验证四个点是否全部落地。

---

## 场景 4: "优化 evolution engine 性能"

用 develop 模式做性能优化。Master 判断是否真的更快了。

```yaml
mode: develop
objective: |
  优化 quant_agent/internal/evolution 包性能：
  - 运行 BenchmarkEngine 确认 baseline
  - pprof 分析热点
  - 优化，目标降低 30%+ ns/op
  - 全部测试通过
engine: claude-code

target:
  files: ["quant_agent/internal/evolution/"]
  readonly: ["quant_agent/memory/", "pkg/"]

harness:
  command: "go test -bench=BenchmarkEngine -benchtime=3s ./quant_agent/internal/evolution/... && go test ./quant_agent/internal/evolution/... -count=1"
  timeout: 5m

budget:
  max_duration: 8h

master:
  check_interval: 5m
```

---

## 场景 5: "react 包太复杂了"

先 research 出方案，再 develop 实施。

### Step 1

```yaml
mode: research
objective: |
  分析 quant_agent/react 包（40+ 文件）：
  1. 现有架构优劣
  2. 3 种精简方案
  3. 推荐最优
engine: claude-code
parallel: 3

target:
  files: [".autoresearch/report.md"]
  readonly: ["quant_agent/react/", "quant_agent/tools/", "quant_agent/memory/"]

harness:
  command: "test -s .autoresearch/report.md && echo 'ok'"

budget:
  max_duration: 4h

diversity_hints:
  - "合并方案：减少文件数"
  - "状态机方案：重新设计 loop"
  - "极简方案：最少文件实现同功能"
```

### Step 2

```yaml
mode: develop
objective: "按照选定方案精简 quant_agent/react"
engine: claude-code

context:
  files: [".autoresearch/reports/session-2/report.md"]

target:
  files: ["quant_agent/react/"]
  readonly: ["quant_agent/tools/", "quant_agent/memory/"]

harness:
  command: "go build ./quant_agent/react/... && go test ./quant_agent/react/... -v -count=1"

budget:
  max_duration: 8h
```

---

## 决策路径

```
模糊想法
  │
  ├─ 需要先调研？
  │   ├─ 是 → ar start "调研 X" --research --parallel 3
  │   └─ 否 → ar start "实现 X" --develop
  │
  ├─ 需要精确配置？
  │   ├─ 是 → ar init "..." → vim ar.yaml → ar start
  │   └─ 否 → ar start "..." 一句话启动
  │
  ├─ 有多种方案？
  │   ├─ 是 → --parallel N（ar init 后编辑 diversity_hints）
  │   └─ 否 → 默认 parallel: 1
  │
  └─ ar start → master 监督 subagent → 睡觉
     → ar review → keep / archive / drop
     → 需要接着 develop？→ ar init "实施 X" --develop --context 上一步报告
```

Master agent 的核心价值：**你睡着了，它替你盯着。** subagent 偷懒、崩溃、卡住，
master 引导或重启。你醒来看到的是真正达成目标（或接近目标 + 详细总结）的结果。
