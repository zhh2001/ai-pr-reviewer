# 设计说明

文档基于当前 commit 的实际实现写成。引用的常量、文件名、行为均与代码对齐
（参见仓库根目录的核对清单）。

## 1. 系统架构

monorepo：

- `/backend` —— Go 1.22，模块路径 `github.com/zhh2001/ai-pr-reviewer/backend`，
  二进制 `cmd/server`。
- `/frontend` —— Vue 3 + Vite，Dev server 用 Vite 把 `/api/*` 代理到
  `http://localhost:8080`，所以前端代码里直接 `fetch('/api/...')`。

### 依赖方向

```text
                       +---------------+
                       |   httpserver  |  HTTP 边界 (handler / mux)
                       |    review.go  |
                       +-------+-------+
                               |
                       +-------v-------+
                       |   analyzer    |  并发编排 + 风险过滤
                       +-------+-------+
                               |
              +----------------+----------------+
              |                |                |
      +-------v------+ +-------v-------+ +------v-------+
      |   Github     | |     LLM       | |   (内置 net) |
      |  Fetcher     | |   Client      | |              |
      +-------+------+ +-------+-------+ +--------------+
              |                |
              +-------+--------+
                      |
              +-------v-------+
              |   pr (domain) |  纯类型 + 接口，零外部依赖
              +---------------+
```

- 接口（`pr.Fetcher` / `pr.Summarizer` / `pr.RiskDetector` /
  `pr.SuggestionGenerator`）声明在 domain 包 `internal/pr`，
  实现散落在外层包 `internal/github` 和 `internal/llm`。
- `internal/pr` 不引任何 SDK 类型，只放结构体和接口签名——它是依赖图的叶子，
  让其它包都能依赖它而不必反过来。
- 单个 `*llm.Client` 同时实现 `Summarizer / RiskDetector / SuggestionGenerator`
  三个接口（见 `internal/llm/{summary,risks,suggestions}.go`）：production 时一份
  HTTP client，三次调用，分别选不同模型；测试时 handler 用三组独立的 mock 注入，
  各自验证降级语义。

`internal/httpserver/review_test.go` 与 `internal/analyzer/analyzer_test.go`
能在不连 DeepSeek、不连 GitHub 的情况下走完全链路，靠的就是这个边界。

## 2. 模型选择

DeepSeek 提供 OpenAI 兼容接口，`base_url = https://api.deepseek.com`。三个子任务分两档：

| 通道 | 模型 | 为什么 |
| --- | --- | --- |
| `summary` | `deepseek-v4-flash` | 总结只需把"改了什么、为什么、影响面"说清，可读即可，对延迟敏感。 |
| `suggestions` | `deepseek-v4-flash` | 建议是增强性意见，多一条少一条不阻断 merge，可以容错。 |
| `risks` | `deepseek-v4-pro` | 风险要求结构化输出 + 行号定位 + 误报控制，准确性比延迟更重要。 |

把贵的推理预算留给"必须准"的通道，便宜的延迟敏感任务用 flash，是有意的分层而不是
为了用全 SKU。常量与原因都写在 `internal/llm/client.go` 的注释里。

### 结构化输出的稳定性

`risks` 与 `suggestions` 需要 JSON。两道保险：

1. `ResponseFormat = ChatCompletionResponseFormatTypeJSONObject`——
   API 层强制返回合法 JSON。
2. system prompt 里硬约束："只输出一个 JSON 对象，不要任何解释、不要 markdown
   代码块、不要前后缀。顶层结构必须是 `{"risks":[...]}` / `{"suggestions":[...]}`"。

只用其一不够：DeepSeek 没有第二条时偶发吐空白卡死；没有第一条时偶发把 JSON 包在
` ```json` 代码块里。解析端再加一层 `stripFences`（`internal/llm/parse.go`），
即便系统提示被忽略也能恢复。三层一起够稳。

`summary` 不走 `json_object`——它是自由文本，开 JSON 模式反而会逼模型把文本塞
`{"summary":"..."}` 里，多一道无意义的反序列化。

## 3. 上下文获取

### 拉取

`internal/github/fetcher.go`：

- `PullRequests.Get` 拿元信息（title / body / author / base / head）。
- `PullRequests.ListFiles` 拉变更文件列表，含 `patch`。`PerPage: 100`，循环到
  `resp.NextPage == 0` 为止——PR 文件数大于 100 时会跨页。
- 错误统一过一遍 `classifyFetchError`，对外只暴露 `*FetchError`（详见第 6 节），
  handler 层不引 go-github。

### 截断（字节预算）

`internal/llm/prompt.go` 的 `renderContext` 是 summary / risks / suggestions 共享
的拼装函数：先写头（PR 元信息），剩下的字节预算均分给每个文件的 patch，
单文件 patch 超过 `perFile = budget / numFiles` 字节就尾部截断、追加
`\n... [truncated]`。

三个任务用不同预算：

| 常量 | 字节 | 为什么 |
| --- | --- | --- |
| `MaxSummaryPromptBytes` | `32 * 1024` | 32KB 够把每个文件的 diff "意思"说清，再多对总结质量基本没收益。 |
| `MaxRisksPromptBytes` | `48 * 1024` | 风险要看完整的 patch 才能精准定位行号，截断越激进越容易漏报或误报。 |
| `MaxSuggestionsPromptBytes` | `32 * 1024` | 与 summary 同级。再多上下文对"可读性 / 命名 / 补测试"这种判断没有显著帮助。 |

### 当前局限

- **均分预算**对单文件巨改的 PR 不友好：一个 5KB diff + 一个 50KB diff，
  在 32KB 总预算下各分到 16KB，前者被多分了一倍空间不用，后者仍要被截断。
  贪心或按 `additions+deletions` 比例分配会更合理，但需要更多调参，先不上。
- **patch 是 byte-level 截断**：可能切在 hunk 中间，让模型看到半个上下文行。
  在 patch 层按 `\n@@` hunk 边界切会更友好，但实现要多写一层 parser。
- **整文件视野缺失**：模型只看 `git diff`，看不到被改函数的完整定义、调用方、
  类型定义。这是误报的主要来源之一。

### 未来方向

- **按比例 / 贪心分配预算**：替换 `perFile = budget / numFiles` 的均分逻辑。
- **仓库级 RAG**：把仓库索引成向量库，给 risks/suggestions 通道补关键符号定义
  / 调用方上下文，让"这里漏 nil 检查"之类的判断有真实代码做依据。
- **多轮检索**：先让模型扫一遍 patch 输出"我需要看 X 的实现"，再喂回去。
  对长 PR 比单轮塞 48KB 上下文更划算。

## 4. 误报 / 漏报控制

`risks` 的每条结果带 `confidence ∈ [0,1]`，由模型自评。`internal/analyzer/filter.go`
的纯函数 `filterRisksByConfidence` 保留 `confidence >= threshold` 的项，
返回新 slice（不就地改）。

阈值走环境变量 `RISK_CONFIDENCE_THRESHOLD`，默认 `0.5`。
解析：非法 / 空回落默认；越界（`1.5` / `-0.3`）夹紧到 `[0, 1]`。

`>=` 边界写进了测试（`TestFilterRisksByConfidence` 的 `boundary kept (>=)`
子用例）。阈值越高 → 召回越低、精度越高。

### 透明化

过滤掉的条数走 `ReviewResult.RisksFiltered`：

- detector 成功 + 确有项被剔除 → `risks_filtered: N` 出现在响应里
- detector 失败 或 没有项被剔除 → 字段 omit（`omitempty`）

前端 `RisksSection.vue` 在 `risks_filtered > 0` 时显示一行
"已过滤 N 条低置信度项"。调用方既看到当前精度档位的结果，也能直接知道"还有 N 条
被压住了"——降低门槛就能再调出来。

### 不做什么

- **没有规则黑名单**。不做"包含某 keyword 就过滤"这类拍脑袋的硬编码，
  全靠 confidence 一个连续旋钮，行为可解释。
- **没有跨 PR 的学习**。本期不维护"过去 N 次 review 中哪类风险被人忽略"的反馈环，
  没必要为 demo 做半个产品。

## 5. 响应速度

### 决策：WaitGroup 而不是 errgroup

三个 LLM 子任务（summary / risks / suggestions）相互独立。串行实现里耗时是
`T_summary + T_risks + T_suggestions`；并发后是 `max(...)`。

为什么不用 `golang.org/x/sync/errgroup`？因为 errgroup 的语义是"一败俱败"——
任一任务返回非 nil error 就 cancel 共享 ctx。这与本项目要求的"独立降级"直接冲突：
单通道失败时另外两个**必须继续跑完**，对应字段 `summary_error / risks_error /
suggestions_error` 各自带各自的错误，整体仍 200 返回。

所以用 `sync.WaitGroup` + 每任务独立的局部变量（`summary/summaryErr`,
`risks/risksErr`, `suggestions/sugErr`），主 goroutine 在 `wg.Wait()` 之后单线程
汇总进 `ReviewResult`。禁止多个 goroutine 写同一字段。

数据竞争靠 `go test -race ./...` 守门。其中 `TestAnalyze_RunsConcurrently` 用三方
rendezvous `sync.WaitGroup` 验证三任务确实并发起来（串行会卡死），不依赖时间断言
避免 flaky。

### 整体超时

`analyzer.New` 接收 `timeout`（来自 `Config.AnalyzeTimeout`，env
`ANALYZE_TIMEOUT_SECONDS`，默认 `90s`）。`Analyze` 内部用
`context.WithTimeout(ctx, timeout)` 派生一个共享 ctx 喂给三个 goroutine。

超时**只**触发 ctx 取消，**不**触发任务相互取消——三任务各自的调用收到
`ctx.Err()` 后正常返回错误，handler 走对应通道的降级路径。
`TestAnalyze_TimeoutMarksAllChannels` 锁住这条不变量。

### 量级

按 sum→max 的算法变化，以及 DeepSeek 公开的 flash / pro 典型延迟量级，
单次请求的墙钟时间应从大致 `8–16s` 降到 `4–8s`（取决于 risks 这条最慢通道）。

这里给的是**算法分析 + 量级估算**，不是当前 commit 上的基准。要拿到真实数字，
方法是：把 ANALYZE_TIMEOUT_SECONDS 调到足够大，重复请求一个固定 PR 计 `time curl`，
对比把 `analyzer.Analyze` 临时改成串行后的结果。这步留给本机有 DeepSeek key 的人
现场跑。

## 6. 健壮性

### 三通道独立降级

`Analyzer.Analyze` 装配 `pr.ReviewResult` 时：

```text
detector 成功 → result.Risks       = filter(risks, threshold)
                result.RisksFiltered = len(orig) - len(kept)
detector 失败 → result.RisksError  = err.Error()
                日志 log.Printf("detect risks pr %s/%s#%d: %v", ...)
```

summary / suggestions 同型。三者互不影响。这条契约由 4 个 handler 测试 +
2 个 analyzer 测试锁住：`TestReview_{Summarizer,Detector,Generator}Error`、
`TestAnalyze_Only{Summarizer,Detector,Generator}Fails`。

### 上游错误按状态码分

`internal/github/errors.go` 的 `classifyFetchError` 把 go-github 错误归到
`FetchErrorKind` 三类：

- `KindNotFound` —— `*gh.ErrorResponse` 且 status == 404
- `KindRateLimited` —— `*gh.RateLimitError` 或 `*gh.AbuseRateLimitError`
  （类型识别，不依赖 message 文本匹配）
- `KindUpstream` —— 其它（401 / 422 / 5xx / DNS / 连接断开）

handler `writeFetchError` 据此映射成 HTTP 状态：

| Kind | HTTP | 文案 |
| --- | --- | --- |
| KindNotFound | 404 | "PR 不存在或无权访问（GitHub 返回 404）" |
| KindRateLimited | 429 | "GitHub 限流，请稍后重试" |
| KindUpstream | 502 | `fetch pr: <上游原始消息>` |
| 解析失败 / 缺字段 | 400 | 具体 error |
| analyzer 错误 | 200 | 主体在 `*_error` 字段 |

前端 `App.vue` 顶部错误 banner 直接展示 status + message，对 404 / 429 用户可读
（"PR 不存在"、"限流"）；分通道错误在对应 section 内显示一行灰色降级提示，
不污染其它 section 的渲染。

## 7. 未来扩展

- **GitLab / Bitbucket**：抽 `pr.Fetcher` 接口的另一个实现（沿用本项目已有的边界）。
- **回写为 PR 评论**：拿到 ReviewResult 后调 GitHub `POST /repos/{}/{}/pulls/{}/reviews`，
  按文件 / 行号挂 inline comment（需要把 `Risk.Line / Suggestion.Line` 做行号有效性
  二次校验，模型偶尔给的行号会偏）。
- **Webhook / CI**：监听 GitHub `pull_request` 事件，自动跑 review、把结果
  贴回 PR；CI 中作为 status check。
- **结果缓存**：以 `(owner, repo, sha)` 为 key 缓存 ReviewResult，避免 PR 没改时
  重复烧 LLM token。
- **流式返回**：summary / suggestions 走 SSE，前端边收边渲染；risks 仍等齐 JSON 后
  整体返回（结构化输出难以增量解析）。
- **行级定位增强**：现在 `Risk.Line` 是模型自报；可叠一层"行号是否落在 diff 的
  `@@ -a,b +c,d @@` hunk 范围内"的校验，落不到的回退到 file-level（line=0）。
- **模型路由**：根据 PR 改动量自适应选模型——小 PR 全 flash，大 PR / 安全敏感
  路径全 pro。需要先攒一批人工标注的对比数据再做。
