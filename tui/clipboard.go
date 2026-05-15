package tui

import "errors"

// errNoClipboardImage 表示系统剪贴板当前没有图片数据 (可能是空,也可能只有文本)。
// 各平台的 readClipboardImage 实现应当把"没图"统一映射到这个值,
// 调用方据此区分"没图"与"读取出错"。
var errNoClipboardImage = errors.New("clipboard does not contain image data")
