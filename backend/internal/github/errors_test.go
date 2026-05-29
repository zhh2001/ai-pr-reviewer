package github

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v66/github"
)

func mkResp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "https", Host: "api.github.com", Path: "/foo"},
		},
	}
}

func TestClassifyFetchError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantKind   FetchErrorKind
		wantStatus int
	}{
		{
			"github 404 → not found",
			&gh.ErrorResponse{Response: mkResp(404), Message: "Not Found"},
			KindNotFound, 404,
		},
		{
			"github 401 → upstream",
			&gh.ErrorResponse{Response: mkResp(401), Message: "Bad credentials"},
			KindUpstream, 401,
		},
		{
			"github 403 generic (no rate-limit signal) → upstream",
			&gh.ErrorResponse{Response: mkResp(403), Message: "Forbidden"},
			KindUpstream, 403,
		},
		{
			"github 422 → upstream",
			&gh.ErrorResponse{Response: mkResp(422), Message: "Unprocessable"},
			KindUpstream, 422,
		},
		{
			"github 500 → upstream",
			&gh.ErrorResponse{Response: mkResp(500), Message: "Internal"},
			KindUpstream, 500,
		},
		{
			"primary rate limit → rate limited",
			&gh.RateLimitError{Response: mkResp(403), Message: "API rate limit exceeded"},
			KindRateLimited, 403,
		},
		{
			"secondary (abuse) rate limit → rate limited",
			&gh.AbuseRateLimitError{Response: mkResp(403), Message: "You have triggered an abuse detection mechanism"},
			KindRateLimited, 403,
		},
		{
			"plain io error → upstream, status 0",
			io.EOF, KindUpstream, 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyFetchError(tc.err)
			if got == nil {
				t.Fatalf("expected non-nil FetchError")
			}
			if got.Kind != tc.wantKind {
				t.Errorf("Kind = %v, want %v", got.Kind, tc.wantKind)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("Status = %d, want %d", got.Status, tc.wantStatus)
			}
			if !errors.Is(got, tc.err) {
				t.Errorf("FetchError should wrap original error for errors.Is")
			}
			if got.Error() == "" {
				t.Errorf("Error() should be non-empty")
			}
		})
	}
}

func TestClassifyFetchError_Nil(t *testing.T) {
	if got := classifyFetchError(nil); got != nil {
		t.Errorf("expected nil for nil input, got %+v", got)
	}
}

// 即便上游响应里没有 Response（罕见，但 go-github 历史上偶有），归类仍不应 panic。
func TestClassifyFetchError_MissingResponse(t *testing.T) {
	err := &gh.ErrorResponse{Message: "weird"}
	got := classifyFetchError(err)
	if got == nil || got.Kind != KindUpstream || got.Status != 0 {
		t.Errorf("missing-response case: got %+v", got)
	}
}
