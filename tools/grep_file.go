package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepFile 在文件中按正则查找内容。
// 参数:
//
//	pattern (string) 正则表达式
//	path    (string) 搜索目录或文件（默认当前目录）
//	glob    (string) 文件名模式（如 "*.go"），可选
func GrepFile(args map[string]any) ToolResult {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return ToolResult{Output: "错误: pattern 参数为空", Success: false}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("正则错误: %v", err), Success: false}
	}

	root, _ := args["path"].(string)
	if root == "" {
		root, _ = os.Getwd()
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径错误: %v", err), Success: false}
	}

	glob, _ := args["glob"].(string)

	var sb strings.Builder
	maxResults := 200
	count := 0

	process := func(p string) bool {
		f, err := os.Open(p)
		if err != nil {
			return true
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			if re.MatchString(line) {
				if count >= maxResults {
					fmt.Fprintln(&sb, "... 结果被截断")
					return false
				}
				fmt.Fprintf(&sb, "%s:%d:%s\n", p, lineNo, line)
				count++
			}
		}
		return true
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径不存在: %v", err), Success: false}
	}
	if !info.IsDir() {
		process(absRoot)
	} else {
		walkErr := filepath.Walk(absRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			name := info.Name()
			if info.IsDir() && (name == ".git" || name == "node_modules" || name == "vendor") {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			if glob != "" {
				ok, _ := filepath.Match(glob, name)
				if !ok {
					return nil
				}
			}
			if !process(p) {
				return filepath.SkipAll
			}
			return nil
		})
		if walkErr != nil && walkErr != filepath.SkipAll {
			return ToolResult{Output: fmt.Sprintf("遍历失败: %v", walkErr), Success: false}
		}
	}

	if count == 0 {
		return ToolResult{Output: "无匹配", Success: true}
	}
	return ToolResult{Output: sb.String(), Success: true}
}
