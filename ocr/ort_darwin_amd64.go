//go:build darwin && amd64

package ocr

const (
	ortLibName        = "onnxruntime_amd64.dylib"
	ortDownloadURL    = "https://github.com/microsoft/onnxruntime/releases/download/v1.24.4/onnxruntime-osx-x86_64-1.24.4.tgz"
	ortArchiveLibPath = "onnxruntime-osx-x86_64-1.24.4/lib/libonnxruntime.1.24.4.dylib"
	ortArchiveFormat  = "tgz"
)