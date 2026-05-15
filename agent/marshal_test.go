package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMarshalAssistantEmptyContentEmitsEmptyString 防止回归:
// 模型只输出 reasoning_content 时,assistant 消息序列化必须含 content 字段(哪怕空字符串),
// 否则 DeepSeek API 会 400 "Invalid assistant message: content or tool_calls must be set"。
func TestMarshalAssistantEmptyContentEmitsEmptyString(t *testing.T) {
	m := ChatMessage{
		Role:             "assistant",
		Content:          "",
		ReasoningContent: "internal thoughts...",
		ToolCalls:        nil,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal err: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"content":""`) {
		t.Errorf("expected content field present (empty string), got: %s", s)
	}
}

// TestMarshalAssistantWithToolCallsOmitsContentOK:
// 有 tool_calls 时不需要 content,空 content 仍可省略(omitempty 生效)。
func TestMarshalAssistantWithToolCallsOmitsContentOK(t *testing.T) {
	m := ChatMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "call_1", Type: "function", Function: ToolCallFunc{Name: "Read"}},
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal err: %v", err)
	}
	s := string(b)
	if strings.Contains(s, `"content"`) {
		t.Errorf("assistant with tool_calls and empty content shouldn't emit content, got: %s", s)
	}
	if !strings.Contains(s, `"tool_calls"`) {
		t.Errorf("expected tool_calls present, got: %s", s)
	}
}

// TestMarshalUserMessageStillOmits:
// 非 assistant 角色 + 空 content,应该按原逻辑 omitempty 省略 content。
func TestMarshalUserMessageStillOmits(t *testing.T) {
	m := ChatMessage{Role: "system", Content: ""}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal err: %v", err)
	}
	if strings.Contains(string(b), `"content"`) {
		t.Errorf("system with empty content should still be omitted, got: %s", string(b))
	}
}
