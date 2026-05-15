package ocr

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // 支持 GIF (首帧)
	_ "image/jpeg" // 支持 JPEG
	_ "image/png"  // 支持 PNG
	"math"
	"os"
)

// loadImage 读图。支持 PNG/JPEG/GIF (GIF 取首帧)。
// 返回 NRGBA(顺手统一到 8 位 RGBA 通道,后面归一化方便)。
func loadImage(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return toNRGBA(img), nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}

// resizeBilinear 双线性插值缩放,纯 Go 实现避免再引入 image 处理库。
// 对 OCR 场景足够;不追求 lanczos 那种质量。
func resizeBilinear(src *image.NRGBA, dstW, dstH int) *image.NRGBA {
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()
	if srcW == dstW && srcH == dstH {
		return src
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	// 边缘对齐:像素中心对齐,公式 srcX = (dstX + 0.5) * srcW/dstW - 0.5
	scaleX := float64(srcW) / float64(dstW)
	scaleY := float64(srcH) / float64(dstH)
	for y := 0; y < dstH; y++ {
		sy := (float64(y)+0.5)*scaleY - 0.5
		y0 := int(math.Floor(sy))
		y1 := y0 + 1
		dy := sy - float64(y0)
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= srcH {
			y1 = srcH - 1
		}
		for x := 0; x < dstW; x++ {
			sx := (float64(x)+0.5)*scaleX - 0.5
			x0 := int(math.Floor(sx))
			x1 := x0 + 1
			dx := sx - float64(x0)
			if x0 < 0 {
				x0 = 0
			}
			if x1 >= srcW {
				x1 = srcW - 1
			}
			p00 := src.NRGBAAt(x0, y0)
			p01 := src.NRGBAAt(x1, y0)
			p10 := src.NRGBAAt(x0, y1)
			p11 := src.NRGBAAt(x1, y1)
			w00 := (1 - dx) * (1 - dy)
			w01 := dx * (1 - dy)
			w10 := (1 - dx) * dy
			w11 := dx * dy
			r := uint8(float64(p00.R)*w00 + float64(p01.R)*w01 + float64(p10.R)*w10 + float64(p11.R)*w11)
			g := uint8(float64(p00.G)*w00 + float64(p01.G)*w01 + float64(p10.G)*w10 + float64(p11.G)*w11)
			b := uint8(float64(p00.B)*w00 + float64(p01.B)*w01 + float64(p10.B)*w10 + float64(p11.B)*w11)
			dst.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return dst
}

// normalizeHWC2CHW 把 NRGBA 图像按 ImageNet mean/std 归一化,
// 同时从 HWC 转 CHW,产出 ONNX 输入要求的 float32 平面布局。
// 输出长度 = 3 * H * W,顺序: R 平面 || G 平面 || B 平面。
func normalizeHWC2CHW(img *image.NRGBA, mean, std [3]float32) []float32 {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	out := make([]float32, 3*h*w)
	plane := h * w
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			idx := y*w + x
			out[idx] = (float32(c.R)/255.0 - mean[0]) / std[0]
			out[plane+idx] = (float32(c.G)/255.0 - mean[1]) / std[1]
			out[2*plane+idx] = (float32(c.B)/255.0 - mean[2]) / std[2]
		}
	}
	return out
}

// cropRect 从 NRGBA 中按矩形裁剪,坐标自动夹紧到图内。
func cropRect(src *image.NRGBA, r image.Rectangle) *image.NRGBA {
	b := src.Bounds()
	r = r.Intersect(b)
	if r.Empty() {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			out.SetNRGBA(x, y, src.NRGBAAt(r.Min.X+x, r.Min.Y+y))
		}
	}
	return out
}

// roundUpTo32 把 v 上对齐到 32 的倍数,det 模型要求输入 H/W 是 32 的倍数。
func roundUpTo32(v int) int {
	r := v % 32
	if r == 0 {
		return v
	}
	return v + (32 - r)
}
