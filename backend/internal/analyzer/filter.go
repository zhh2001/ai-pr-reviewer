package analyzer

import "github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"

// filterRisksByConfidence 保留 confidence >= threshold 的项，返回新 slice，不就地修改。
// 输入为空时返回空 slice（不 nil），方便调用方无差别处理。
func filterRisksByConfidence(risks []pr.Risk, threshold float64) []pr.Risk {
	out := make([]pr.Risk, 0, len(risks))
	for _, r := range risks {
		if r.Confidence >= threshold {
			out = append(out, r)
		}
	}
	return out
}
