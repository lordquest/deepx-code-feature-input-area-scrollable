package tools

import (
	"deepx/ocr"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImgOCR 工具入口:对单张图片路径做 OCR,把识别出的文本返回给模型。
// 用于不支持多模态输入的 LLM (例如 DeepSeek),由模型显式调用获取图片文字。
// 内部走 PP-OCRv5_mobile det+rec ONNX,首次调用会下载 ~37MB 资产到 ~/.deepx/ocr/。
//
// **副作用**:识别完成后(无论成功 / 失败),如果图片落在 ~/.deepx/ocr/cache/
// (deepx 自己粘贴落盘的临时文件),自动删除避免缓存累积。
// 用户传入的其他路径(workspace 内的源文件等)绝不删 — OCR 是"读"工具不是"清理"工具。
func ImgOCR(args map[string]any) ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		return ToolResult{Output: "错误: path 参数为空", Success: false}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("路径错误: %v", err), Success: false}
	}
	// 不管下面成功失败,函数返回前都尝试清理 paste cache。
	defer cleanupPastedImage(abs)

	engine := ocr.Global()
	// 第一次调用会触发下载;若已下载好就是个轻量 stat。
	if !engine.IsReady() {
		if err := engine.EnsureAssets(nil); err != nil {
			return ToolResult{
				Output:  fmt.Sprintf("OCR 资产下载失败: %v\n缓存目录: %s", err, engine.CacheDir()),
				Success: false,
			}
		}
	}

	text, err := engine.RecognizeText(abs)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("OCR 失败: %v", err), Success: false}
	}
	if strings.TrimSpace(text) == "" {
		return ToolResult{Output: "(图片中未识别到文字)", Success: true}
	}
	return ToolResult{Output: text, Success: true}
}

// cleanupPastedImage 检查路径是否在 deepx 的 paste cache 目录下,是则删除。
// 不在该目录下静默跳过(保护用户的源文件)。删除失败也静默(可能文件已被外部清理)。
func cleanupPastedImage(absPath string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cacheDir := filepath.Join(home, ".deepx", "ocr", "cache")
	// 必须用 filepath.Clean + HasPrefix + 分隔符,避免 "~/.deepx/ocr/cache_evil/x.png" 误匹配
	prefix := filepath.Clean(cacheDir) + string(os.PathSeparator)
	if strings.HasPrefix(absPath, prefix) {
		_ = os.Remove(absPath)
	}
}
