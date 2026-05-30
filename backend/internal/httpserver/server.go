package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/analyzer"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/config"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/llm"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// New 用真实依赖装配 handler：同一个 llm.Client 同时实现 Summarizer / RiskDetector /
// SuggestionGenerator，三个子任务在 analyzer 里并发跑。
func New(cfg config.Config) http.Handler {
	client := llm.NewClient(cfg.DeepSeekAPIKey)
	a := analyzer.New(client, client, client, cfg.AnalyzeTimeout, cfg.RiskConfidenceThreshold)
	return newMux(github.NewFetcher(cfg.GitHubToken), a)
}

// NewWithDeps 暴露给 handler 测试，允许注入 mock 子任务。
// 用 config.DefaultAnalyzeTimeout 装配 analyzer，handler 测试不关心超时行为
// （超时单测在 internal/analyzer 里覆盖）。
func NewWithDeps(
	fetcher pr.Fetcher,
	summarizer pr.Summarizer,
	detector pr.RiskDetector,
	generator pr.SuggestionGenerator,
) http.Handler {
	a := analyzer.New(summarizer, detector, generator, config.DefaultAnalyzeTimeout, config.DefaultRiskConfidenceThreshold)
	return newMux(fetcher, a)
}

func newMux(fetcher pr.Fetcher, a *analyzer.Analyzer) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.Handle("POST /api/review", reviewHandler(fetcher, a))
	mux.Handle("POST /api/review/stream", streamHandler(fetcher, a))
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
