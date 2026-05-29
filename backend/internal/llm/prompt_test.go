package llm

import (
	"strings"
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func TestBuildSummaryPrompt_NoTruncation(t *testing.T) {
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 1,
		Title:       "tighten parsing",
		Description: "Reject empty refs.",
		Author:      "alice",
		BaseRef:     "main", HeadRef: "feat/parse",
		Files: []pr.FileChange{
			{Path: "parse.go", Status: "modified", Additions: 3, Deletions: 1, Patch: "@@\n-old\n+new\n"},
			{Path: "parse_test.go", Status: "modified", Additions: 5, Patch: "@@\n+tests\n"},
		},
	}

	p := BuildSummaryPrompt(c)

	if strings.Contains(p.User, "[truncated]") {
		t.Errorf("should not be truncated, got: %s", p.User)
	}
	for _, want := range []string{"foo/bar#1", "tighten parsing", "Reject empty refs.", "parse.go", "parse_test.go", "+new", "+tests"} {
		if !strings.Contains(p.User, want) {
			t.Errorf("user prompt missing %q\n---\n%s", want, p.User)
		}
	}
	if p.System == "" {
		t.Errorf("system prompt empty")
	}
	if strings.Contains(p.System, "JSON") {
		t.Errorf("summary should not request JSON mode, system=%q", p.System)
	}
}

func TestBuildSummaryPrompt_TruncatedUnderTightBudget(t *testing.T) {
	bigPatch := strings.Repeat("x", 5000)
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 9,
		Title: "big change",
		Files: []pr.FileChange{
			{Path: "a.go", Status: "modified", Additions: 100, Patch: bigPatch},
			{Path: "b.go", Status: "modified", Additions: 100, Patch: bigPatch},
		},
	}

	const budget = 1024
	p := buildSummaryPromptWithBudget(c, budget)

	if !strings.Contains(p.User, "[truncated]") {
		t.Errorf("expected truncation marker, got:\n%s", p.User)
	}
	// 两个文件都应该被列出（不能因预算紧就吞掉某个文件的存在）。
	if !strings.Contains(p.User, "a.go") || !strings.Contains(p.User, "b.go") {
		t.Errorf("both files should appear in prompt, got:\n%s", p.User)
	}
	// 软预算：允许 head 与 per-file 分隔符稍微越界，但不应失控。
	if got := len(p.User); got > 2*budget {
		t.Errorf("prompt size %d wildly exceeds budget %d", got, budget)
	}
	// 截断的 patch 不应保留完整 5000 个 x。
	if strings.Contains(p.User, strings.Repeat("x", 5000)) {
		t.Errorf("full 5000-char patch leaked through")
	}
}

func TestTruncatePatch(t *testing.T) {
	cases := []struct {
		name      string
		patch     string
		max       int
		wantMark  bool
		wantBound int // 期望输出最大长度
	}{
		{"under budget", "hello", 100, false, 100},
		{"exactly at budget", strings.Repeat("a", 50), 50, false, 50},
		{"over budget", strings.Repeat("a", 200), 100, true, 100},
		{"max smaller than marker", strings.Repeat("a", 200), 5, true, len(truncatedMarker)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncatePatch(tc.patch, tc.max)
			hasMark := strings.Contains(got, "[truncated]")
			if hasMark != tc.wantMark {
				t.Errorf("[truncated] presence: got %v want %v; out=%q", hasMark, tc.wantMark, got)
			}
			if len(got) > tc.wantBound {
				t.Errorf("len(out) = %d, want <= %d", len(got), tc.wantBound)
			}
		})
	}
}
