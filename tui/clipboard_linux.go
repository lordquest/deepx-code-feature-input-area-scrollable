//go:build linux

package tui

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
)

// readClipboardImage 在 Linux 上调用 wl-paste (Wayland) 或 xclip (X11) 读取 image/png。
// 没有纯 Go 不带 cgo 的剪贴板二进制读取方案,只能依赖外部工具。
func readClipboardImage() ([]byte, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-paste"); err == nil {
			return runClipboardCmd("wl-paste", "--type", "image/png")
		}
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return runClipboardCmd("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	}
	return nil, errors.New("clipboard image requires wl-paste (Wayland) or xclip (X11) installed")
}

func runClipboardCmd(name string, args ...string) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// 没图、剪贴板为空、目标类型不存在 都会让命令退出码非零
		return nil, errNoClipboardImage
	}
	if out.Len() == 0 {
		return nil, errNoClipboardImage
	}
	return out.Bytes(), nil
}
