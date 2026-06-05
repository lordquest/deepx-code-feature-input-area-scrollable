package tui

import "testing"

// webURLExposed 决定是否打"局域网暴露"安全警告:回环视为仅本机,其余 IP 视为已暴露。
func TestWebURLExposed(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"http://127.0.0.1:8080/?t=abc", false},
		{"http://localhost:8080/?t=abc", false},
		{"http://[::1]:8080/?t=abc", false},
		{"http://192.168.1.50:8080/?t=abc", true},
		{"http://10.0.0.3:8080/?t=abc", true},
		{"http://0.0.0.0:8080/?t=abc", true},     // 理论上不会回显 0.0.0.0,但若回显也应视为暴露
		{"http://example.local:8080/?t=abc", false}, // 非 IP 域名:无从判断,不误报
		{"::bad url::", false},
	}
	for _, c := range cases {
		if got := webURLExposed(c.url); got != c.want {
			t.Errorf("webURLExposed(%q) = %v, 期望 %v", c.url, got, c.want)
		}
	}
}
