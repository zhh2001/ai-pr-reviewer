package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/analyzer"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

type reviewRequest struct {
	PRURL string `json:"pr_url"`
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
			writeError(w, http.StatusBadGateway, fmt.Sprintf("fetch pr: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, a.Analyze(r.Context(), changes))
	}
}
