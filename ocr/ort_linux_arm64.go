//go:build linux && arm64

package ocr

const (
	ortLibName        = "onnxruntime_arm64.so"
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-linux-aarch64-1.24.4.tgz"
	ortArchiveLibPath = "onnxruntime-linux-aarch64-1.24.4/lib/libonnxruntime.so.1.24.4"
	ortArchiveFormat  = "tgz"
)
