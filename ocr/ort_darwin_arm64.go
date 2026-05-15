//go:build darwin && arm64

package ocr

// ORT 动态库 (libonnxruntime) 各平台不同;
// 这些常量由 assets.go 的下载器消费,把库摆到 purego 期望的命名位置。

const (
	ortLibName        = "onnxruntime_arm64.dylib" // purego DefaultLibraryPath 期望的本地名
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-osx-arm64-1.24.4.tgz"
	ortArchiveLibPath = "onnxruntime-osx-arm64-1.24.4/lib/libonnxruntime.1.24.4.dylib"
	ortArchiveFormat  = "tgz"
)
