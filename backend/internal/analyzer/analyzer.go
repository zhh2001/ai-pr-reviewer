// Package analyzer 把 PR 变更喂给三个 LLM 子任务（summary / risks / suggestions）
// 并发执行。两个对外编排：
//   - Analyze：等三条都跑完，装配成一个 pr.ReviewResult 一次性返回。
//   - AnalyzeStream：把每条完成的事件实时推到一个 channel，handler 那一层
//     按到达顺序写到 NDJSON 流里。
//
// 设计取舍（对两个编排都成立）：
//   - 不用 errgroup。三通道必须各自跑完、各自降级，任一失败不应取消其它两个；
//     这与 errgroup 默认的"一败俱败"语义相反。
//   - 用 sync.WaitGroup + 各任务独立局部变量（Analyze）/独立 Event 写入
//     channel（AnalyzeStream），避免多个 goroutine 共享可变状态。
//   - 整体超时由编排内部派生的 ctx 控制，三个子任务共享同一 deadline，
//     超时会让卡死的调用立即用 ctx.Err() 返回，对应通道走错误降级。
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

// EventKind 标识 AnalyzeStream 发出的事件属于哪个通道。
type EventKind string

const (
	EventSummary     EventKind = "summary"
	EventRisks       EventKind = "risks"
	EventSuggestions EventKind = "suggestions"
)

// Event 是单个通道完成时的结果包：Err 非 nil 表示该通道失败，其它 payload 字段
// 与 Kind 匹配的那个有效；成功路径下 Err 为 nil。
type Event struct {
	Kind          EventKind
	Summary       string
	Risks         []pr.Risk
	RisksFiltered int
	Suggestions   []pr.Suggestion
	Err           error
}

// Analyze 等三条通道全部跑完，装配成 ReviewResult 一次性返回。语义未变。
func (a *Analyzer) Analyze(ctx context.Context, changes *pr.PRChanges) pr.ReviewResult {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	var (
		summary       string
		summaryErr    error
		risks         []pr.Risk
		risksFiltered int
		risksErr      error
		suggestions   []pr.Suggestion
		sugErr        error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		summary, summaryErr = a.runSummary(ctx, changes)
	}()
	go func() {
		defer wg.Done()
		risks, risksFiltered, risksErr = a.runRisks(ctx, changes)
	}()
	go func() {
		defer wg.Done()
		suggestions, sugErr = a.runSuggestions(ctx, changes)
	}()
	wg.Wait()

	result := pr.ReviewResult{Changes: changes}
	if summaryErr != nil {
		result.SummaryError = summaryErr.Error()
	} else {
		result.Summary = summary
	}
	if risksErr != nil {
		result.RisksError = risksErr.Error()
	} else {
		result.Risks = risks
		result.RisksFiltered = risksFiltered
	}
	if sugErr != nil {
		result.SuggestionsError = sugErr.Error()
	} else {
		result.Suggestions = suggestions
	}
	return result
}

// AnalyzeStream 并发跑三个子任务，每条完成就把一个 Event 写到返回的 channel。
// 三条都结束后 channel 关闭，调用方 range 自然退出。
//
// channel 是 3 个 worker 多写、调用方单读——Go channel 操作天然 atomic，
// 不需要额外锁；调用方收到事件后串行化写 NDJSON 流即可。
// 顺序不保证：哪条先完成就先到。
func (a *Analyzer) AnalyzeStream(ctx context.Context, changes *pr.PRChanges) <-chan Event {
	out := make(chan Event, 3)
	go func() {
		defer close(out)

		streamCtx, cancel := context.WithTimeout(ctx, a.timeout)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			s, err := a.runSummary(streamCtx, changes)
			out <- Event{Kind: EventSummary, Summary: s, Err: err}
		}()
		go func() {
			defer wg.Done()
			risks, filtered, err := a.runRisks(streamCtx, changes)
			out <- Event{Kind: EventRisks, Risks: risks, RisksFiltered: filtered, Err: err}
		}()
		go func() {
			defer wg.Done()
			sug, err := a.runSuggestions(streamCtx, changes)
			out <- Event{Kind: EventSuggestions, Suggestions: sug, Err: err}
		}()
		wg.Wait()
	}()
	return out
}

// runSummary / runRisks / runSuggestions 是 Analyze 和 AnalyzeStream 共用的
// 单通道入口。把"调用子接口 + 失败时记 log"两件事抽到一处，让两个编排只关心
// 'event 怎么装配'，行为对外保持一致。

func (a *Analyzer) runSummary(ctx context.Context, changes *pr.PRChanges) (string, error) {
	s, err := a.summarizer.Summarize(ctx, changes)
	if err != nil {
		log.Printf("summarize pr %s/%s#%d: %v", changes.Owner, changes.Repo, changes.Number, err)
	}
	return s, err
}

// runRisks 返回：保留下来的 risks、被过滤掉的条数、错误。detector 失败时不过滤、
// filtered 为 0，与 Analyze 历史行为一致。
func (a *Analyzer) runRisks(ctx context.Context, changes *pr.PRChanges) ([]pr.Risk, int, error) {
	risks, err := a.detector.DetectRisks(ctx, changes)
	if err != nil {
		log.Printf("detect risks pr %s/%s#%d: %v", changes.Owner, changes.Repo, changes.Number, err)
		return nil, 0, err
	}
	kept := filterRisksByConfidence(risks, a.riskThreshold)
	return kept, len(risks) - len(kept), nil
}

func (a *Analyzer) runSuggestions(ctx context.Context, changes *pr.PRChanges) ([]pr.Suggestion, error) {
	sug, err := a.generator.GenerateSuggestions(ctx, changes)
	if err != nil {
		log.Printf("generate suggestions pr %s/%s#%d: %v", changes.Owner, changes.Repo, changes.Number, err)
	}
	return sug, err
}
