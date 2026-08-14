package fenbi

import (
	"testing"
	"time"
)

func TestTimeRangeParams(t *testing.T) {
	cases := []struct {
		name string
		tr   TimeRange
		want string
	}{
		{"all", TimeRange{All: true}, "timeRange=0"},
		{"90days", TimeRange{Start: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)}, "timeRange=-1&startDate=20260516&endDate=20260814"},
	}
	for _, c := range cases {
		if got := c.tr.params(); got != c.want {
			t.Errorf("%s: params = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStripHTMLContent(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<p>题干内容</p>", "题干内容"},
		{"<p><img src='x'/></p>文字", "文字"},
		{"  多 个    空格  ", "多 个 空格"},
	}
	for _, c := range cases {
		if got := stripHTMLContent(c.in); got != c.want {
			t.Errorf("stripHTMLContent(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAttachDeviceID(t *testing.T) {
	c := NewClient()
	c.SetDeviceID("v1_test")
	got := c.attachDeviceID("https://tiku.fenbi.com/api/xingce/questions?ids=1&app=web")
	if got != "https://tiku.fenbi.com/api/xingce/questions?ids=1&app=web&deviceId=v1_test" {
		t.Errorf("attachDeviceID 结果: %s", got)
	}
	// 已有 deviceId 不应重复
	got2 := c.attachDeviceID("https://tiku.fenbi.com/api/x?deviceId=old")
	if got2 != "https://tiku.fenbi.com/api/x?deviceId=old" {
		t.Errorf("不应重复注入: %s", got2)
	}
	// 无 deviceId 时不注入
	c2 := NewClient()
	if got3 := c2.attachDeviceID("https://tiku.fenbi.com/api/x"); got3 != "https://tiku.fenbi.com/api/x" {
		t.Errorf("无 deviceId 不应注入: %s", got3)
	}
}
