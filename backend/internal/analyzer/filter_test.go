package analyzer

import (
	"testing"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

func TestFilterRisksByConfidence(t *testing.T) {
	high := pr.Risk{File: "a.go", Confidence: 0.9}
	mid := pr.Risk{File: "b.go", Confidence: 0.5}
	low := pr.Risk{File: "c.go", Confidence: 0.2}

	cases := []struct {
		name      string
		risks     []pr.Risk
		threshold float64
		wantKept  []pr.Risk
	}{
		{"empty input", nil, 0.5, []pr.Risk{}},
		{"all above threshold", []pr.Risk{high, {File: "x.go", Confidence: 0.8}}, 0.5,
			[]pr.Risk{high, {File: "x.go", Confidence: 0.8}}},
		{"all below threshold", []pr.Risk{low, {File: "x.go", Confidence: 0.1}}, 0.5,
			[]pr.Risk{}},
		{"boundary kept (>=)", []pr.Risk{mid}, 0.5, []pr.Risk{mid}},
		{"just below boundary dropped", []pr.Risk{{File: "z.go", Confidence: 0.4999}}, 0.5,
			[]pr.Risk{}},
		{"mixed", []pr.Risk{high, mid, low}, 0.5, []pr.Risk{high, mid}},
		{"threshold zero keeps all", []pr.Risk{high, mid, low}, 0, []pr.Risk{high, mid, low}},
		{"threshold one keeps only perfect", []pr.Risk{
			{File: "p.go", Confidence: 1.0}, high, mid,
		}, 1.0, []pr.Risk{{File: "p.go", Confidence: 1.0}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterRisksByConfidence(tc.risks, tc.threshold)
			if len(got) != len(tc.wantKept) {
				t.Fatalf("len = %d, want %d; got=%+v", len(got), len(tc.wantKept), got)
			}
			for i := range got {
				if got[i] != tc.wantKept[i] {
					t.Errorf("kept[%d] = %+v, want %+v", i, got[i], tc.wantKept[i])
				}
			}
			// "过滤了几条"语义：原数 - 留下数。
			wantFiltered := len(tc.risks) - len(tc.wantKept)
			gotFiltered := len(tc.risks) - len(got)
			if gotFiltered != wantFiltered {
				t.Errorf("filtered count = %d, want %d", gotFiltered, wantFiltered)
			}
		})
	}
}

// 纯函数语义：不就地修改入参。
func TestFilterRisksByConfidence_DoesNotMutateInput(t *testing.T) {
	in := []pr.Risk{
		{File: "a.go", Confidence: 0.9},
		{File: "b.go", Confidence: 0.1},
		{File: "c.go", Confidence: 0.6},
	}
	snapshot := append([]pr.Risk{}, in...)
	_ = filterRisksByConfidence(in, 0.5)
	for i := range in {
		if in[i] != snapshot[i] {
			t.Errorf("input mutated at %d: got %+v, want %+v", i, in[i], snapshot[i])
		}
	}
}
