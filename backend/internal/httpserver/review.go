package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

type reviewRequest struct {
	PRURL string `json:"pr_url"`
}

func reviewHandler(fetcher pr.Fetcher, summarizer pr.Summarizer, detector pr.RiskDetector) http.HandlerFunc {
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

		result := pr.ReviewResult{Changes: changes}

		summary, err := summarizer.Summarize(r.Context(), changes)
		if err != nil {
			log.Printf("summarize pr %s/%s#%d: %v", ref.Owner, ref.Repo, ref.Number, err)
			result.SummaryError = err.Error()
		} else {
			result.Summary = summary
		}

		risks, err := detector.DetectRisks(r.Context(), changes)
		if err != nil {
			log.Printf("detect risks pr %s/%s#%d: %v", ref.Owner, ref.Repo, ref.Number, err)
			result.RisksError = err.Error()
		} else {
			result.Risks = risks
		}

		writeJSON(w, http.StatusOK, result)
	}
}
