// Package config 负责 ~/.deepx/model.yaml 的读写。
//
// YAML 结构(每个 role 独立 base_url / model / api_key,允许用不同 provider):
//
//	flash:
//	  base_url: https://api.deepseek.com
//	  model: deepseek-v4-flash
//	  api_key: sk-...
//	pro:
//	  base_url: https://api.deepseek.com
//	  model: deepseek-v4-pro
//	  api_key: sk-...
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ModelEntry 单个 role(flash / pro)的完整配置。
type ModelEntry struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	APIKey  string `yaml:"api_key"`
}

// Config 整份 model.yaml 的反序列化目标。
type Config struct {
	Flash ModelEntry `yaml:"flash"`
	Pro   ModelEntry `yaml:"pro"`
}

const (
	dirName  = ".deepx"
	fileName = "model.yaml"

	defaultBaseURL    = "https://api.deepseek.com"
	defaultFlashModel = "deepseek-v4-flash"
	defaultProModel   = "deepseek-v4-pro"
)

// Path 返回 ~/.deepx/model.yaml 绝对路径。
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(home, dirName, fileName), nil
}

// Exists 配置文件是否已存在。出错或不存在均返回 false。
func Exists() bool {
	p, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Default 用单一 apiKey 构造初始配置:flash/pro 共享 base_url 和 key,只是 model id 不同。
// 用户之后可以手动编辑 model.yaml 把 flash 改成其它便宜模型 / 切换 base_url。
func Default(apiKey string) *Config {
	return &Config{
		Flash: ModelEntry{
			BaseURL: defaultBaseURL,
			Model:   defaultFlashModel,
			APIKey:  apiKey,
		},
		Pro: ModelEntry{
			BaseURL: defaultBaseURL,
			Model:   defaultProModel,
			APIKey:  apiKey,
		},
	}
}

// Load 从 ~/.deepx/model.yaml 读配置。文件缺失或解析失败返回 err。
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", p, err)
	}
	return &c, nil
}

// Save 写配置到 ~/.deepx/model.yaml,目录不存在会自动创建。
// 文件权限 0600(只有当前用户可读写,因为含 api key)。
func Save(c *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
