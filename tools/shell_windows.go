//go:build windows

package tools

import (
	"os/exec"
	"syscall"
)

// plainShellCmd 在 Windows 上用 cmd.exe 跑命令,但**绕过 Go 的自动参数转义**。
//
// 问题(issue #171):exec.Command("cmd","/C",command) 把整条命令当成一个参数,Go 的
// syscall.EscapeArg 会把其中的 " 转义成 \" —— 而 cmd.exe 不认这种反斜杠转义,导致带空格的
// 引号参数被错误拆分(如 --title "Hello World" 被拆成 "Hello 和 World")。
//
// 修法:直接设 SysProcAttr.CmdLine,Go 便不再对参数做转义,把命令串原样交给 cmd 按它自己的
// 规则解析——等价于用户在 cmd 里手敲。`/S` + 整体一对外层引号:cmd 会剥掉这对外层引号、把里面的
// 命令逐字执行,内部引号交给 cmd 正常处理;&&、||、|、& 等 cmd 原生支持的操作符也保持可用
// (这正是不改用 powershell 的原因——Windows 自带的 PowerShell 5.1 不支持 && / ||)。
func plainShellCmd(command, cwd string) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /S /C "` + command + `"`}
	if cwd != "" {
		cmd.Dir = cwd
	}
	return cmd
}
