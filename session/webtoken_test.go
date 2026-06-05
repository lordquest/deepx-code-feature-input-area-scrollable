package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WebToken 应:首次生成并持久化到 meta.json;之后(同一 / 新建 Manager)恒返回同一值。
func TestWebTokenStableAndPersisted(t *testing.T) {
	// 隔离 HOME,避免读到真实 ~/.deepx。
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)            // unix
	t.Setenv("USERPROFILE", tmp)     // windows

	ws := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}

	m1, err := New(ws)
	if err != nil {
		t.Fatal(err)
	}
	tok := m1.WebToken()
	if tok == "" {
		t.Fatal("首次 WebToken 不应为空")
	}
	// 同一 Manager 再取 → 不变
	if got := m1.WebToken(); got != tok {
		t.Errorf("同一 Manager 令牌应不变:%q != %q", got, tok)
	}
	// meta.json 里确实落了盘
	data, err := os.ReadFile(filepath.Join(m1.RootDir(), "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), tok) {
		t.Errorf("meta.json 应包含已持久化的 web_token,内容:%s", data)
	}
	// 重新打开同一 workspace(模拟重启)→ 仍是同一令牌
	m2, err := New(ws)
	if err != nil {
		t.Fatal(err)
	}
	if got := m2.WebToken(); got != tok {
		t.Errorf("重开 session 令牌应不变(跨重启稳定):%q != %q", got, tok)
	}
}
