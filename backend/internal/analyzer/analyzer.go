// Package analyzer 把 PR 变更喂给三个 LLM 子任务（summary / risks / suggestions）
// 并发执行，再装配成 pr.ReviewResult。
//
// 设计取舍：
//   - 不用 errgroup。三通道必须各自跑完、各自降级，任一失败不应取消其它两个；
//     这与 errgroup 默认的"一败俱败"语义相反。
//   - 用 sync.WaitGroup + 各任务独立局部变量，主 goroutine 在 Wait() 之后汇总，
//     避免多个 goroutine 写同一字段引起数据竞争。
//   - 整体超时由 Analyze 内部派生的 ctx 控制，三个子任务共享同一 deadline，
//     超时会让卡死的调用立即用 ctx.Err() 返回，对应通道走 *_error 降级。
//   - risks 通道在 detector 成功后再按置信度阈值做后置过滤；过滤只发生在 risks
//     这一份通道内，不影响 summary / suggestions，detector 报错时不过滤。
package analyzer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

// Analyzer 编排 summary / risks / suggestions 三个 LLM 子任务并发执行。
type Analyzer struct {
	summarizer    pr.Summarizer
	detector      pr.RiskDetector
	generator     pr.SuggestionGenerator
	timeout       time.Duration
	riskThreshold float64
}

func New(
	summarizer pr.Summarizer,
	detector pr.RiskDetector,
	generator pr.SuggestionGenerator,
	timeout time.Duration,
	riskThreshold float64,
) *Analyzer {
	return &Analyzer{
		summarizer:    summarizer,
		detector:      detector,
		generator:     generator,
		timeout:       timeout,
		riskThreshold: riskThreshold,
	}
}

// Analyze 并发执行三个子任务，结果与各自的错误装进 ReviewResult。
//
// 三个通道相互独立：任一子任务失败只影响自己那块字段，不会取消另外两个。
// risks 通道在成功后再按 riskThreshold 做后置过滤，RisksFiltered 记录被滤掉条数。
func (a *Analyzer) Analyze(ctx context.Context, changes *pr.PRChanges) pr.ReviewResult {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 每个子任务一组独立的局部变量。禁止多个 goroutine 写同一个字段。
	var (
		summary    string
		summaryErr error

		risks    []pr.Risk
		risksErr error

		suggestions []pr.Suggestion
		sugErr      error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		summary, summaryErr = a.summarizer.Summarize(ctx, changes)
	}()
	go func() {
		defer wg.Done()
		risks, risksErr = a.detector.DetectRisks(ctx, changes)
	}()
	go func() {
		defer wg.Done()
		suggestions, sugErr = a.generator.GenerateSuggestions(ctx, changes)
	}()
	wg.Wait()

	result := pr.ReviewResult{Changes: changes}
	logRef := changes

	if summaryErr != nil {
		log.Printf("summarize pr %s/%s#%d: %v", logRef.Owner, logRef.Repo, logRef.Number, summaryErr)
		result.SummaryError = summaryErr.Error()
	} else {
		result.Summary = summary
	}
	if risksErr != nil {
		log.Printf("detect risks pr %s/%s#%d: %v", logRef.Owner, logRef.Repo, logRef.Number, risksErr)
		result.RisksError = risksErr.Error()
	} else {
		kept := filterRisksByConfidence(risks, a.riskThreshold)
		result.Risks = kept
		result.RisksFiltered = len(risks) - len(kept)
	}
	if sugErr != nil {
		log.Printf("generate suggestions pr %s/%s#%d: %v", logRef.Owner, logRef.Repo, logRef.Number, sugErr)
		result.SuggestionsError = sugErr.Error()
	} else {
		result.Suggestions = suggestions
	}
	return result
}
