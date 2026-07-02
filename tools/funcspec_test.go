package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIFunctionSpec_RawParameters(t *testing.T) {
	// RawParameters 非空 → 原样作为 parameters。
	f := OpenAIFunctionSpec{
		Name: "structured_output", Description: "d",
		RawParameters: json.RawMessage(`{"type":"object","properties":{"findings":{"type":"array"}}}`),
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"parameters":{"type":"object","properties":{"findings":{"type":"array"}}}`) {
		t.Fatalf("RawParameters 应原样作 parameters,实际: %s", s)
	}
}

func TestOpenAIFunctionSpec_TypedParameters(t *testing.T) {
	// 无 RawParameters → 按 ToolParam 序列化(现有工具不变)。
	f := OpenAIFunctionSpec{
		Name: "Read", Description: "d",
		Parameters: ToolParam{Type: "object", Properties: map[string]PropDef{"path": {Type: "string"}}, Required: []string{"path"}},
	}
	b, _ := json.Marshal(f)
	s := string(b)
	if !strings.Contains(s, `"parameters":{"type":"object"`) || !strings.Contains(s, `"path"`) {
		t.Fatalf("ToolParam 应正常序列化,实际: %s", s)
	}
}

// TestOpenAIFunctionSpec_RoundTrip 钉死回归:Parameters/RawParameters 都是 json:"-",
// 必须有对称的 UnmarshalJSON,否则 Marshal→Unmarshal→Marshal 会丢 parameters
// (退化成 {"type":"","properties":null}),导致压缩快照还原后 API 报
// "null is not of type object"。这是 workflow 提交引入的必现回归,此测试防止复发。
func TestOpenAIFunctionSpec_RoundTrip(t *testing.T) {
	var specs []OpenAIToolSpec
	for _, tool := range Tools {
		specs = append(specs, tool.ToOpenAISpec())
	}

	first, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}

	var restored []OpenAIToolSpec
	if err := json.Unmarshal(first, &restored); err != nil {
		t.Fatal(err)
	}

	second, err := json.Marshal(restored)
	if err != nil {
		t.Fatal(err)
	}

	// 压缩要求前缀逐字节一致,round-trip 必须字节相等。
	if string(first) != string(second) {
		t.Fatalf("round-trip 非字节一致:\n原始: %s\n还原后: %s", first, second)
	}
	// 具体防退化:parameters 不得变成空对象。
	if strings.Contains(string(second), `"parameters":{"type":"","properties":null}`) {
		t.Fatalf("parameters 丢失退化为空对象: %s", second)
	}
}
