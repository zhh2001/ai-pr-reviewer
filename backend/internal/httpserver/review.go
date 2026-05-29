package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/analyzer"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

type reviewRequest struct {
	PRURL string `json:"pr_url"`
}

// writeFetchError 把 github.FetchError 按 Kind 精确映射成 HTTP 状态：
//   - KindNotFound    → 404，明确告诉用户"不存在或无权访问"
//   - KindRateLimited → 429，提示稍后重试
//   - 其它（含未分类） → 502，保留上游原始消息便于排查
func writeFetchError(w http.ResponseWriter, err error) {
	var fe *github.FetchError
	if errors.As(err, &fe) {
		switch fe.Kind {
		case github.KindNotFound:
			writeError(w, http.StatusNotFound, "PR 不存在或无权访问（GitHub 返回 404）")
			return
		case github.KindRateLimited:
			writeError(w, http.StatusTooManyRequests, "GitHub 限流，请稍后重试")
			return
		}
	}
	writeError(w, http.StatusBadGateway, fmt.Sprintf("fetch pr: %v", err))
}

func reviewHandler(fetcher pr.Fetcher, a *analyzer.Analyzer) http.HandlerFunc {
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
		changes, err := fetcher.Fetch(r.Context(), ref)
		if err != nil {
			writeFetchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a.Analyze(r.Context(), changes))
	}
}
