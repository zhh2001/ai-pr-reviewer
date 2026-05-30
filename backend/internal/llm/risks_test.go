package llm

import (
	"strings"
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func TestBuildRisksPrompt_NoTruncation(t *testing.T) {
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 1,
		Title: "tighten validation",
		Files: []pr.FileChange{
			{Path: "validate.go", Status: "modified", Additions: 4, Patch: "@@\n+if x == nil\n"},
		},
	}
	p := BuildRisksPrompt(c)
	if strings.Contains(p.User, "[truncated]") {
		t.Errorf("should not be truncated: %s", p.User)
	}
	if !strings.Contains(p.User, "validate.go") {
		t.Errorf("file path missing: %s", p.User)
	}
	if !strings.Contains(p.System, "JSON") || !strings.Contains(p.System, "risks") {
		t.Errorf("risks system prompt should request JSON with risks key: %q", p.System)
	}
	// 风险 system prompt 不该和 summary 的混了。
	if strings.Contains(p.System, "5-10 句") {
		t.Errorf("risks prompt accidentally uses summary template: %q", p.System)
	}
}

func TestBuildRisksPrompt_TruncatedUnderTightBudget(t *testing.T) {
	bigPatch := strings.Repeat("y", 6000)
	c := &pr.PRChanges{
		Owner: "foo", Repo: "bar", Number: 9,
		Files: []pr.FileChange{
			{Path: "a.go", Patch: bigPatch},
			{Path: "b.go", Patch: bigPatch},
		},
	}
	p := buildRisksPromptWithBudget(c, 2048)
	if !strings.Contains(p.User, "[truncated]") {
		t.Errorf("expected truncation marker, got:\n%s", p.User)
	}
	if !strings.Contains(p.User, "a.go") || !strings.Contains(p.User, "b.go") {
		t.Errorf("both files should appear, got:\n%s", p.User)
	}
	if strings.Contains(p.User, strings.Repeat("y", 6000)) {
		t.Errorf("full 6000-char patch leaked through")
	}
}

func TestRisksBudgetLargerThanSummary(t *testing.T) {
	// 明确约束：风险预算严格大于总结预算。
	if MaxRisksPromptBytes <= MaxSummaryPromptBytes {
		t.Errorf("risks budget %d should be larger than summary budget %d",
			MaxRisksPromptBytes, MaxSummaryPromptBytes)
	}
}

func TestParseRisksJSON_Canonical(t *testing.T) {
	raw := `{"risks":[
        {"file":"a.go","line":12,"severity":"high","category":"bug","description":"空指针解引用","confidence":0.9},
        {"file":"b.go","line":0,"severity":"low","category":"style","description":"未导出方法名拼写","confidence":0.3}
    ]}`
	got, err := ParseRisksJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2; got=%+v", len(got), got)
	}
	want0 := pr.Risk{File: "a.go", Line: 12, Severity: "high", Category: "bug", Description: "空指针解引用", Confidence: 0.9}
	if got[0] != want0 {
		t.Errorf("risk[0]: got %+v, want %+v", got[0], want0)
	}
	if got[1].File != "b.go" || got[1].Line != 0 || got[1].Confidence != 0.3 {
		t.Errorf("risk[1] mismatch: %+v", got[1])
	}
}

func TestParseRisksJSON_FencedCodeBlock(t *testing.T) {
	raw := "```json\n" + `{"risks":[{"file":"x.go","line":1,"severity":"medium","category":"security","description":"未校验 token","confidence":0.7}]}` + "\n```"
	got, err := ParseRisksJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].File != "x.go" || got[0].Severity != "medium" {
		t.Errorf("fenced parse mismatch: %+v", got)
	}
}

func TestParseRisksJSON_FencedNoLangTag(t *testing.T) {
	raw := "```\n{\"risks\":[]}\n```"
	got, err := ParseRisksJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Errorf("got nil, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestParseRisksJSON_Malformed(t *testing.T) {
	cases := []string{
		"not json at all",
		`{"risks": [`,
		`{risks: []}`, // 缺引号
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := ParseRisksJSON(raw)
			if err == nil {
				t.Fatalf("ParseRisksJSON(%q) = %+v, want error", raw, got)
			}
		})
	}
}

// 混合用例：第二条 line 是字符串，反序列化失败应被跳过，前后两条保留。
func TestParseRisksJSON_MixedItemsSkipsBad(t *testing.T) {
	raw := `{"risks":[
        {"file":"a.go","line":12,"severity":"high","category":"bug","description":"first","confidence":0.9},
        {"file":"b.go","line":"oops","severity":"low","category":"style","description":"bad type","confidence":0.4},
        {"file":"c.go","line":34,"severity":"medium","category":"performance","description":"third","confidence":0.6}
    ]}`
	got, err := ParseRisksJSON(raw)
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

// 数组里全是非对象（数字），每条 Unmarshal 失败 → 返回空 slice，无 error。
func TestParseRisksJSON_AllItemsBadReturnsEmpty(t *testing.T) {
	raw := `{"risks":[1, 2, "nope"]}`
	got, err := ParseRisksJSON(raw)
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

// 数组结构本身畸形（key 对应的值不是数组）→ 仍 error，整通道失败。
func TestParseRisksJSON_ArrayItselfMalformed(t *testing.T) {
	cases := []string{
		`{"risks":"not an array"}`,
		`{"risks": 42}`,
		`{"risks": {"a":1}}`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := ParseRisksJSON(raw)
			if err == nil {
				t.Fatalf("ParseRisksJSON(%q) = %+v, want error", raw, got)
			}
		})
	}
}
