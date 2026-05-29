package github

import (
	"errors"
	"net/http"

	gh "github.com/google/go-github/v66/github"
)

// FetchErrorKind 是 fetch 失败的语义分类。
// 把"上游 HTTP 状态 → 业务语义"的映射封死在这一层，handler 不必引 go-github。
type FetchErrorKind int

const (
	// KindUpstream 是兜底：未识别的 4xx / 5xx 或网络错误。
	KindUpstream FetchErrorKind = iota
	// KindNotFound 是上游 404：PR 或仓库不存在 / 当前 token 无权访问。
	KindNotFound
	// KindRateLimited 覆盖主限流 (RateLimitError) 与次级限流 (AbuseRateLimitError)。
	KindRateLimited
)

// FetchError 把任意上游错误归类成带 Kind 的领域错误。
// 保留原 error 以支持 errors.Is/As，Status/Msg 用于日志。
type FetchError struct {
	Kind   FetchErrorKind
	Status int
	Msg    string
	Err    error
}

func (e *FetchError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Msg
}

func (e *FetchError) Unwrap() error { return e.Err }

// classifyFetchError 把 go-github 返回的 err 归类。
// 顺序：先匹配具体的 RateLimit / AbuseRateLimit 类型，再回退到 ErrorResponse 按状态码分；
// 都不匹配（DNS / 连接断开等）归 Upstream。
func classifyFetchError(err error) *FetchError {
	if err == nil {
		return nil
	}

	var rle *gh.RateLimitError
	if errors.As(err, &rle) {
		return &FetchError{
			Kind:   KindRateLimited,
			Status: statusOf(rle.Response),
			Msg:    rle.Message,
			Err:    err,
		}
	}

	var arle *gh.AbuseRateLimitError
	if errors.As(err, &arle) {
		return &FetchError{
			Kind:   KindRateLimited,
			Status: statusOf(arle.Response),
			Msg:    arle.Message,
			Err:    err,
		}
	}

	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		status := statusOf(ghErr.Response)
		kind := KindUpstream
		if status == http.StatusNotFound {
			kind = KindNotFound
		}
		return &FetchError{Kind: kind, Status: status, Msg: ghErr.Message, Err: err}
	}

	return &FetchError{Kind: KindUpstream, Err: err}
}

func statusOf(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
