package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileTree 以树状结构显示目录。
// 参数:
//
//	path  (string)      根目录（默认当前目录）
//	depth (int, 可选)   最大深度，默认 3
func FileTree(args map[string]any) ToolResult {
	root, _ := args["path"].(string)
	if root == "" {
		root, _ = os.Getwd()
	}
	depth := toInt(args["depth"], 3)
	if depth < 1 {
		depth = 1
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径错误: %v", err), Success: false}
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径不存在: %v", err), Success: false}
	}
	if !info.IsDir() {
		return ToolResult{Output: "路径不是目录", Success: false}
	}

	var sb strings.Builder
	sb.WriteString(absRoot + "\n")
	walk(&sb, absRoot, "", 1, depth)
	return ToolResult{Output: sb.String(), Success: true}
}

func walk(sb *strings.Builder, dir, prefix string, cur, max int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	visible := entries[:0]
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}
		visible = append(visible, e)
	}
	sort.Slice(visible, func(i, j int) bool {
		if visible[i].IsDir() != visible[j].IsDir() {
			return visible[i].IsDir()
		}
		return visible[i].Name() < visible[j].Name()
	})

	for i, e := range visible {
		last := i == len(visible)-1
		branch := "├── "
		nextPrefix := prefix + "│   "
		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		fmt.Fprintf(sb, "%s%s%s\n", prefix, branch, name)
		if e.IsDir() && cur < max {
			walk(sb, filepath.Join(dir, e.Name()), nextPrefix, cur+1, max)
		}
	}
}
