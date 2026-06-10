package agent

import "testing"

func roles(msgs []ChatMessage) []string {
	r := make([]string, len(msgs))
	for i, m := range msgs {
		r[i] = m.Role
	}
	return r
}

func TestSanitizeDropsOrphanTool(t *testing.T) {
	// tool 消息前面没有对应 tool_call → 孤儿,应被丢弃
	in := []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "tool", ToolCallID: "x", Content: "orphan result"},
		{Role: "assistant", Content: "answer"},
	}
	out := sanitizeToolPairs(in)
	got := roles(out)
	if len(got) != 2 || got[0] != "user" || got[1] != "assistant" {
		t.Fatalf("孤儿 tool 应被剔除,got %v", got)
	}
}

func TestSanitizeKeepsValidPair(t *testing.T) {
	in := []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "x", Function: ToolCallFunc{Name: "Read"}}}},
		{Role: "tool", ToolCallID: "x", Content: "ok"},
		{Role: "assistant", Content: "done"},
	}
	out := sanitizeToolPairs(in)
	if len(out) != 4 {
		t.Fatalf("完整配对应原样保留,got %d 条: %v", len(out), roles(out))
	}
}

func TestSanitizeStripsDanglingToolCalls(t *testing.T) {
	// assistant 带 tool_calls 但没有对应 tool 响应:
	//  - 有正文 → 剥掉 tool_calls,保留正文
	//  - 无正文 → 整条丢弃
	in := []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "让我查一下", ToolCalls: []ToolCall{{ID: "x", Function: ToolCallFunc{Name: "Read"}}}},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "y", Function: ToolCallFunc{Name: "Grep"}}}}, // 无正文、无响应 → 丢
		{Role: "assistant", Content: "结果如下"},
	}
	out := sanitizeToolPairs(in)
	got := roles(out)
	if len(got) != 3 {
		t.Fatalf("悬挂 tool_calls 处理错误,got %d 条: %v", len(got), got)
	}
	if len(out[1].ToolCalls) != 0 || out[1].Content != "让我查一下" {
		t.Errorf("应剥掉无响应 tool_calls、保留正文,got %+v", out[1])
	}
}
