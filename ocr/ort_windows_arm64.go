//go:build windows && arm64

package ocr

const (
	ortLibName        = "onnxruntime.dll"
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-win-arm64-1.24.4.zip"
	ortArchiveLibPath = "onnxruntime-win-arm64-1.24.4/lib/onnxruntime.dll"
	ortArchiveFormat  = "zip"
)
