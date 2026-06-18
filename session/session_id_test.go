package session

import (
	"runtime"
	"testing"
)

// 核心正确性:Windows 路径大小写不敏感→归一;其它平台大小写敏感→保持区分(防 issue #121 的反向回归)。
func TestSessionIDFor_CaseHandling(t *testing.T) {
	a := sessionIDFor(`C:\GitLab\webapi`)
	b := sessionIDFor(`c:\gitlab\webapi`)
	if runtime.GOOS == "windows" {
		if a != b {
			t.Fatalf("Windows 应大小写无关,却 %q != %q", a, b)
		}
	} else {
		if a == b {
			t.Fatal("非 Windows(大小写敏感)不应归一,否则会把不同路径合并成一个 session")
		}
	}
}

// 迁移靠 rawSessionID 按【原始大小写】定位旧目录,所以它必须保持大小写敏感、且 16 hex。
func TestRawSessionID(t *testing.T) {
	if rawSessionID("/X") == rawSessionID("/x") {
		t.Fatal("raw 必须大小写敏感")
	}
	if got := len(rawSessionID("/x")); got != 16 {
		t.Fatalf("应 16 hex, got %d", got)
	}
}
