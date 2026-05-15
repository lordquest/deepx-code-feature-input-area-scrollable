//go:build !darwin

package tui

import (
	"fmt"
	"os/exec"
	"strings"
)

// writeClipboardText 用平台原生 CLI 把文本写到系统剪贴板。
// Linux: 优先 xclip,失败回退 xsel。
// Windows: clip.exe。
// 失败时返回 err,调用方静默忽略。
func writeClipboardText(text string) error {
	candidates := [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
		{"clip"}, // windows clip.exe
	}
	for _, args := range candidates {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no usable clipboard helper found (need one of: xclip / xsel / wl-copy / clip)")
}
