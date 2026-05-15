package ocr

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// PP-OCR rec 标准预处理参数。PP-OCRv5 rec 用 0.5 mean/std (与 v3/v4 相同)。
var (
	recMean = [3]float32{0.5, 0.5, 0.5}
	recStd  = [3]float32{0.5, 0.5, 0.5}
)

const (
	recHeight  = 48 // PP-OCRv5 rec 模型固定输入高度
	recMaxW    = 1280
	recMinW    = 32
	recRatio   = 4.0 // 经验值,根据 H 来推目标 W: H * 文字宽高比
	recBatchOK = 32  // 同长度的 crop 可能批一起跑;MVP 先一张张跑,留位
)

// loadDict 读 ppocrv5_dict.txt,每行一个字符,索引 0 留给 CTC blank。
// 返回 charset[0] = "" (blank), charset[i] = 第 i 行字符。
func loadDict(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dict: %w", err)
	}
	defer f.Close()

	chars := []string{""} // index 0 = blank
	scanner := bufio.NewScanner(f)
	// 字典文件每行可能有 trailing \n,Scanner 默认 strip
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		chars = append(chars, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// 末尾再补一个 " " 类似 PaddleOCR 行为 (空格通常单独建索引)
	chars = append(chars, " ")
	return chars, nil
}

// runRec 对单张 crop 跑识别,返回解码后文本。
// crop 长宽不定,本函数负责按规则缩放到 H=recHeight,再喂给模型。
func (o *OCR) runRec(crop *image.NRGBA) (string, error) {
	srcW, srcH := crop.Bounds().Dx(), crop.Bounds().Dy()
	if srcW < 2 || srcH < 2 {
		return "", nil
	}

	// 等比缩到 H=48,宽度按比例,再夹到 [recMinW, recMaxW]
	ratio := float64(srcW) / float64(srcH)
	dstW := int(float64(recHeight) * ratio)
	if dstW < recMinW {
		dstW = recMinW
	}
	if dstW > recMaxW {
		dstW = recMaxW
	}
	resized := resizeBilinear(crop, dstW, recHeight)

	// 归一化
	input := normalizeHWC2CHW(resized, recMean, recStd)

	// 推理
	inputTensor, err := ort.NewTensor([]int64{1, 3, int64(recHeight), int64(dstW)}, input)
	if err != nil {
		return "", fmt.Errorf("rec: create tensor: %w", err)
	}
	defer inputTensor.Destroy()

	if len(o.rec.InputNames) == 0 {
		return "", fmt.Errorf("rec: model has no input")
	}
	outputs, err := o.rec.Run(map[string]*ort.Value{o.rec.InputNames[0]: inputTensor})
	if err != nil {
		return "", fmt.Errorf("rec: run: %w", err)
	}
	for _, v := range outputs {
		defer v.Destroy()
	}

	var outVal *ort.Value
	for _, v := range outputs {
		outVal = v
		break
	}
	probs, err := ort.GetTensorData[float32](outVal)
	if err != nil {
		return "", fmt.Errorf("rec: read output: %w", err)
	}
	shape, err := outVal.GetShape()
	if err != nil {
		return "", fmt.Errorf("rec: output shape: %w", err)
	}
	// 期望形状 [1, T, C]
	if len(shape) != 3 {
		return "", fmt.Errorf("rec: unexpected output shape: %v", shape)
	}
	T := int(shape[1])
	C := int(shape[2])

	return ctcGreedyDecode(probs, T, C, o.charset), nil
}

// ctcGreedyDecode 在 [T, C] 概率矩阵上贪心选最大值,
// 再做 CTC 折叠 (合并相邻重复 + 去掉 blank=0)。
// 字符集 charset[0] 是 blank,charset[i] 是真实字符。
func ctcGreedyDecode(probs []float32, T, C int, charset []string) string {
	if T == 0 || C == 0 {
		return ""
	}
	var b strings.Builder
	prev := -1
	for t := 0; t < T; t++ {
		// 找当前时间步概率最大的 class
		best := 0
		maxP := probs[t*C]
		for c := 1; c < C; c++ {
			p := probs[t*C+c]
			if p > maxP {
				maxP = p
				best = c
			}
		}
		// CTC 规则: 跳过 blank(0),跳过与上一个相同的
		if best != 0 && best != prev && best < len(charset) {
			b.WriteString(charset[best])
		}
		prev = best
	}
	return b.String()
}
