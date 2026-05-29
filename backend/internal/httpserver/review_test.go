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

func doReview(t *testing.T, fetcher pr.Fetcher, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewWithFetcher(fetcher)
	req := httptest.NewRequest(http.MethodPost, "/api/review", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReview_OK(t *testing.T) {
	want := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 7,
		Title: "hi", Author: "alice",
		BaseRef: "main", HeadRef: "feat",
		Files: []pr.FileChange{{Path: "a.go", Status: "added", Additions: 1, Patch: "@@"}},
	}
	mf := &mockFetcher{res: want}

	rec := doReview(t, mf, `{"pr_url":"https://github.com/foo/bar/pull/7"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if mf.gotRef != (pr.Ref{Owner: "foo", Repo: "bar", Number: 7}) {
		t.Errorf("fetcher got ref %+v", mf.gotRef)
	}
	var got pr.PRChanges
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rec.Body.String())
	}
	if got.Owner != want.Owner || got.Number != want.Number || got.Title != want.Title {
		t.Errorf("body mismatch: got %+v", got)
	}
	if len(got.Files) != 1 || got.Files[0] != want.Files[0] {
		t.Errorf("files mismatch: %+v", got.Files)
	}
}

func TestReview_ShorthandAccepted(t *testing.T) {
	mf := &mockFetcher{res: &pr.PRChanges{}}
	rec := doReview(t, mf, `{"pr_url":"foo/bar#42"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if mf.gotRef != (pr.Ref{Owner: "foo", Repo: "bar", Number: 42}) {
		t.Errorf("fetcher got ref %+v", mf.gotRef)
	}
}

func TestReview_InvalidJSON(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, `not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_MissingURL(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_InvalidPRURL(t *testing.T) {
	rec := doReview(t, &mockFetcher{}, `{"pr_url":"not a url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	assertErrorBody(t, rec.Body)
}

func TestReview_FetcherError(t *testing.T) {
	mf := &mockFetcher{err: errors.New("boom")}
	rec := doReview(t, mf, `{"pr_url":"foo/bar#1"}`)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := assertErrorBody(t, rec.Body)
	if !strings.Contains(body, "boom") {
		t.Errorf("error body missing upstream message: %s", body)
	}
}

func TestHealthz(t *testing.T) {
	h := NewWithFetcher(&mockFetcher{})
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
