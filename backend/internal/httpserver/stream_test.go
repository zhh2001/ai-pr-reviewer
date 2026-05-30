package httpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/github"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func doStream(
	t *testing.T,
	fetcher pr.Fetcher,
	summarizer pr.Summarizer,
	detector pr.RiskDetector,
	generator pr.SuggestionGenerator,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	h := NewWithDeps(fetcher, summarizer, detector, generator)
	req := httptest.NewRequest(http.MethodPost, "/api/review/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// readNDJSON 按行解析 body 成 []map[string]any。NDJSON 一行一个对象。
func readNDJSON(t *testing.T, body *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("ndjson line not valid JSON: %v; line=%q", err, line)
		}
		out = append(out, obj)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	return out
}

func TestStream_OK(t *testing.T) {
	changes := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 7,
		Title: "demo",
		Files: []pr.FileChange{{Path: "a.go", Status: "added", Additions: 1, Patch: "@@"}},
	}
	mf := &mockFetcher{res: changes}
	ms := &mockSummarizer{res: "改了 hello。"}
	md := &mockDetector{res: []pr.Risk{
		{File: "a.go", Line: 5, Severity: "high", Category: "bug", Description: "x", Confidence: 0.85},
	}}
	mg := &mockGenerator{res: []pr.Suggestion{{File: "a.go", Category: "naming", Suggestion: "rename"}}}

	rec := doStream(t, mf, ms, md, mg, `{"pr_url":"foo/bar#7"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want application/x-ndjson", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := rec.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Errorf("X-Accel-Buffering = %q", got)
	}

	events := readNDJSON(t, rec.Body)
	if len(events) != 5 {
		t.Fatalf("want 5 events (changes + 3 channels + done), got %d: %+v", len(events), events)
	}
	if events[0]["type"] != "changes" {
		t.Errorf("first event should be 'changes', got %v", events[0]["type"])
	}
	if _, ok := events[0]["changes"].(map[string]any); !ok {
		t.Errorf("first event missing 'changes' payload")
	}
	if events[len(events)-1]["type"] != "done" {
		t.Errorf("last event should be 'done', got %v", events[len(events)-1]["type"])
	}

	// 中间三条按 type 索引验证。
	mid := map[string]map[string]any{}
	for _, ev := range events[1 : len(events)-1] {
		t, _ := ev["type"].(string)
		if _, dup := mid[t]; dup {
			t2 := ev["type"]
			t = string([]byte("dup-" + t2.(string))) // 简单去重 key
		}
		mid[t] = ev
	}
	for _, k := range []string{"summary", "risks", "suggestions"} {
		if _, ok := mid[k]; !ok {
			t.Errorf("missing event type %q in: %+v", k, events)
		}
	}
	if mid["summary"]["summary"] != "改了 hello。" {
		t.Errorf("summary payload wrong: %v", mid["summary"])
	}
	if rl, ok := mid["risks"]["risks"].([]any); !ok || len(rl) != 1 {
		t.Errorf("risks payload wrong: %v", mid["risks"])
	}
	if mid["risks"]["risks_filtered"] != float64(0) {
		t.Errorf("risks_filtered = %v, want 0", mid["risks"]["risks_filtered"])
	}
	if sl, ok := mid["suggestions"]["suggestions"].([]any); !ok || len(sl) != 1 {
		t.Errorf("suggestions payload wrong: %v", mid["suggestions"])
	}
}

func TestStream_DetectorErrorEmitsRiskErrorEventOnly(t *testing.T) {
	changes := &pr.PRChanges{Owner: "o", Repo: "r", Number: 1, Title: "t"}
	mf := &mockFetcher{res: changes}
	ms := &mockSummarizer{res: "sum"}
	md := &mockDetector{err: errors.New("risk-down")}
	mg := &mockGenerator{res: []pr.Suggestion{{File: "a.go"}}}

	rec := doStream(t, mf, ms, md, mg, `{"pr_url":"o/r#1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	events := readNDJSON(t, rec.Body)
	if len(events) != 5 {
		t.Fatalf("want 5 events, got %d: %+v", len(events), events)
	}

	var risksEv map[string]any
	var sumEv map[string]any
	var sugEv map[string]any
	for _, ev := range events {
		switch ev["type"] {
		case "risks":
			risksEv = ev
		case "summary":
			sumEv = ev
		case "suggestions":
			sugEv = ev
		}
	}
	if risksEv == nil || risksEv["error"] != "risk-down" {
		t.Errorf("risks event should carry error: %+v", risksEv)
	}
	if _, hasRisks := risksEv["risks"]; hasRisks {
		t.Errorf("risks error event should not include risks payload: %+v", risksEv)
	}
	if sumEv["summary"] != "sum" {
		t.Errorf("summary should still be present: %+v", sumEv)
	}
	if sl, ok := sugEv["suggestions"].([]any); !ok || len(sl) != 1 {
		t.Errorf("suggestions should still be present: %+v", sugEv)
	}
}

func TestStream_FetcherErrorDoesNotStartStream(t *testing.T) {
	mf := &mockFetcher{err: &github.FetchError{Kind: github.KindNotFound, Status: 404, Msg: "Not Found"}}
	rec := doStream(t, mf, &mockSummarizer{}, &mockDetector{}, &mockGenerator{}, `{"pr_url":"foo/bar#1"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("error response should be JSON, Content-Type = %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "不存在") {
		t.Errorf("expected 404 message in body, got %q", body)
	}
	// 流没开过：不应该出现 NDJSON 风格的多行 type 事件。
	if strings.Contains(body, `"type":"changes"`) || strings.Contains(body, `"type":"done"`) {
		t.Errorf("body should NOT contain stream events: %q", body)
	}
}

func TestStream_RateLimitedBeforeStream(t *testing.T) {
	mf := &mockFetcher{err: &github.FetchError{Kind: github.KindRateLimited, Status: 403, Msg: "rate"}}
	rec := doStream(t, mf, &mockSummarizer{}, &mockDetector{}, &mockGenerator{}, `{"pr_url":"o/r#1"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStream_BadJSONBody(t *testing.T) {
	rec := doStream(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, &mockGenerator{}, `not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStream_MissingPRURL(t *testing.T) {
	rec := doStream(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, &mockGenerator{}, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStream_InvalidPRURL(t *testing.T) {
	rec := doStream(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, &mockGenerator{}, `{"pr_url":"???"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
