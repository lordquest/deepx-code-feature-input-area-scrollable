package tui

import "testing"

// TestDetectAppSideCursorBlink:只有 VS Code 集成终端才 app 侧闪光标,其余交给终端。
// 默认必须是 false —— 一旦回退成"所有终端都 app 侧闪",issue #167 的吐 q 就会复活。
func TestDetectAppSideCursorBlink(t *testing.T) {
	cases := []struct {
		termProgram string
		want        bool
	}{
		{"vscode", true},
		{"Apple_Terminal", false},
		{"iTerm.app", false},
		{"WezTerm", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.termProgram, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", c.termProgram)
			if got := detectAppSideCursorBlink(); got != c.want {
				t.Fatalf("TERM_PROGRAM=%q: detectAppSideCursorBlink() = %v, want %v", c.termProgram, got, c.want)
			}
		})
	}
}

// TestCursorBlinkTickDoesNotToggleCursor:非 VS Code 终端下,600ms 心跳不得再切光标可见性。
// 切一次 = bubbletea 重发一条 DECSCUSR("\x1b[N q"),在 CSI 解析不健全的终端上会漏出字面 q
// (issue #167)。心跳本身仍要继续(它还驱动 codegraph 状态轮询与 workflow 计时重画)。
func TestCursorBlinkTickDoesNotToggleCursor(t *testing.T) {
	orig := appSideCursorBlink
	defer func() { appSideCursorBlink = orig }()

	appSideCursorBlink = false
	m := model{}
	for range 4 {
		next, cmd := m.Update(cursorBlinkTickMsg{})
		m = next.(model)
		if m.cursorBlinkOff {
			t.Fatal("非 app 侧闪烁的终端下,cursorBlinkOff 必须恒为 false(光标常驻,样式不变)")
		}
		if cmd == nil {
			t.Fatal("cursorBlinkTick 心跳不能停:codegraph 状态轮询和 workflow 计时重画都搭它的车")
		}
	}

	// VS Code:仍按原逻辑逐拍翻转,保住那边的闪烁手感。
	appSideCursorBlink = true
	m = model{}
	next, _ := m.Update(cursorBlinkTickMsg{})
	if !next.(model).cursorBlinkOff {
		t.Fatal("VS Code 集成终端下 app 侧闪烁应照常翻转 cursorBlinkOff")
	}
}
