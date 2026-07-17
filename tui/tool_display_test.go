package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updatePreviewArgs 用 json.Marshal 构造 Update 工具的 args,
// 由标准库自动转义路径里的反斜杠,避免 Windows 上手写 JSON 因 \U/\s 非法转义导致 json.Unmarshal 失败。
func updatePreviewArgs(path, oldS, newS string) string {
	b, err := json.Marshal(map[string]string{
		"path":       path,
		"old_string": oldS,
		"new_string": newS,
	})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestUpdatePreviewLineNumbers 字符串模式 + 文件可读:locateLineInFile 反推出行号,
// old/new 块都带行号。
func TestUpdatePreviewLineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	args := updatePreviewArgs(path, "line2\nline3", "newA\nnewB")
	out := formatUpdatePreview(args)
	if !strings.Contains(out, "2 - line2") {
		t.Errorf("expected '2 - line2', got:\n%s", out)
	}
	if !strings.Contains(out, "3 - line3") {
		t.Errorf("expected '3 - line3', got:\n%s", out)
	}
	if !strings.Contains(out, "2 + newA") {
		t.Errorf("expected '2 + newA', got:\n%s", out)
	}
	if !strings.Contains(out, "3 + newB") {
		t.Errorf("expected '3 + newB', got:\n%s", out)
	}
}

// TestUpdatePreviewNoLineInfo 文件读不到 / 没匹配:退化成无行号渲染。
func TestUpdatePreviewNoLineInfo(t *testing.T) {
	args := updatePreviewArgs("/nonexistent/file.txt", "foo", "bar")
	out := formatUpdatePreview(args)
	if !strings.Contains(out, "- foo") {
		t.Errorf("expected '- foo' fallback, got:\n%s", out)
	}
	if !strings.Contains(out, "+ bar") {
		t.Errorf("expected '+ bar' fallback, got:\n%s", out)
	}
}

// TestColorizeDiffBlock fence 行不显示,+/- 行染色,sign 检测能跳过 leading 数字行号。
func TestColorizeDiffBlock(t *testing.T) {
	in := "📝 Update (x.go)\n\n~~~diff\n  42 - old\n  42 + new\n~~~"
	out := colorizeDiffBlock(in)
	if strings.Contains(out, "~~~") {
		t.Errorf("fence should be hidden, got:\n%s", out)
	}
	if !strings.Contains(out, "42 - old") {
		t.Errorf("content preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI styling, got:\n%s", out)
	}
}
