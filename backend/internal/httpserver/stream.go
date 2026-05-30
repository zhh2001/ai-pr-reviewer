package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/analyzer"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// streamHandler 实现 POST /api/review/stream。
// 协议：NDJSON（每行一个 JSON 对象）。事件类型：
//
//	{"type":"changes","changes":{...}}
//	{"type":"summary","summary":"..."}      或 {"type":"summary","error":"..."}
//	{"type":"risks","risks":[...],"risks_filtered":N} 或 {"type":"risks","error":"..."}
//	{"type":"suggestions","suggestions":[...]} 或 {"type":"suggestions","error":"..."}
//	{"type":"done"}
//
// parse/fetch 仍是同步前置：失败用现有 writeError / writeFetchError 映射成 HTTP 状态
// 与非流式端点完全一致。只有 fetch 成功后才把状态切到 200 + NDJSON、开始写流。
func streamHandler(fetcher pr.Fetcher, a *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json body: %v", err))
			return
		}
		if req.PRURL == "" {
			writeError(w, http.StatusBadRequest, "pr_url is required")
			return
		}
		ref, err := github.ParseRef(req.PRURL)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "streaming not supported by this ResponseWriter")
			return
		}

		changes, err := fetcher.Fetch(r.Context(), ref)
		if err != nil {
			writeFetchError(w, err)
			return
		}

		// 切到 NDJSON：所有 Header 必须在 WriteHeader 之前 Set。
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")
		// X-Accel-Buffering: no 让 nginx 这类反代不要缓冲——直通的好，前端才看得到分批。
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w) // Encode 自带 trailing '\n'，正好 NDJSON。
		emit := func(obj any) {
			_ = enc.Encode(obj)
			flusher.Flush()
		}

		emit(map[string]any{"type": "changes", "changes": changes})

		for ev := range a.AnalyzeStream(r.Context(), changes) {
			emit(eventToWire(ev))
		}

		emit(map[string]any{"type": "done"})
	}
}

// eventToWire 把 analyzer.Event 翻译成 NDJSON 单行的字段集。
// 错误路径用 {"type":"<kind>","error":"..."}；成功路径用 type + 各自 payload。
// risks 成功时一并带上 risks_filtered，让客户端可以渲染"已过滤 N 条"。
func eventToWire(ev analyzer.Event) map[string]any {
	switch ev.Kind {
	case analyzer.EventSummary:
		if ev.Err != nil {
			return map[string]any{"type": "summary", "error": ev.Err.Error()}
		}
		return map[string]any{"type": "summary", "summary": ev.Summary}
	case analyzer.EventRisks:
		if ev.Err != nil {
			return map[string]any{"type": "risks", "error": ev.Err.Error()}
		}
		risks := ev.Risks
		if risks == nil {
			risks = []pr.Risk{}
		}
		return map[string]any{
			"type":           "risks",
			"risks":          risks,
			"risks_filtered": ev.RisksFiltered,
		}
	case analyzer.EventSuggestions:
		if ev.Err != nil {
			return map[string]any{"type": "suggestions", "error": ev.Err.Error()}
		}
		sug := ev.Suggestions
		if sug == nil {
			sug = []pr.Suggestion{}
		}
		return map[string]any{"type": "suggestions", "suggestions": sug}
	}
	return map[string]any{"type": string(ev.Kind)}
}
