package review

import (
	"testing"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
)

func TestWalkTreeLeaves(t *testing.T) {
	tree := []fenbi.KeypointNode{
		{
			ID: 1, Name: "模块A",
			QuestionIDs: []int64{101, 102}, // 有题但子级也有题 → 不作为叶子
			Children: []fenbi.KeypointNode{
				{ID: 11, Name: "子类A1", QuestionIDs: []int64{111}},
				{ID: 12, Name: "子类A2", QuestionIDs: nil, Children: []fenbi.KeypointNode{
					{ID: 121, Name: "孙类A21", QuestionIDs: []int64{1211, 1212}},
				}},
			},
		},
		{
			ID: 2, Name: "模块B",
			QuestionIDs: []int64{201}, // 有题且无子级 → 根级叶子
		},
		{ID: 3, Name: "空模块", QuestionIDs: nil},
	}

	var leafs []struct {
		ID   int64
		Name string
		IDs  []int64
	}
	walkTree(tree, "", &leafs)

	if len(leafs) != 3 {
		t.Fatalf("leafs = %d, want 3 (子类A1/孙类A21/模块B)", len(leafs))
	}
	names := map[string]bool{}
	for _, l := range leafs {
		names[l.Name] = true
	}
	for _, want := range []string{"模块A / 子类A1", "模块A / 子类A2 / 孙类A21", "模块B"} {
		if !names[want] {
			t.Errorf("缺少叶子: %s (实际: %v)", want, names)
		}
	}
	// 根级叶子（模块B）不含分隔符 → buildGroups 中 segments[1] 守卫应生效
	for _, l := range leafs {
		if l.Name == "模块B" && l.ID != 2 {
			t.Errorf("模块B id = %d, want 2", l.ID)
		}
	}
}
