package service

import "testing"

func TestExtractBvid(t *testing.T) {
	cases := []struct {
		name      string
		sourceURL string
		want      string
	}{
		{"标准链接", "https://www.bilibili.com/video/BV1DgxCzREbM", "BV1DgxCzREbM"},
		{"带查询参数", "https://www.bilibili.com/video/BV1DgxCzREbM?p=2", "BV1DgxCzREbM"},
		{"带尾斜杠", "https://www.bilibili.com/video/BV1DgxCzREbM/", "BV1DgxCzREbM"},
		{"非 B 站", "https://example.com/video/abc", ""},
		{"空", "", ""},
		{"前后空白", "  https://www.bilibili.com/video/BV1DgxCzREbM  ", "BV1DgxCzREbM"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractBvid(c.sourceURL); got != c.want {
				t.Fatalf("extractBvid(%q) = %q, want %q", c.sourceURL, got, c.want)
			}
		})
	}
}
