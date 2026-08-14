package review

import "testing"

func TestTypeName(t *testing.T) {
	cases := map[int]string{1: "单选", 2: "多选", 3: "判断", 4: "填空", 5: "材料", 99: "题目"}
	for in, want := range cases {
		if got := typeName(in); got != want {
			t.Errorf("typeName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>考点内容</p>", "考点内容"},
		{"<b>加粗</b> 与 <i>斜体</i>", "加粗 与 斜体"},
	}
	for _, c := range cases {
		if got := stripHTML(c.in); got != c.want {
			t.Errorf("stripHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
