package ocr

import (
	"fmt"
	"image"
	"sort"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// PP-OCR det 标准预处理参数 (ImageNet 归一化)
var (
	detMean = [3]float32{0.485, 0.456, 0.406}
	detStd  = [3]float32{0.229, 0.224, 0.225}
)

const (
	detMaxSide  = 960 // 长边上限,超过会被等比缩
	detThresh   = 0.3 // 概率二值化阈值
	detBoxScale = 1.5 // bbox 向外扩展系数 (粗暴模拟 unclip),保证文字边缘不被切
	detMinSide  = 3   // bbox 短边小于该像素直接丢弃
)

// runDet 跑检测模型,返回原图坐标系的轴对齐 bbox 列表。
// MVP 版后处理:仅二值化 + 连通域 + 外接矩形,跳过多边形/旋转/Vatti unclip。
// 对屏幕截图/网页内容/文档这类水平文字场景足够。
func (o *OCR) runDet(img *image.NRGBA) ([]image.Rectangle, error) {
	srcW, srcH := img.Bounds().Dx(), img.Bounds().Dy()

	// 1. 等比缩到长边 <= detMaxSide,再上对齐 32
	scale := 1.0
	maxSide := srcW
	if srcH > maxSide {
		maxSide = srcH
	}
	if maxSide > detMaxSide {
		scale = float64(detMaxSide) / float64(maxSide)
	}
	resizedW := roundUpTo32(int(float64(srcW) * scale))
	resizedH := roundUpTo32(int(float64(srcH) * scale))
	if resizedW < 32 {
		resizedW = 32
	}
	if resizedH < 32 {
		resizedH = 32
	}
	resized := resizeBilinear(img, resizedW, resizedH)

	// 2. 归一化并组 CHW
	input := normalizeHWC2CHW(resized, detMean, detStd)

	// 3. ONNX 推理
	inputTensor, err := ort.NewTensor([]int64{1, 3, int64(resizedH), int64(resizedW)}, input)
	if err != nil {
		return nil, fmt.Errorf("det: create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	if len(o.det.InputNames) == 0 {
		return nil, fmt.Errorf("det: model has no input")
	}
	outputs, err := o.det.Run(map[string]*ort.Value{o.det.InputNames[0]: inputTensor})
	if err != nil {
		return nil, fmt.Errorf("det: run: %w", err)
	}
	for _, v := range outputs {
		defer v.Destroy()
	}

	// 4. 取第一个输出 (概率图,形状 [1,1,H,W])
	var outVal *ort.Value
	for _, v := range outputs {
		outVal = v
		break
	}
	probData, err := ort.GetTensorData[float32](outVal)
	if err != nil {
		return nil, fmt.Errorf("det: read output: %w", err)
	}
	shape, err := outVal.GetShape()
	if err != nil {
		return nil, fmt.Errorf("det: output shape: %w", err)
	}
	if len(shape) < 4 {
		return nil, fmt.Errorf("det: unexpected output shape: %v", shape)
	}
	outH, outW := int(shape[len(shape)-2]), int(shape[len(shape)-1])

	// 5. DB 后处理 (简化版): 阈值 → 连通域 → bbox
	boxes := segmentationToBoxes(probData, outW, outH, detThresh)

	// 6. 把 bbox 坐标按缩放比例映射回原图
	scaleX := float64(srcW) / float64(outW)
	scaleY := float64(srcH) / float64(outH)
	var result []image.Rectangle
	for _, b := range boxes {
		r := image.Rect(
			int(float64(b.Min.X)*scaleX),
			int(float64(b.Min.Y)*scaleY),
			int(float64(b.Max.X)*scaleX),
			int(float64(b.Max.Y)*scaleY),
		)
		// 向外扩展少许像素,弥补 axis-aligned bbox 切边
		dx := int(float64(r.Dx()) * (detBoxScale - 1.0) / 2)
		dy := int(float64(r.Dy()) * (detBoxScale - 1.0) / 2)
		r.Min.X -= dx
		r.Max.X += dx
		r.Min.Y -= dy
		r.Max.Y += dy
		r = r.Intersect(image.Rect(0, 0, srcW, srcH))
		if r.Dx() < detMinSide || r.Dy() < detMinSide {
			continue
		}
		result = append(result, r)
	}

	// 按 y 后 x 排序(类似阅读顺序),后续拼接文本时更接近自然顺序
	sort.Slice(result, func(i, j int) bool {
		// 相近行视为同一行 (容忍 1/2 box 高度差)
		hi := result[i].Dy()
		hj := result[j].Dy()
		hmax := hi
		if hj > hmax {
			hmax = hj
		}
		if abs(result[i].Min.Y-result[j].Min.Y) <= hmax/2 {
			return result[i].Min.X < result[j].Min.X
		}
		return result[i].Min.Y < result[j].Min.Y
	})

	return result, nil
}

// segmentationToBoxes 二值化概率图后用 BFS 找连通域,每个连通域取外接矩形。
// 这是替代 cv2.findContours 的最朴素方案;对水平文本盒子的几何形状准确度足够。
func segmentationToBoxes(prob []float32, w, h int, thresh float32) []image.Rectangle {
	if len(prob) < w*h {
		return nil
	}
	visited := make([]bool, w*h)
	var boxes []image.Rectangle

	// 二值掩码 + 4 连通 BFS
	queue := make([]int, 0, 1024)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if visited[idx] || prob[idx] < thresh {
				continue
			}
			// BFS 找连通域
			queue = queue[:0]
			queue = append(queue, idx)
			visited[idx] = true
			minX, maxX, minY, maxY := x, x, y, y
			for qi := 0; qi < len(queue); qi++ {
				i := queue[qi]
				cy, cx := i/w, i%w
				if cx < minX {
					minX = cx
				}
				if cx > maxX {
					maxX = cx
				}
				if cy < minY {
					minY = cy
				}
				if cy > maxY {
					maxY = cy
				}
				// 4 邻域
				for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx, ny := cx+d[0], cy+d[1]
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					ni := ny*w + nx
					if visited[ni] || prob[ni] < thresh {
						continue
					}
					visited[ni] = true
					queue = append(queue, ni)
				}
			}
			boxes = append(boxes, image.Rect(minX, minY, maxX+1, maxY+1))
		}
	}
	return boxes
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
