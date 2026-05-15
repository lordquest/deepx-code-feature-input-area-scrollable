//go:build windows && amd64

package ocr

const (
	ortLibName        = "onnxruntime.dll" // Windows 端 purego 不带架构后缀
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-win-x64-1.24.4.zip"
	ortArchiveLibPath = "onnxruntime-win-x64-1.24.4/lib/onnxruntime.dll"
	ortArchiveFormat  = "zip"
)
