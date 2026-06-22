package agent

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestClampToolOutput_SmallPassthrough(t *testing.T) {
	in := "ok: 3 files changed"
	if got := clampToolOutput("Bash", in); got != in {
		t.Fatalf("small output should pass through unchanged, got %q", got)
	}
}

func TestStripBase64Blobs_RemovesScreenshot(t *testing.T) {
	// 模拟 browser_screenshot:短前缀 + 一大段单行 base64。
	blob := strings.Repeat("A", 200000)
	in := "data:image/png;base64," + blob
	got := stripBase64Blobs(in)
	if strings.Contains(got, blob) {
		t.Fatalf("base64 blob should be stripped")
	}
	if !strings.Contains(got, "已省略") {
		t.Fatalf("expected placeholder, got %q", got)
	}
	if len(got) > 1024 {
		t.Fatalf("stripped output too large: %d bytes", len(got))
	}
}

func TestStripBase64Blobs_KeepsNormalText(t *testing.T) {
	// 正常代码 / 文本含空白和标点,不应被当作 base64 误删。
	in := strings.Repeat("func foo(x int) int { return x + 1 }\n", 500)
	if got := stripBase64Blobs(in); got != in {
		t.Fatalf("normal text must be preserved")
	}
}

func TestClampToolOutput_TruncatesHuge(t *testing.T) {
	// 巨大的纯文本(非 base64,因含空格不构成连续 base64 串)应被字节上限截断。
	in := strings.Repeat("word ", 500000) // ~2.5MB
	got := clampToolOutput("Read", in)
	if len(got) > maxToolOutputBytes+512 {
		t.Fatalf("output not clamped: %d bytes", len(got))
	}
	if !strings.Contains(got, "已截断") {
		t.Fatalf("expected truncation notice")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("clamped output must be valid UTF-8")
	}
}

func TestClampToolOutput_UTF8Boundary(t *testing.T) {
	// 多字节字符正好跨越截断点时,不能截出半个 rune。
	in := strings.Repeat("中", maxToolOutputBytes) // 每个 3 字节,远超上限
	got := clampToolOutput("Read", in)
	if !utf8.ValidString(got) {
		t.Fatalf("clamped multibyte output must remain valid UTF-8")
	}
}
