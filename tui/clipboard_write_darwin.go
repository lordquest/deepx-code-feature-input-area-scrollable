//go:build darwin

package tui

import (
	"os/exec"
	"strings"
)

// writeClipboardText 用 macOS 的 pbcopy 把文本写到系统剪贴板。
// pbcopy 是系统自带,无外部依赖。失败时返回 err,调用方静默忽略即可
// (复制失败不是关键路径,用户可以重新选)。
func writeClipboardText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
