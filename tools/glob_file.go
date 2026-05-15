package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GlobFile 按 glob 模式查找文件。
// 支持 ** 递归通配。
// 参数:
//
//	pattern (string) 匹配模式，如 "**/*.go"
//	path    (string) 搜索根目录（可选，默认当前目录）
func GlobFile(args map[string]any) ToolResult {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return ToolResult{Output: "错误: pattern 参数为空", Success: false}
	}
	root, _ := args["path"].(string)
	if root == "" {
		root, _ = os.Getwd()
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径错误: %v", err), Success: false}
	}

	type match struct {
		path string
		mod  int64
	}
	var matches []match

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
		rel, _ := filepath.Rel(absRoot, p)
		if globMatch(pattern, rel) {
			matches = append(matches, match{p, info.ModTime().Unix()})
		}
		return nil
	})
	if walkErr != nil {
		return ToolResult{Output: fmt.Sprintf("遍历失败: %v", walkErr), Success: false}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].mod > matches[j].mod })

	if len(matches) == 0 {
		return ToolResult{Output: "无匹配", Success: true}
	}
	var sb strings.Builder
	limit := 200
	for i, m := range matches {
		if i >= limit {
			fmt.Fprintf(&sb, "... 还有 %d 个结果被截断\n", len(matches)-limit)
			break
		}
		fmt.Fprintln(&sb, m.path)
	}
	return ToolResult{Output: sb.String(), Success: true}
}

// globMatch 支持 ** 递归通配的 glob 匹配。
func globMatch(pattern, name string) bool {
	pattern = filepath.ToSlash(pattern)
	name = filepath.ToSlash(name)

	if !strings.Contains(pattern, "**") {
		ok, _ := filepath.Match(pattern, name)
		if ok {
			return true
		}
		// 也允许仅匹配 basename
		ok2, _ := filepath.Match(pattern, filepath.Base(name))
		return ok2
	}
	return matchDoubleStar(pattern, name)
}

func matchDoubleStar(pattern, name string) bool {
	pp := strings.Split(pattern, "/")
	np := strings.Split(name, "/")
	return matchSegments(pp, np)
}

func matchSegments(pp, np []string) bool {
	for len(pp) > 0 {
		if pp[0] == "**" {
			if len(pp) == 1 {
				return true
			}
			for i := 0; i <= len(np); i++ {
				if matchSegments(pp[1:], np[i:]) {
					return true
				}
			}
			return false
		}
		if len(np) == 0 {
			return false
		}
		ok, _ := filepath.Match(pp[0], np[0])
		if !ok {
			return false
		}
		pp = pp[1:]
		np = np[1:]
	}
	return len(np) == 0
}
