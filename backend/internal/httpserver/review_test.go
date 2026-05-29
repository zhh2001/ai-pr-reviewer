package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

type mockFetcher struct {
	gotRef pr.Ref
	res    *pr.PRChanges
	err    error
}

func (m *mockFetcher) Fetch(_ context.Context, ref pr.Ref) (*pr.PRChanges, error) {
	m.gotRef = ref
	return m.res, m.err
}

type mockSummarizer struct {
	gotChanges *pr.PRChanges
	res        string
	err        error
}

func (m *mockSummarizer) Summarize(_ context.Context, c *pr.PRChanges) (string, error) {
	m.gotChanges = c
	return m.res, m.err
}

type mockDetector struct {
	gotChanges *pr.PRChanges
	res        []pr.Risk
	err        error
}

func (m *mockDetector) DetectRisks(_ context.Context, c *pr.PRChanges) ([]pr.Risk, error) {
	m.gotChanges = c
	return m.res, m.err
}

func doReview(t *testing.T, fetcher pr.Fetcher, summarizer pr.Summarizer, detector pr.RiskDetector, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewWithDeps(fetcher, summarizer, detector)
	req := httptest.NewRequest(http.MethodPost, "/api/review", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReview_OK(t *testing.T) {
	changes := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 7,
		Title: "hi", Author: "alice",
		BaseRef: "main", HeadRef: "feat",
		Files: []pr.FileChange{{Path: "a.go", Status: "added", Additions: 1, Patch: "@@"}},
	}
	mf := &mockFetcher{res: changes}
	ms := &mockSummarizer{res: "改了 hello 函数的返回类型，影响 a.go。"}
	md := &mockDetector{res: []pr.Risk{
		{File: "a.go", Line: 5, Severity: "high", Category: "bug", Description: "空指针", Confidence: 0.85},
	}}

	rec := doReview(t, mf, ms, md, `{"pr_url":"https://github.com/foo/bar/pull/7"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if mf.gotRef != (pr.Ref{Owner: "foo", Repo: "bar", Number: 7}) {
		t.Errorf("fetcher got ref %+v", mf.gotRef)
	}
	if ms.gotChanges != changes || md.gotChanges != changes {
		t.Errorf("summarizer and detector should both receive the same changes pointer")
	}

	var got pr.ReviewResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rec.Body.String())
	}
	if got.Changes == nil || got.Changes.Number != 7 {
		t.Errorf("changes envelope missing: %+v", got)
	}
	if got.Summary != "改了 hello 函数的返回类型，影响 a.go。" {
		t.Errorf("summary mismatch: %q", got.Summary)
	}
	if len(got.Risks) != 1 || got.Risks[0].File != "a.go" || got.Risks[0].Severity != "high" {
		t.Errorf("risks mismatch: %+v", got.Risks)
	}
	if got.SummaryError != "" || got.RisksError != "" {
		t.Errorf("error fields should be empty: %+v", got)
	}
}

func TestReview_DetectorError(t *testing.T) {
	changes := &pr.PRChanges{Owner: "foo", Repo: "bar", Number: 1, Title: "x"}
	mf := &mockFetcher{res: changes}
	ms := &mockSummarizer{res: "summary ok"}
	md := &mockDetector{err: errors.New("deepseek 503")}

	rec := doReview(t, mf, ms, md, `{"pr_url":"foo/bar#1"}`)

	// 风险识别失败不能让整个请求失败。
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got pr.ReviewResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rec.Body.String())
	}
	if got.Changes == nil || got.Changes.Title != "x" {
		t.Errorf("changes still required: %+v", got)
	}
	if got.Summary != "summary ok" {
		t.Errorf("summary should still come through when only risks failed: %q", got.Summary)
	}
	if len(got.Risks) != 0 {
		t.Errorf("risks should be empty on error: %+v", got.Risks)
	}
	if !strings.Contains(got.RisksError, "deepseek 503") {
		t.Errorf("risks_error missing upstream: %q", got.RisksError)
	}
}

func TestReview_SummarizerError(t *testing.T) {
	changes := &pr.PRChanges{Owner: "foo", Repo: "bar", Number: 1, Title: "x"}
	mf := &mockFetcher{res: changes}
	ms := &mockSummarizer{err: errors.New("deepseek 502")}
	md := &mockDetector{res: []pr.Risk{{File: "a.go", Severity: "low"}}}

	rec := doReview(t, mf, ms, md, `{"pr_url":"foo/bar#1"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got pr.ReviewResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rec.Body.String())
	}
	if got.Summary != "" {
		t.Errorf("summary should be empty on error, got %q", got.Summary)
	}
	if !strings.Contains(got.SummaryError, "deepseek 502") {
		t.Errorf("summary_error missing upstream: %q", got.SummaryError)
	}
	if len(got.Risks) != 1 {
		t.Errorf("risks should still come through when only summary failed: %+v", got.Risks)
	}
}

func TestReview_ShorthandAccepted(t *testing.T) {
	mf := &mockFetcher{res: &pr.PRChanges{}}
	ms := &mockSummarizer{res: "ok"}
	md := &mockDetector{res: []pr.Risk{}}
	rec := doReview(t, mf, ms, md, `{"pr_url":"foo/bar#42"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if mf.gotRef != (pr.Ref{Owner: "foo", Repo: "bar", Number: 42}) {
		t.Errorf("fetcher got ref %+v", mf.gotRef)
	}
}

func TestReview_InvalidJSON(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, `not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_MissingURL(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_InvalidPRURL(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, &mockSummarizer{}, &mockDetector{}, `{"pr_url":"not a url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_FetcherError(t *testing.T) {
	mf := &mockFetcher{err: errors.New("boom")}
	rec := doReview(t, mf, &mockSummarizer{}, &mockDetector{}, `{"pr_url":"foo/bar#1"}`)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := assertErrorBody(t, rec.Body)
	if !strings.Contains(body, "boom") {
		t.Errorf("error body missing upstream message: %s", body)
	}
}

func TestHealthz(t *testing.T) {
	h := NewWithDeps(&mockFetcher{}, &mockSummarizer{}, &mockDetector{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func assertErrorBody(t *testing.T, body io.Reader) string {
	t.Helper()
	var got map[string]string
	raw, _ := io.ReadAll(body)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("error body not json: %v; raw=%s", err, string(raw))
	}
	if got["error"] == "" {
		t.Errorf("expected non-empty error field, got %v", got)
	}
	return string(raw)
}
