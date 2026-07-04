package eventbus

import "strings"

// topicPattern 表示一个 topic 匹配模式
type topicPattern struct {
	raw      string   // 原始模式字符串，如 "order.*"
	segments []string // "*"表示通配段，其它为精确段
}

// parseTopicPattern 解析 topic 模式。
// `*` → 匹配全部; `order.*` → 前缀匹配; `*.created` → 后缀匹配。
func parseTopicPattern(raw string) topicPattern {
	parts := strings.Split(raw, ".")
	segments := make([]string, len(parts))
	for i, p := range parts {
		segments[i] = strings.TrimSpace(p)
	}
	return topicPattern{raw: raw, segments: segments}
}

// matches 判断 topic 是否匹配此模式
func (p topicPattern) matches(topic string) bool {
	// 快速路径：精确匹配
	if !strings.Contains(p.raw, "*") {
		return p.raw == topic
	}

	topicParts := strings.Split(topic, ".")
	patLen := len(p.segments)

	// 末尾是 * 且只有一个 * → 前缀匹配（如 "order.*" 匹配 "order.x", "order.x.y"）
	if patLen >= 1 && p.segments[patLen-1] == "*" && countStars(p.segments) == 1 {
		if len(topicParts) < patLen {
			return false
		}
		for i := 0; i < patLen-1; i++ {
			if topicParts[i] != p.segments[i] {
				return false
			}
		}
		return true
	}

	// 开头是 * 且只有一个 * → 后缀匹配（如 "*.created" 匹配 "x.created", "x.y.created"）
	if patLen >= 1 && p.segments[0] == "*" && countStars(p.segments) == 1 {
		if len(topicParts) < patLen {
			return false
		}
		offset := len(topicParts) - patLen
		for i := 1; i < patLen; i++ {
			if topicParts[offset+i] != p.segments[i] {
				return false
			}
		}
		return true
	}

	// 只有一个 "*" 段 → 匹配所有（如 "*" 匹配任何 topic）
	if patLen == 1 && p.segments[0] == "*" {
		return true
	}

	// 精确段数匹配：每段一对一比较，* 段通配
	if len(topicParts) != patLen {
		return false
	}
	for i := range p.segments {
		if p.segments[i] == "*" {
			continue
		}
		if topicParts[i] != p.segments[i] {
			return false
		}
	}
	return true
}

func countStars(segs []string) int {
	n := 0
	for _, s := range segs {
		if s == "*" {
			n++
		}
	}
	return n
}
