# 设计说明

文档基于当前 commit 的实际实现写成。引用的常量、文件名、行为均与代码对齐
（参见仓库根目录的核对清单）。

## 1. 系统架构

monorepo：

- `/backend` —— Go 1.22，模块路径 `github.com/zhh2001/ai-pr-reviewer/backend`,
  二进制 `cmd/server`。
- `/frontend` —— Vue 3 + Vite，Dev server 用 Vite 把 `/api/*` 代理到
  `http://localhost:8080`，所以前端代码里直接 `fetch('/api/...')`。

### 依赖方向

```text
                       +---------------+
                       |   httpserver  |  HTTP 边界 (review.go / stream.go)
                       +-------+-------+
                               |
                       +-------v-------+
                       |   analyzer    |  并发编排（收集型 + 流式）+ 风险过滤
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

`internal/httpserver/review_test.go` / `stream_test.go` 与
`internal/analyzer/analyzer_test.go` 能在不连 DeepSeek、不连 GitHub 的情况下走完
全链路，靠的就是这个边界。

## 2. 模型选择

DeepSeek 提供 OpenAI 兼容接口，`base_url = https://api.deepseek.com`。三个子任务目前
都用 `deepseek-v4-flash`：

| 通道 | 模型 | 为什么 |
| --- | --- | --- |
| `summary` | `deepseek-v4-flash` | 总结只需把"改了什么、为什么、影响面"说清，可读即可，对延迟敏感。 |
| `suggestions` | `deepseek-v4-flash` | 建议是增强性意见，多一条少一条不阻断 merge，可以容错。 |
| `risks` | `deepseek-v4-flash` | 起初选 `deepseek-v4-pro` 追求准（结构化 JSON + 行号 + 误报控制）。真实联调里 pro 上的 risks 在 48KB 上下文 + 思维链下经常超过当时的 90s 分析超时，整段 risks 走 `risks_error` 降级丢掉。flash 默认开思考、推理质量已接近 pro 但显著更快——准确性轻微让步换稳定能返回的结果。 |

常量与取舍都写在 `internal/llm/client.go` 的注释里，并由
`TestModelConstants` 锁住"三档都是 flash"的现状。

### 结构化输出的稳定性

`risks` 与 `suggestions` 需要 JSON。两道保险：

1. `ResponseFormat = ChatCompletionResponseFormatTypeJSONObject`——
   API 层强制返回合法 JSON。
2. system prompt 里硬约束："只输出一个 JSON 对象，不要任何解释、不要 markdown
   代码块、不要前后缀。顶层结构必须是 `{"risks":[...]}` / `{"suggestions":[...]}`"。

只用其一不够：DeepSeek 没有第二条时偶发吐空白卡死；没有第一条时偶发把 JSON 包在
` ```json` 代码块里。解析端再加一层 `stripFences` 与"逐条 Unmarshal 跳过坏条"
（都在 `internal/llm/parse.go`）——即便系统提示被忽略、单条 item 字段类型不对，
也能保住其它条的可见性。

`summary` 不走 `json_object`——它是自由文本，开 JSON 模式反而会逼模型把文本塞
`{"summary":"..."}` 里，多一道无意义的反序列化。

## 3. 上下文获取

### 拉取

`internal/github/fetcher.go`：

- `PullRequests.Get` 拿元信息（title / body / author / base / head）。
- `PullRequests.ListFiles` 拉变更文件列表，含 `patch`。`PerPage: 100`，循环到
  `resp.NextPage == 0` 为止——PR 文件数大于 100 时会跨页。
- 错误统一过一遍 `classifyFetchError`，对外只暴露 `*FetchError`（详见第 7 节），
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

## 5. 延迟与性能

### 算法上的目标

三个 LLM 子任务相互独立。串行实现里耗时是 `T_summary + T_risks + T_suggestions`；
并发实现把它压成 `max(T_summary, T_risks, T_suggestions)`。配合**流式分批返回**
（见第 6 节），用户感知到的"首屏"进一步缩到 `T_fetch_PR ≈ 1.3-1.5s`——changes 就先
到了，summary / risks / suggestions 完成时各自再陆续到。

### 实测（本机 `.env`，`ANALYZE_TIMEOUT_SECONDS=90`，匿名 GitHub，真实 DeepSeek key）

| PR | files | 实测墙钟 | 各通道结果 |
| --- | --- | --- | --- |
| `sashabaranov/go-openai#1000` | 1 | **6.88s** | 三通道全成功；summary 168 字 / 1 risk / 2 suggestion |
| `sashabaranov/go-openai#1000`（重测） | 1 | 23.11s | 三通道全成功 |
| `sashabaranov/go-openai#1000`（再测） | 1 | 91.23s | 三通道**全 timeout**（DeepSeek 端瘫一会儿） |
| `spf13/cobra#2000` | 1 | 3.99s | 三通道全成功；summary 121 字 / 0 risk / 0 suggestion |
| `spf13/cobra#2000`（重测） | 1 | 305.18s | 三通道全 timeout |
| `kubernetes/kubernetes#10000`（v4-flash） | 61 | **9.75s** | 三通道全成功；summary 314 字 / 1 risk / 5 suggestion |
| `kubernetes/kubernetes#10000`（曾用 v4-pro） | 61 | 91.37s | summary + suggestions 回了，**risks timeout** |

从上面这张表能读出三件事：

1. **risks 切到 flash 是有意义的**：k8s 61 文件那条，v4-pro 在 90s 里 risks 拿不回来，切 flash 后 9.75s 全回了——同样工作负载、同样超时档位，结果完全不同。
2. **DeepSeek 单次延迟波动极大**：同一个 1 文件 PR（cobra#2000）一次 4 秒、另一次 305 秒 timeout；同一个 go-openai#1000 一次 7 秒、另一次 91 秒 timeout。这是**外部服务的不稳定**，不是本系统的问题——本机进程、本机 fetch、本机解析都不会有这种 100 倍量级的抖动。
3. **大 PR ≠ 慢**：61 文件的 k8s PR（9.75s）反而比 1 文件的 cobra PR 的某些重测（305s timeout）快，因为 LLM 端的延迟主要由它当时的排队 / 思考长度决定，和我们这里塞进去的 patch 大小不成简单正比。

### 系统怎么兜底外部抖动

- **180s 整体超时**（`DefaultAnalyzeTimeout`）。原来 90s 偶发不够，从联调数据观察 p99 后调高到 180s。常量由 `TestDefaultAnalyzeTimeout` 锁住。
- **三通道独立降级**：任一通道超时只让自己那条走 `*_error`，另两条照常返回。k8s 那次曾用 v4-pro 的跑就是这条机制——risks timeout 不影响 summary + suggestions。
- **流式分批返回**（第 6 节）：哪怕 risks 真的扛不住，前端在 ~1.5s 就拿到 changes 开始渲染，summary / suggestions 完成时各自渲染。用户不会盯着空白等 90 秒。

### 决策：WaitGroup 而不是 errgroup

`golang.org/x/sync/errgroup` 的语义是"一败俱败"——任一任务返回非 nil error 就 cancel 共享 ctx。这与本项目"独立降级"的契约直接冲突。

所以两个编排都用 `sync.WaitGroup`：

- 收集型 `Analyze`：每任务独立局部变量 + `wg.Wait()` 之后单线程汇总进 `ReviewResult`。
- 流式 `AnalyzeStream`：三个 worker 把 `Event` 写到 buffered(3) channel，单消费方
  按到达顺序读出来；channel 操作天然 atomic，不需要额外锁。

数据竞争靠 `go test -race ./...` 守门。`TestAnalyze_RunsConcurrently` 用三方 rendezvous
`sync.WaitGroup` 锁住"真的并发起来了"（串行会卡死），不依赖时间断言避免 flaky。

### 整体超时

`analyzer.New` 接收 `timeout`（来自 `Config.AnalyzeTimeout`，env
`ANALYZE_TIMEOUT_SECONDS`，默认 `180s`）。两个编排都用
`context.WithTimeout(ctx, timeout)` 派生一个共享 ctx 喂给三个 goroutine。

超时**只**触发 ctx 取消，**不**触发任务相互取消——三任务各自的调用收到
`ctx.Err()` 后正常返回错误，handler 走对应通道的降级路径。
`TestAnalyze_TimeoutMarksAllChannels` 锁住这条不变量。

## 6. 流式架构

`POST /api/review/stream` 在 `internal/httpserver/stream.go` 实现，与 `POST /api/review` 共用 `Analyzer`，区别只在编排和写出方式。

### 协议：NDJSON

一行一个 JSON 对象，写一行 Flush 一次。响应头：

```
Content-Type: application/x-ndjson
Cache-Control: no-cache
X-Accel-Buffering: no
```

`X-Accel-Buffering: no` 让 nginx 这类反向代理别再二次缓冲；vite dev server 实测也
是直通转发，不需要任何配置。

事件类型：

| `type` | 字段 | 何时发 |
| --- | --- | --- |
| `changes` | `changes: {…}` | 抓到 PR 元 + 文件列表后立刻发，作为流的第一条 |
| `summary` | `summary` 或 `error` | summary 通道完成 |
| `risks` | `risks`, `risks_filtered` 或 `error` | risks 通道完成（含阈值过滤） |
| `suggestions` | `suggestions` 或 `error` | suggestions 通道完成 |
| `done` | _(无)_ | 三个通道都结束后发出，作为流的最后一条 |

### Fetch 先于流

parse pr_url、检查 Flusher、调 `Fetcher.Fetch` 都在写任何 NDJSON 字节**之前**完成：

- parse 失败 → `400` + JSON 错误体
- fetch 失败 → 走 `writeFetchError` 映射的 `404 / 429 / 502` + JSON 错误体
- Flusher 不可用 → `500` + JSON 错误体

只有 fetch 成功（HTTP 200 即将写出）才会 `w.Header().Set` 切到 NDJSON、`WriteHeader(200)`、开始 emit。这条边界让流式端点的**错误状态码语义与非流端点完全一致**。`TestStream_FetcherErrorDoesNotStartStream` 锁住"fetch 失败时 body 不含任何 `"type":"changes"` / `"done"` 字符串"。

### 扇出 / 扇入

```text
   Analyzer.AnalyzeStream(ctx, changes) -> <-chan Event
              │
              │  ctx, _ := context.WithTimeout(ctx, 180s)
              │
   ┌──────────┼──────────┐
   │ go runSummary       │
   │ go runRisks         │ ── 三个 worker，每条完成就 out <- Event{Kind, …}
   │ go runSuggestions   │
   └──────────┬──────────┘
              │  wg.Wait(); close(out)
              ▼
        buffered(3) channel
              │
              ▼
   stream handler (单 goroutine)
     for ev := range stream { enc.Encode(eventToWire(ev)); flusher.Flush() }
     emit({"type":"done"})
```

- 3 个 worker 多写、handler 单读——Go channel 操作 atomic，不需要锁。
- handler 只负责"按到达顺序串行 Encode + Flush"——绝不让多个 goroutine 直接写 ResponseWriter，避免数据竞争和写到一半被打断。
- `wg.Wait()` 之后 `close(out)`，`range` 自然退出，最后 emit `done`。
- 跑 `go test -race ./...` 把上面整套路径覆盖了；`TestAnalyzeStream_*` 用 mock 接口验证事件总数、顺序无关性、过滤计数、单通道失败不影响其它通道。

### 流式下保留的独立降级

任一通道失败只在自己那条 event 上写 `error`：

```
{"type":"summary","summary":"..."}
{"type":"risks","error":"Post \"...\": context deadline exceeded"}
{"type":"suggestions","suggestions":[...]}
{"type":"done"}
```

`TestStream_DetectorErrorEmitsRiskErrorEventOnly` 把"detector 报错只让 risks 事件
带 error、summary / suggestions 不变"钉进单元测试。

### 前端渐进 UX

前端 `lib/ndjson.js` 把"分块 → 行 → 事件"的缓冲切分抽成纯函数 `splitLines`，
单独被 14 条 vitest 用例覆盖（单块多行 / 跨块拼接 / CRLF / 空行 / 末尾残行 / 等）。
`lib/stream-client.js` 用它驱动 `response.body.getReader() + TextDecoder`。

App.vue 收到事件后按 type 渐进写入 `result.value`：

| 事件到 | 浏览器画面变化 |
| --- | --- |
| 初始（loading + 无 result） | 整页 4-block Skeleton |
| `changes` | ChangesOverview 出 + ResultSummary 一行（`— risks · — suggestions · N files`）+ 三个 section 各自 kind-specific Skeleton |
| `summary` | Summary 区骨架→ markdown 文本（DOMPurify + markdown-it 双层消毒） |
| `risks` | Risks 区骨架→ severity pill + confidence bar + chip；ResultSummary 上 `—` 换成数字并画 severity 分布 |
| `suggestions` | Suggestions 区骨架→ 按 category 分组的列表 |
| `done` | loading 关，按钮恢复 |

### 实测时序（vite 代理透传）

经 `http://localhost:5173/api/review/stream` 打小 PR：

```
[+1.397s] type=changes      files=1
[+4.921s] type=summary      summary_len=208
[+5.564s] type=suggestions  n=2
[+9.420s] type=risks        n=1 filtered=0
[+9.441s] type=done
```

5 行陆续到达，间隔分别是 +3.5s / +0.6s / +3.9s / +0.02s——如果 vite 缓冲了，这 5 行会一起在 ~9.4s 时一齐到。**事实没有，证明 vite dev proxy 把 NDJSON chunk 直通转发**，浏览器端 `fetch().getReader()` 拿到的就是后端 `Flush()` 出去的字节。

## 7. 健壮性

### 三通道独立降级

`Analyzer.Analyze` 装配 `pr.ReviewResult` 时：

```text
detector 成功 → result.Risks       = filter(risks, threshold)
                result.RisksFiltered = len(orig) - len(kept)
detector 失败 → result.RisksError  = err.Error()
                日志 log.Printf("detect risks pr %s/%s#%d: %v", ...)
```

summary / suggestions 同型。三者互不影响。这条契约由 4 个 handler 测试 +
3 个 analyzer 测试锁住：`TestReview_{Summarizer,Detector,Generator}Error`、
`TestAnalyze_Only{Summarizer,Detector,Generator}Fails`。流式端点的对应不变量由
`TestStream_DetectorErrorEmitsRiskErrorEventOnly` 与
`TestAnalyzeStream_DetectorErrorIsolated` 锁住。

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
| analyzer 错误 | 200 | 非流：主体在 `*_error` 字段；流：通道 event 的 `error` 字段 |

前端 `App.vue` 顶部错误 banner 直接展示 status + message，对 404 / 429 用户可读；
分通道错误在对应 section 内显示一行灰色降级提示，不污染其它 section 的渲染。

## 8. 未来扩展

- **GitLab / Bitbucket**：抽 `pr.Fetcher` 接口的另一个实现（沿用本项目已有的边界）。
- **回写为 PR 评论**：拿到 ReviewResult 后调 GitHub `POST /repos/{}/{}/pulls/{}/reviews`，
  按文件 / 行号挂 inline comment（需要把 `Risk.Line / Suggestion.Line` 做行号有效性
  二次校验，模型偶尔给的行号会偏）。
- **Webhook / CI**：监听 GitHub `pull_request` 事件，自动跑 review、把结果
  贴回 PR；CI 中作为 status check。
- **结果缓存**：以 `(owner, repo, sha)` 为 key 缓存 ReviewResult，避免 PR 没改时
  重复烧 LLM token。
- **token-level 流式**：当前 NDJSON 是按通道粒度推（summary 完成一次推一整段）；
  可以进一步把 summary 改成 token 流（DeepSeek 支持 SSE），让长文本边生成边显示。
  risks / suggestions 仍维持整段 JSON 推送（结构化输出难以增量解析）。
- **行级定位增强**：现在 `Risk.Line` 是模型自报；可叠一层"行号是否落在 diff 的
  `@@ -a,b +c,d @@` hunk 范围内"的校验，落不到的回退到 file-level（line=0）。
- **模型路由**：根据 PR 改动量自适应选模型——小 PR 全 flash，大 PR / 安全敏感
  路径试 pro。需要先攒一批人工标注的对比数据再做。
