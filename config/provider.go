package config

// provider.yaml 是「已配置供应商」的存档,供 /provider 快捷切换。
//
// 每次 /config 完成会把当前配置按供应商名(deepseek / mimo / kimi / qwen / custom)
// upsert 进来;/provider 从这里读名字列表、把选中供应商的 flash/pro 写回 model.yaml。
//
// YAML 结构(供应商名 → 该供应商的 {flash, pro},与 model.yaml 的 Config 同构):
//
//	deepseek:
//	  flash: {base_url, model, api_key, context_window, max_tokens}
//	  pro:   {base_url, model, api_key, context_window, max_tokens}
//	mimo:
//	  flash: {...}
//	  pro:   {...}

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const providerFileName = "provider.yaml"

// Providers 是 provider.yaml 的反序列化目标:供应商名 → 该供应商的 {flash, pro} 配置。
type Providers map[string]Config

// ProviderPath 返回 ~/.deepx/provider.yaml 绝对路径。
func ProviderPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(home, dirName, providerFileName), nil
}

// LoadProviders 读 provider.yaml。文件不存在视为空(返回空 map,非错误);解析失败返回 err。
func LoadProviders() (Providers, error) {
	p, err := ProviderPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Providers{}, nil
	}
	if err != nil {
		return nil, err
	}
	var ps Providers
	if err := yaml.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", p, err)
	}
	if ps == nil {
		ps = Providers{}
	}
	return ps, nil
}

// SaveProvider 把一份配置按供应商名 upsert 进 provider.yaml(其余供应商原样保留)。
// custom 统一占 "custom" 一个槽,后配置的覆盖旧的。文件权限 0600(含 api key)。
func SaveProvider(name string, c *Config) error {
	if name == "" || c == nil {
		return nil
	}
	ps, err := LoadProviders()
	if err != nil {
		// 读失败(文件损坏)就从空开始,免得一份坏文件永久挡住存档。
		ps = Providers{}
	}
	ps[name] = *c
	return saveProviders(ps)
}

func saveProviders(ps Providers) error {
	p, err := ProviderPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(ps)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

// LoadProvider 取单个供应商的存档配置;不存在返回 (nil, false, nil)。
func LoadProvider(name string) (*Config, bool, error) {
	ps, err := LoadProviders()
	if err != nil {
		return nil, false, err
	}
	c, ok := ps[name]
	if !ok {
		return nil, false, nil
	}
	return &c, true, nil
}

// ProviderNames 返回 provider.yaml 中已存的供应商名:预设供应商(ProviderOptions)按其顺序
// 排在前,其余未知名按字母序排在后,便于 /provider 选择器稳定展示。文件为空时返回空切片。
func ProviderNames() ([]string, error) {
	ps, err := LoadProviders()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ps))
	seen := make(map[string]bool, len(ps))
	for _, p := range ProviderOptions {
		if _, ok := ps[p]; ok {
			names = append(names, p)
			seen[p] = true
		}
	}
	rest := make([]string, 0)
	for k := range ps {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(names, rest...), nil
}
