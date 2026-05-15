package ocr

import (
	"os"
	"strings"
)

// mirrorCandidates 把单条 github.com / raw.githubusercontent.com URL 展开成
// "国内镜像优先 + 原站兜底" 的候选列表,供 downloadFile 顺序尝试。
//
// 顺序:
//  1. raw.githubusercontent.com → jsDelivr CDN (小文本秒级)
//  2. DEEPX_GH_PROXY 环境变量(用户自定义代理前缀)
//  3. 内置 GH 代理列表 (ghfast.top / gh-proxy.com / ghproxy.net)
//  4. 原 URL (海外用户、所有代理失效时兜底)
//
// 行为开关:
//   - DEEPX_DISABLE_GH_MIRROR=1 → 跳过所有镜像,只返回原 URL
//   - DEEPX_GH_PROXY=<url>      → 注入到第二位 (jsDelivr 之后,内置代理之前)
func mirrorCandidates(url string) []string {
	if os.Getenv("DEEPX_DISABLE_GH_MIRROR") == "1" {
		return []string{url}
	}

	var out []string
	seen := map[string]bool{}
	add := func(u string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}

	// 1) 用户自定义代理(最高优先级)
	if proxy := strings.TrimRight(os.Getenv("DEEPX_GH_PROXY"), "/"); proxy != "" {
		add(proxy + "/" + url)
	}

	// 2) 内置 GH 代理,仅对 github.com 系域名拼接
	// 实测 (2025-2026) ghfast.top 在国内最快,比 jsDelivr CDN 边缘节点稳定
	if isGitHubURL(url) {
		for _, p := range builtinGHProxies {
			add(p + "/" + url)
		}
	}

	// 3) jsDelivr 退到 GH 代理之后做备用 — 它对中国用户偶尔慢/不稳,
	// 但作为非 github.com 系域名的备份还是有用(走 Cloudflare,不被 GFW 直接卡)
	if jsd := toJsDelivr(url); jsd != "" {
		add(jsd)
	}

	// 4) 原 URL 兜底
	add(url)
	return out
}

// builtinGHProxies 内置可用代理。顺序按稳定性排,失败会顺序回退。
// 任何一个失效不影响整体,因为最后还有原 URL 兜底。
var builtinGHProxies = []string{
	"https://ghfast.top",
	"https://gh-proxy.com",
	"https://ghproxy.net",
}

func isGitHubURL(url string) bool {
	return strings.HasPrefix(url, "https://github.com/") ||
		strings.HasPrefix(url, "https://raw.githubusercontent.com/") ||
		strings.HasPrefix(url, "https://objects.githubusercontent.com/")
}

// toJsDelivr 把 raw.githubusercontent.com URL 改写成 cdn.jsdelivr.net 镜像。
// 输入: https://raw.githubusercontent.com/<user>/<repo>/<ref>/<path>
// 输出: https://cdn.jsdelivr.net/gh/<user>/<repo>@<ref>/<path>
// 非 raw 链接 / 路径不全时返回空串,调用方据此决定是否跳过。
func toJsDelivr(url string) string {
	const prefix = "https://raw.githubusercontent.com/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	rest := url[len(prefix):]
	parts := strings.SplitN(rest, "/", 4)
	if len(parts) < 4 {
		return ""
	}
	user, repo, ref, path := parts[0], parts[1], parts[2], parts[3]
	return "https://cdn.jsdelivr.net/gh/" + user + "/" + repo + "@" + ref + "/" + path
}
