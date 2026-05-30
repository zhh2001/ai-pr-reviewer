package llm

import (
	"strings"
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func TestBuildSuggestionsPrompt_NoTruncation(t *testing.T) {
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 1,
		Title: "add util",
		Files: []pr.FileChange{
			{Path: "util.go", Status: "added", Additions: 12, Patch: "@@\n+func helper() {}\n"},
		},
	}
	p := BuildSuggestionsPrompt(c)
	if strings.Contains(p.User, "[truncated]") {
		t.Errorf("should not be truncated: %s", p.User)
	}
	if !strings.Contains(p.User, "util.go") {
		t.Errorf("file path missing: %s", p.User)
	}
	if !strings.Contains(p.System, "JSON") || !strings.Contains(p.System, "suggestions") {
		t.Errorf("suggestions system prompt should request JSON with suggestions key: %q", p.System)
	}
	// 必须明确划清和 risks 的边界，否则两通道会重叠。
	if !strings.Contains(p.System, "risks") {
		t.Errorf("suggestions prompt should call out the risks/suggestions boundary: %q", p.System)
	}
	if strings.Contains(p.System, "severity") {
		t.Errorf("suggestions prompt accidentally references risks schema: %q", p.System)
	}
}

func TestBuildSuggestionsPrompt_TruncatedUnderTightBudget(t *testing.T) {
	bigPatch := strings.Repeat("z", 6000)
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 9,
		Files: []pr.FileChange{
			{Path: "a.go", Patch: bigPatch},
			{Path: "b.go", Patch: bigPatch},
		},
	}
	p := buildSuggestionsPromptWithBudget(c, 2048)
	if !strings.Contains(p.User, "[truncated]") {
		t.Errorf("expected truncation marker, got:\n%s", p.User)
	}
	if !strings.Contains(p.User, "a.go") || !strings.Contains(p.User, "b.go") {
		t.Errorf("both files should appear, got:\n%s", p.User)
	}
	if strings.Contains(p.User, strings.Repeat("z", 6000)) {
		t.Errorf("full 6000-char patch leaked through")
	}
}

func TestParseSuggestionsJSON_Canonical(t *testing.T) {
	raw := `{"suggestions":[
        {"file":"a.go","line":12,"category":"naming","suggestion":"helper 改名为 normalizePath 更明确"},
        {"file":"a_test.go","line":0,"category":"testing","suggestion":"为空字符串入参补一个用例"}
    ]}`
	got, err := ParseSuggestionsJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2; got=%+v", len(got), got)
	}
	want0 := pr.Suggestion{File: "a.go", Line: 12, Category: "naming", Suggestion: "helper 改名为 normalizePath 更明确"}
	if got[0] != want0 {
		t.Errorf("suggestion[0]: got %+v, want %+v", got[0], want0)
	}
	if got[1].Category != "testing" || got[1].Line != 0 {
		t.Errorf("suggestion[1] mismatch: %+v", got[1])
	}
}

func TestParseSuggestionsJSON_FencedCodeBlock(t *testing.T) {
	raw := "```json\n" +
		`{"suggestions":[{"file":"x.go","line":1,"category":"readability","suggestion":"把循环里的两层 if 合并"}]}` +
		"\n```"
	got, err := ParseSuggestionsJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].File != "x.go" || got[0].Category != "readability" {
		t.Errorf("fenced parse mismatch: %+v", got)
	}
}

func TestParseSuggestionsJSON_Malformed(t *testing.T) {
	cases := []string{
		"completely not json",
		`{"suggestions": [`,
		`{suggestions: []}`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := ParseSuggestionsJSON(raw)
			if err == nil {
				t.Fatalf("ParseSuggestionsJSON(%q) = %+v, want error", raw, got)
			}
		})
	}
}

// 第二条 line 是字符串，反序列化失败应被跳过，其余条保留。
func TestParseSuggestionsJSON_MixedItemsSkipsBad(t *testing.T) {
	raw := `{"suggestions":[
        {"file":"a.go","line":12,"category":"naming","suggestion":"first"},
        {"file":"b.go","line":"oops","category":"style","suggestion":"bad type"},
        {"file":"c.go","line":34,"category":"testing","suggestion":"third"}
    ]}`
	got, err := ParseSuggestionsJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2; got=%+v", len(got), got)
	}
	if got[0].File != "a.go" || got[1].File != "c.go" {
		t.Errorf("kept items mismatch: %+v", got)
	}
}

func TestParseSuggestionsJSON_AllItemsBadReturnsEmpty(t *testing.T) {
	// 用确凿会反序列化失败的数字和字符串。null 在 Go 反序列化到 struct 会成功
	// （零值），不属于"失败"语义，所以不在这条用例覆盖范围。
	raw := `{"suggestions":[1, 2, "nope"]}`
	got, err := ParseSuggestionsJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Errorf("got nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestParseSuggestionsJSON_ArrayItselfMalformed(t *testing.T) {
	cases := []string{
		`{"suggestions":"not an array"}`,
		`{"suggestions": 42}`,
		`{"suggestions": {"a":1}}`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := ParseSuggestionsJSON(raw)
			if err == nil {
				t.Fatalf("ParseSuggestionsJSON(%q) = %+v, want error", raw, got)
			}
		})
	}
}
