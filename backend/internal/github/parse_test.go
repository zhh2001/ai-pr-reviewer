package github

import (
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func TestParseRef(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    pr.Ref
		wantErr bool
	}{
		{
			name: "full url",
			in:   "https://github.com/foo/bar/pull/123",
			want: pr.Ref{Owner: "foo", Repo: "bar", Number: 123},
		},
		{
			name: "full url with trailing slash",
			in:   "https://github.com/foo/bar/pull/123/",
			want: pr.Ref{Owner: "foo", Repo: "bar", Number: 123},
		},
		{
			name: "http scheme",
			in:   "http://github.com/foo/bar/pull/1",
			want: pr.Ref{Owner: "foo", Repo: "bar", Number: 1},
		},
		{
			name: "url with surrounding whitespace",
			in:   "  https://github.com/foo/bar/pull/9  ",
			want: pr.Ref{Owner: "foo", Repo: "bar", Number: 9},
		},
		{
			name: "shorthand",
			in:   "foo/bar#123",
			want: pr.Ref{Owner: "foo", Repo: "bar", Number: 123},
		},
		{
			name: "shorthand with dots and dashes",
			in:   "my-org/some.repo#42",
			want: pr.Ref{Owner: "my-org", Repo: "some.repo", Number: 42},
		},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "   ", wantErr: true},
		{name: "wrong host", in: "https://gitlab.com/foo/bar/pull/1", wantErr: true},
		{name: "wrong path segment", in: "https://github.com/foo/bar/pulls/1", wantErr: true},
		{name: "missing number", in: "https://github.com/foo/bar/pull/", wantErr: true},
		{name: "non-numeric number", in: "https://github.com/foo/bar/pull/abc", wantErr: true},
		{name: "zero number", in: "https://github.com/foo/bar/pull/0", wantErr: true},
		{name: "negative number", in: "https://github.com/foo/bar/pull/-1", wantErr: true},
		{name: "extra path", in: "https://github.com/foo/bar/pull/1/files", wantErr: true},
		{name: "shorthand missing repo", in: "foo#1", wantErr: true},
		{name: "shorthand non-numeric", in: "foo/bar#abc", wantErr: true},
		{name: "shorthand zero", in: "foo/bar#0", wantErr: true},
		{name: "shorthand with spaces", in: "foo /bar#1", wantErr: true},
		{name: "garbage", in: "lol", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRef(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseRef(%q) = %+v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRef(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseRef(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}
