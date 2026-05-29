package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/config"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/llm"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// New 用真实的 GitHub Fetcher 和 DeepSeek 客户端装配 handler。
// 同一个 llm.Client 同时实现 Summarizer 与 RiskDetector，只是内部用不同模型。
func New(cfg config.Config) http.Handler {
	client := llm.NewClient(cfg.DeepSeekAPIKey)
	return NewWithDeps(github.NewFetcher(cfg.GitHubToken), client, client)
}

// NewWithDeps 暴露给测试，允许注入 mock 依赖。
func NewWithDeps(fetcher pr.Fetcher, summarizer pr.Summarizer, detector pr.RiskDetector) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.Handle("POST /api/review", reviewHandler(fetcher, summarizer, detector))
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
