package analyzer_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zhh2001/ai-pr-reviewer/backend/internal/analyzer"
	"github.com/zhh2001/ai-pr-reviewer/backend/internal/pr"
)

type funcSummarizer func(context.Context, *pr.PRChanges) (string, error)

func (f funcSummarizer) Summarize(ctx context.Context, c *pr.PRChanges) (string, error) {
	return f(ctx, c)
}

type funcDetector func(context.Context, *pr.PRChanges) ([]pr.Risk, error)

func (f funcDetector) DetectRisks(ctx context.Context, c *pr.PRChanges) ([]pr.Risk, error) {
	return f(ctx, c)
}

type funcGenerator func(context.Context, *pr.PRChanges) ([]pr.Suggestion, error)

func (f funcGenerator) GenerateSuggestions(ctx context.Context, c *pr.PRChanges) ([]pr.Suggestion, error) {
	return f(ctx, c)
}

func okSummarizer(s string) funcSummarizer {
	return func(context.Context, *pr.PRChanges) (string, error) { return s, nil }
}
func okDetector(r []pr.Risk) funcDetector {
	return func(context.Context, *pr.PRChanges) ([]pr.Risk, error) { return r, nil }
}
func okGenerator(s []pr.Suggestion) funcGenerator {
	return func(context.Context, *pr.PRChanges) ([]pr.Suggestion, error) { return s, nil }
}

func TestAnalyze_AllOK(t *testing.T) {
	a := analyzer.New(
		okSummarizer("ok"),
		okDetector([]pr.Risk{{File: "a.go", Severity: "low"}}),
		okGenerator([]pr.Suggestion{{File: "a.go", Category: "docs"}}),
		2*time.Second,
		0.0, // 现有用例不关心过滤，阈值 0 保留所有 risks
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{Owner: "o", Repo: "r", Number: 1})
	if res.Summary != "ok" {
		t.Errorf("summary: %q", res.Summary)
	}
	if len(res.Risks) != 1 || len(res.Suggestions) != 1 {
		t.Errorf("risks=%+v suggestions=%+v", res.Risks, res.Suggestions)
	}
	if res.SummaryError != "" || res.RisksError != "" || res.SuggestionsError != "" {
		t.Errorf("no errors expected: %+v", res)
	}
}

// 三个 _Error 测试锁住"一个失败不污染另外两个"的并发不变量。
func TestAnalyze_OnlySummarizerFails(t *testing.T) {
	a := analyzer.New(
		funcSummarizer(func(context.Context, *pr.PRChanges) (string, error) {
			return "", errors.New("sum-boom")
		}),
		okDetector([]pr.Risk{{File: "a.go"}}),
		okGenerator([]pr.Suggestion{{File: "a.go"}}),
		2*time.Second,
		0.0, // 现有用例不关心过滤，阈值 0 保留所有 risks
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{})
	if res.SummaryError == "" {
		t.Errorf("summary_error should be set")
	}
	if len(res.Risks) != 1 || len(res.Suggestions) != 1 {
		t.Errorf("other channels should be untouched: %+v", res)
	}
}

func TestAnalyze_OnlyDetectorFails(t *testing.T) {
	a := analyzer.New(
		okSummarizer("ok"),
		funcDetector(func(context.Context, *pr.PRChanges) ([]pr.Risk, error) {
			return nil, errors.New("risk-boom")
		}),
		okGenerator([]pr.Suggestion{{File: "a.go"}}),
		2*time.Second,
		0.0, // 现有用例不关心过滤，阈值 0 保留所有 risks
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{})
	if res.RisksError == "" {
		t.Errorf("risks_error should be set")
	}
	if res.Summary != "ok" || len(res.Suggestions) != 1 {
		t.Errorf("other channels should be untouched: %+v", res)
	}
}

func TestAnalyze_OnlyGeneratorFails(t *testing.T) {
	a := analyzer.New(
		okSummarizer("ok"),
		okDetector([]pr.Risk{{File: "a.go"}}),
		funcGenerator(func(context.Context, *pr.PRChanges) ([]pr.Suggestion, error) {
			return nil, errors.New("sug-boom")
		}),
		2*time.Second,
		0.0, // 现有用例不关心过滤，阈值 0 保留所有 risks
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{})
	if res.SuggestionsError == "" {
		t.Errorf("suggestions_error should be set")
	}
	if res.Summary != "ok" || len(res.Risks) != 1 {
		t.Errorf("other channels should be untouched: %+v", res)
	}
}

// 验证三任务真的并发，不依赖时间断言（不会因负载抖动而 flaky）：
// 三个 mock 各自 Done() 一个共享 barrier，然后 Wait() 等其它两个也 Done()。
// 只有都并发起来 barrier 才能归零；若串行执行，第一个会卡死。
func TestAnalyze_RunsConcurrently(t *testing.T) {
	var barrier sync.WaitGroup
	barrier.Add(3)
	rendezvous := func() {
		barrier.Done()
		barrier.Wait()
	}

	a := analyzer.New(
		funcSummarizer(func(context.Context, *pr.PRChanges) (string, error) {
			rendezvous()
			return "s", nil
		}),
		funcDetector(func(context.Context, *pr.PRChanges) ([]pr.Risk, error) {
			rendezvous()
			return []pr.Risk{}, nil
		}),
		funcGenerator(func(context.Context, *pr.PRChanges) ([]pr.Suggestion, error) {
			rendezvous()
			return []pr.Suggestion{}, nil
		}),
		3*time.Second,
		0.0,
	)

	done := make(chan pr.ReviewResult, 1)
	go func() {
		done <- a.Analyze(context.Background(), &pr.PRChanges{})
	}()

	select {
	case res := <-done:
		if res.Summary != "s" {
			t.Errorf("summary mismatch: %q", res.Summary)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Analyze didn't finish in 2s — tasks ran sequentially, not concurrently")
	}
}

// detector 成功时，risks 按阈值过滤，且 RisksFiltered 记录被滤掉条数；
// summary / suggestions 不受影响，降级语义保持。
func TestAnalyze_FiltersRisksByConfidence(t *testing.T) {
	mixed := []pr.Risk{
		{File: "a.go", Confidence: 0.9},
		{File: "b.go", Confidence: 0.4},
		{File: "c.go", Confidence: 0.5},
		{File: "d.go", Confidence: 0.1},
	}
	a := analyzer.New(
		okSummarizer("summary stays"),
		okDetector(mixed),
		okGenerator([]pr.Suggestion{{File: "z.go", Category: "docs"}}),
		2*time.Second,
		0.5,
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{Owner: "o", Repo: "r", Number: 1})

	if len(res.Risks) != 2 {
		t.Errorf("kept = %d, want 2 (>=0.5): %+v", len(res.Risks), res.Risks)
	}
	for _, r := range res.Risks {
		if r.Confidence < 0.5 {
			t.Errorf("kept risk below threshold: %+v", r)
		}
	}
	if res.RisksFiltered != 2 {
		t.Errorf("risks_filtered = %d, want 2", res.RisksFiltered)
	}
	if res.RisksError != "" {
		t.Errorf("risks_error should stay empty: %q", res.RisksError)
	}
	// 其它两通道不受影响。
	if res.Summary != "summary stays" {
		t.Errorf("summary changed: %q", res.Summary)
	}
	if len(res.Suggestions) != 1 {
		t.Errorf("suggestions changed: %+v", res.Suggestions)
	}
}

// detector 失败时不应做过滤，RisksFiltered 必须为 0，错误照常走 risks_error。
func TestAnalyze_DetectorErrorDoesNotFilter(t *testing.T) {
	a := analyzer.New(
		okSummarizer("ok"),
		funcDetector(func(context.Context, *pr.PRChanges) ([]pr.Risk, error) {
			return nil, errors.New("detector down")
		}),
		okGenerator([]pr.Suggestion{}),
		2*time.Second,
		0.5,
	)
	res := a.Analyze(context.Background(), &pr.PRChanges{})
	if res.RisksError == "" {
		t.Errorf("risks_error should be set")
	}
	if res.RisksFiltered != 0 {
		t.Errorf("risks_filtered = %d, want 0 on detector error", res.RisksFiltered)
	}
	if len(res.Risks) != 0 {
		t.Errorf("risks should be empty on error, got %+v", res.Risks)
	}
}

// AnalyzeStream：三个子任务各推一个 Event 到 channel；全部完成后 channel 关闭。
func TestAnalyzeStream_AllOK(t *testing.T) {
	a := analyzer.New(
		okSummarizer("hi"),
		okDetector([]pr.Risk{{File: "a.go", Severity: "high", Confidence: 0.9}}),
		okGenerator([]pr.Suggestion{{File: "a.go", Category: "naming"}}),
		2*time.Second,
		0.5,
	)
	ch := a.AnalyzeStream(context.Background(), &pr.PRChanges{Owner: "o", Repo: "r", Number: 1})

	got := map[analyzer.EventKind]analyzer.Event{}
	for ev := range ch {
		if _, dup := got[ev.Kind]; dup {
			t.Fatalf("duplicate event for kind %q", ev.Kind)
		}
		got[ev.Kind] = ev
	}
	if len(got) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(got), got)
	}
	if e := got[analyzer.EventSummary]; e.Err != nil || e.Summary != "hi" {
		t.Errorf("summary event: %+v", e)
	}
	if e := got[analyzer.EventRisks]; e.Err != nil || len(e.Risks) != 1 || e.RisksFiltered != 0 {
		t.Errorf("risks event: %+v", e)
	}
	if e := got[analyzer.EventSuggestions]; e.Err != nil || len(e.Suggestions) != 1 {
		t.Errorf("suggestions event: %+v", e)
	}
}

// AnalyzeStream：detector 报错只让 risks 事件带 Err，其它两条仍正常推出。
func TestAnalyzeStream_DetectorErrorIsolated(t *testing.T) {
	a := analyzer.New(
		okSummarizer("sum"),
		funcDetector(func(context.Context, *pr.PRChanges) ([]pr.Risk, error) {
			return nil, errors.New("risk-down")
		}),
		okGenerator([]pr.Suggestion{{File: "a.go"}}),
		2*time.Second,
		0.5,
	)
	ch := a.AnalyzeStream(context.Background(), &pr.PRChanges{Owner: "o", Repo: "r", Number: 1})

	events := map[analyzer.EventKind]analyzer.Event{}
	for ev := range ch {
		events[ev.Kind] = ev
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
	if e := events[analyzer.EventRisks]; e.Err == nil || e.Err.Error() != "risk-down" {
		t.Errorf("risks event should carry detector error, got %+v", e)
	}
	if e := events[analyzer.EventSummary]; e.Err != nil || e.Summary != "sum" {
		t.Errorf("summary should be unaffected, got %+v", e)
	}
	if e := events[analyzer.EventSuggestions]; e.Err != nil || len(e.Suggestions) != 1 {
		t.Errorf("suggestions should be unaffected, got %+v", e)
	}
}

// AnalyzeStream：risks 事件应带过滤计数。
func TestAnalyzeStream_RisksFilteredCount(t *testing.T) {
	mixed := []pr.Risk{
		{File: "a.go", Confidence: 0.9},
		{File: "b.go", Confidence: 0.2},
		{File: "c.go", Confidence: 0.6},
		{File: "d.go", Confidence: 0.1},
	}
	a := analyzer.New(
		okSummarizer("sum"),
		okDetector(mixed),
		okGenerator([]pr.Suggestion{}),
		2*time.Second,
		0.5,
	)
	ch := a.AnalyzeStream(context.Background(), &pr.PRChanges{})

	var risksEv analyzer.Event
	for ev := range ch {
		if ev.Kind == analyzer.EventRisks {
			risksEv = ev
		}
	}
	if len(risksEv.Risks) != 2 {
		t.Errorf("kept = %d, want 2: %+v", len(risksEv.Risks), risksEv.Risks)
	}
	if risksEv.RisksFiltered != 2 {
		t.Errorf("RisksFiltered = %d, want 2", risksEv.RisksFiltered)
	}
}

// AnalyzeStream：channel 必须在三条事件都发完后关闭，不能提前也不能挂着。
func TestAnalyzeStream_ChannelClosesAfterAllEvents(t *testing.T) {
	a := analyzer.New(
		okSummarizer("s"),
		okDetector([]pr.Risk{}),
		okGenerator([]pr.Suggestion{}),
		2*time.Second,
		0.0,
	)
	ch := a.AnalyzeStream(context.Background(), &pr.PRChanges{})

	done := make(chan struct{})
	go func() {
		count := 0
		for range ch {
			count++
		}
		if count != 3 {
			t.Errorf("received %d events before close, want 3", count)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close — AnalyzeStream did not finish")
	}
}

// 整体超时会让仍在等的子任务用 ctx.Err() 返回，所有通道走 *_error 降级。
func TestAnalyze_TimeoutMarksAllChannels(t *testing.T) {
	block := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	a := analyzer.New(
		funcSummarizer(func(ctx context.Context, _ *pr.PRChanges) (string, error) { return "", block(ctx) }),
		funcDetector(func(ctx context.Context, _ *pr.PRChanges) ([]pr.Risk, error) { return nil, block(ctx) }),
		funcGenerator(func(ctx context.Context, _ *pr.PRChanges) ([]pr.Suggestion, error) { return nil, block(ctx) }),
		50*time.Millisecond,
		0.0,
	)

	start := time.Now()
	res := a.Analyze(context.Background(), &pr.PRChanges{Owner: "o", Repo: "r", Number: 1})
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("Analyze took too long: %v — timeout not enforced", elapsed)
	}
	if res.SummaryError == "" || res.RisksError == "" || res.SuggestionsError == "" {
		t.Errorf("all three *_error should be set on timeout: %+v", res)
	}
}
