package agent

import "strings"

// sanitizeToolPairs 在发请求前修正消息序列,保证 tool 与 tool_calls 严格配对,规避两类 400:
//   - 孤儿 tool 消息:role=tool 但前面没有携带对应 tool_call 的 assistant
//     → "Messages with role 'tool' must be a response to a preceding message with 'tool_calls'"
//   - 悬挂 tool_calls:assistant 带 tool_calls 但缺少对应的 tool 响应(部分 API 也会拒)
//
// 这类失配可能由旧版压缩切点、中途中断、供应商流式 tool_calls 异常、或历史损坏造成(见 issue #94)。
// 做法:剔除孤儿 tool;剥掉 assistant 里无响应的 tool_call(剥空且无正文则整条丢弃)。
// 正常对话(配对完整)下原样返回,不动前缀、不影响缓存。
func sanitizeToolPairs(msgs []ChatMessage) []ChatMessage {
	// 第一遍:收集所有出现过的 tool 响应 id(用于判断某个 tool_call 有没有被回应)。
	answered := make(map[string]bool)
	for i := range msgs {
		if msgs[i].Role == "tool" && msgs[i].ToolCallID != "" {
			answered[msgs[i].ToolCallID] = true
		}
	}

	out := make([]ChatMessage, 0, len(msgs))
	valid := make(map[string]bool) // 最近一条 assistant 保留下来的 tool_call id;只有这些 id 的 tool 才合法
	changed := false
	for _, m := range msgs {
		switch m.Role {
		case "assistant":
			if len(m.ToolCalls) > 0 {
				kept := make([]ToolCall, 0, len(m.ToolCalls))
				valid = make(map[string]bool)
				for _, tc := range m.ToolCalls {
					if tc.ID != "" && answered[tc.ID] {
						kept = append(kept, tc)
						valid[tc.ID] = true
					}
				}
				if len(kept) != len(m.ToolCalls) {
					changed = true
					m.ToolCalls = kept
				}
				// tool_calls 全被剥光且无正文 → 整条丢弃(空 assistant 无意义)
				if len(m.ToolCalls) == 0 && strings.TrimSpace(m.Content) == "" {
					changed = true
					continue
				}
			} else {
				valid = make(map[string]bool) // 普通 assistant 终结上一组 tool 配对
			}
			out = append(out, m)
		case "tool":
			if m.ToolCallID != "" && valid[m.ToolCallID] {
				out = append(out, m)
			} else {
				changed = true // 孤儿 tool,丢弃
			}
		default: // user / system:终结上一组 tool 配对
			valid = make(map[string]bool)
			out = append(out, m)
		}
	}
	if !changed {
		return msgs
	}
	return out
}
