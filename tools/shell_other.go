//go:build !windows

package tools

import "os/exec"

// plainShellCmd 在类 Unix 平台用 sh -c 跑命令(POSIX shell 的引号/转义与 Go 传参一致,无需特殊处理)。
func plainShellCmd(command, cwd string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	return cmd
}
