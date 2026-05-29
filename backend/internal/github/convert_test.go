package github

import (
	"testing"

	gh "github.com/google/go-github/v66/github"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func ptr[T any](v T) *T { return &v }

func TestToChanges(t *testing.T) {
	ref := pr.Ref{Owner: "foo", Repo: "bar", Number: 7}
	p := &gh.PullRequest{
		Title: ptr("Add greeting"),
		Body:  ptr("Adds a hello function."),
		User:  &gh.User{Login: ptr("alice")},
		Base:  &gh.PullRequestBranch{Ref: ptr("main")},
		Head:  &gh.PullRequestBranch{Ref: ptr("feature/hello")},
	}
	files := []*gh.CommitFile{
		{
			Filename:  ptr("hello.go"),
			Status:    ptr("added"),
			Additions: ptr(10),
			Deletions: ptr(0),
			Patch:     ptr("@@ -0,0 +1,10 @@\n+package hello\n"),
		},
		{
			Filename:  ptr("README.md"),
			Status:    ptr("modified"),
			Additions: ptr(2),
			Deletions: ptr(1),
			Patch:     ptr("@@ -1,3 +1,4 @@\n-old\n+new\n"),
		},
	}

	got := toChanges(ref, p, files)

	want := &pr.PRChanges{
		Owner:       "foo",
		Repo:        "bar",
		Number:      7,
		Title:       "Add greeting",
		Description: "Adds a hello function.",
		Author:      "alice",
		BaseRef:     "main",
		HeadRef:     "feature/hello",
		Files: []pr.FileChange{
			{Path: "hello.go", Status: "added", Additions: 10, Deletions: 0, Patch: "@@ -0,0 +1,10 @@\n+package hello\n"},
			{Path: "README.md", Status: "modified", Additions: 2, Deletions: 1, Patch: "@@ -1,3 +1,4 @@\n-old\n+new\n"},
		},
	}

	if got.Owner != want.Owner || got.Repo != want.Repo || got.Number != want.Number {
		t.Errorf("ref fields: got %s/%s#%d, want %s/%s#%d",
			got.Owner, got.Repo, got.Number, want.Owner, want.Repo, want.Number)
	}
	if got.Title != want.Title || got.Description != want.Description || got.Author != want.Author {
		t.Errorf("meta: got %q/%q/%q, want %q/%q/%q",
			got.Title, got.Description, got.Author, want.Title, want.Description, want.Author)
	}
	if got.BaseRef != want.BaseRef || got.HeadRef != want.HeadRef {
		t.Errorf("branches: got base=%q head=%q, want base=%q head=%q",
			got.BaseRef, got.HeadRef, want.BaseRef, want.HeadRef)
	}
	if len(got.Files) != len(want.Files) {
		t.Fatalf("files len: got %d, want %d", len(got.Files), len(want.Files))
	}
	for i, w := range want.Files {
		if got.Files[i] != w {
			t.Errorf("file[%d]: got %+v, want %+v", i, got.Files[i], w)
		}
	}
}

func TestToChangesNilFields(t *testing.T) {
	// go-github 的 getter 在指针为 nil 时返回零值；确认我们的转换不会 panic。
	ref := pr.Ref{Owner: "o", Repo: "r", Number: 1}
	got := toChanges(ref, &gh.PullRequest{}, nil)
	if got.Title != "" || got.Author != "" || got.BaseRef != "" || got.HeadRef != "" {
		t.Errorf("expected zero strings, got %+v", got)
	}
	if got.Files == nil {
		t.Errorf("Files should be non-nil empty slice for clean JSON output")
	}
	if len(got.Files) != 0 {
		t.Errorf("Files len = %d, want 0", len(got.Files))
	}
}
