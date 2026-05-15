package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ListDir(args map[string]any) ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		return ToolResult{Output: "错误: 无可用的工作区目录", Success: false}
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径错误: %v", err), Success: false}
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("读取目录失败: %v", err), Success: false}
	}

	var sb strings.Builder
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if entry.IsDir() {
			fmt.Fprintf(&sb, "[目录] %s\n", entry.Name())
		} else {
			fmt.Fprintf(&sb, "[文件] %s (%d bytes)\n", entry.Name(), info.Size())
		}
	}
	if sb.Len() == 0 {
		return ToolResult{Output: "目录为空", Success: true}
	}
	return ToolResult{Output: sb.String(), Success: true}
}
