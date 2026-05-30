package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// parseEnvelopedJSON 解析形如 {"key":[...]} 的对象，把数组反序列化为 []T。
// 即便系统提示已禁止 markdown，模型偶尔仍包 ```json fence，这里做一层兜底剥离。
// 字段缺失 / 为 null 时视作空列表，不视为错误。
//
// 容错策略：
//   - 信封结构本身畸形（无法 Unmarshal 成 object）→ 返回 error，整通道失败。
//   - 数组结构畸形（key 对应的值不是数组）         → 返回 error，整通道失败。
//   - 数组能解析但某些 item 反序列化失败           → 跳过坏条，保留好条。
//   - 所有 item 都坏                              → 返回空列表（非 nil）而非 error。
func parseEnvelopedJSON[T any](raw, key string) ([]T, error) {
	s := stripFences(strings.TrimSpace(raw))
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		snippet := s
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("parse %s json: %w; got=%q", key, err, snippet)
	}
	rawArr, ok := env[key]
	if !ok {
		return []T{}, nil
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(rawArr, &rawItems); err != nil {
		return nil, fmt.Errorf("parse %s array: %w", key, err)
	}
	out := make([]T, 0, len(rawItems))
	skipped := 0
	for _, ri := range rawItems {
		var t T
		if err := json.Unmarshal(ri, &t); err != nil {
			skipped++
			continue
		}
		out = append(out, t)
	}
	if skipped > 0 {
		log.Printf("parseEnvelopedJSON %s: skipped %d malformed item(s) out of %d",
			key, skipped, len(rawItems))
	}
	return out, nil
}

// stripFences 去掉首尾 ```（可带语言标签如 ```json），其它内容原样保留。
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	} else {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimRight(s, " \n\r\t")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
