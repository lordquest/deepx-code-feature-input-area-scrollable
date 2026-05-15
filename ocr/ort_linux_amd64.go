//go:build linux && amd64

package ocr

const (
	ortLibName        = "onnxruntime_amd64.so"
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-linux-x64-1.24.4.tgz"
	ortArchiveLibPath = "onnxruntime-linux-x64-1.24.4/lib/libonnxruntime.so.1.24.4"
	ortArchiveFormat  = "tgz"
)
