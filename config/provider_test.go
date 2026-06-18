package config

import (
	"testing"
)

// TestProviderRoundTrip 验证 provider.yaml 的存档/读取/列名闭环:
// SaveProvider upsert、LoadProvider 取回原值、ProviderNames 按预设顺序排列。
func TestProviderRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())          // ProviderPath 走 os.UserHomeDir() → $HOME
	t.Setenv("USERPROFILE", t.TempDir())   // Windows 兜底,避免在该平台读到真实 home

	// 初始为空。
	if names, err := ProviderNames(); err != nil || len(names) != 0 {
		t.Fatalf("空 provider.yaml 应返回空列表, got %v err=%v", names, err)
	}

	ds := &Config{
		Flash: ModelEntry{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", APIKey: "sk-ds"},
		Pro:   ModelEntry{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-pro", APIKey: "sk-ds"},
	}
	cu := &Config{
		Flash: ModelEntry{BaseURL: "https://x.example/v1", Model: "x-small", APIKey: "sk-x", MaxTokens: 4096},
		Pro:   ModelEntry{BaseURL: "https://x.example/v1", Model: "x-large", APIKey: "sk-x", MaxTokens: 8192},
	}
	if err := SaveProvider("deepseek", ds); err != nil {
		t.Fatal(err)
	}
	if err := SaveProvider("custom", cu); err != nil {
		t.Fatal(err)
	}

	// ProviderNames:预设顺序优先(deepseek 在 custom 前)。
	names, err := ProviderNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "deepseek" || names[1] != "custom" {
		t.Fatalf("期望 [deepseek custom], got %v", names)
	}

	// LoadProvider 取回原值。
	got, ok, err := LoadProvider("custom")
	if err != nil || !ok {
		t.Fatalf("custom 应存在, ok=%v err=%v", ok, err)
	}
	if got.Flash.Model != "x-small" || got.Pro.Model != "x-large" || got.Pro.MaxTokens != 8192 {
		t.Fatalf("custom 配置读回不一致: %+v", got)
	}

	// 不存在的供应商。
	if _, ok, _ := LoadProvider("nope"); ok {
		t.Fatal("不存在的供应商应返回 ok=false")
	}

	// upsert 覆盖同名,且不影响其它供应商。
	ds2 := &Config{Flash: ModelEntry{Model: "deepseek-v5-flash"}, Pro: ModelEntry{Model: "deepseek-v5-pro"}}
	if err := SaveProvider("deepseek", ds2); err != nil {
		t.Fatal(err)
	}
	got2, _, _ := LoadProvider("deepseek")
	if got2.Flash.Model != "deepseek-v5-flash" {
		t.Fatalf("deepseek 应被覆盖为 v5, got %q", got2.Flash.Model)
	}
	if cuStill, ok, _ := LoadProvider("custom"); !ok || cuStill.Flash.Model != "x-small" {
		t.Fatal("覆盖 deepseek 不应影响 custom")
	}
}
