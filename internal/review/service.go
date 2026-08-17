package review

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// Service 六步复盘业务逻辑。
type Service struct {
	Fenbi *fenbi.Client
	Store *store.Store
}

func NewService(f *fenbi.Client, s *store.Store) *Service {
	return &Service{Fenbi: f, Store: s}
}

// Group 页面上的一个知识点分组。
type Group struct {
	ID       int64         `json:"id"`
	Name     string        `json:"name"`
	Module   string        `json:"module"`
	Sub      string        `json:"sub"`
	Count    int           `json:"count"`
	Items    []QuestionItem `json:"items"`
}

// QuestionItem 带复盘信息的题目条目。
type QuestionItem struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	Difficulty    int    `json:"difficulty"`
	ErrorCategory string `json:"errorCategory,omitempty"`
	ReviewState   string `json:"reviewState,omitempty"`
	HasNote4      bool   `json:"hasNote4"`
	Archived      bool   `json:"archived"`
	RedoCount     int    `json:"redoCount"`
}

func typeName(t int) string {
	switch t {
	case 1:
		return "单选"
	case 2:
		return "多选"
	case 3:
		return "判断"
	case 4:
		return "填空"
	case 5:
		return "材料"
	}
	return "题目"
}

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<p>", " ")
	s = strings.ReplaceAll(s, "</p>", " ")
	s = strings.ReplaceAll(s, "<br>", " ")
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 120 {
		out = out[:120] + "…"
	}
	return out
}

// walkTree 收集有题目且子级无题目的叶子节点。
func walkTree(nodes []fenbi.KeypointNode, path string, out *[]struct {
	ID   int64
	Name string
	IDs  []int64
}) {
	for _, n := range nodes {
		name := n.Name
		if path != "" {
			name = path + " / " + n.Name
		}
		childHasQ := false
		for _, c := range n.Children {
			if len(c.QuestionIDs) > 0 {
				childHasQ = true
				break
			}
		}
		if len(n.QuestionIDs) > 0 && !childHasQ {
			*out = append(*out, struct {
				ID   int64
				Name string
				IDs  []int64
			}{n.ID, name, n.QuestionIDs})
		}
		walkTree(n.Children, name, out)
	}
}

// WrongBookData 错题本：知识点分组 + 复盘信息 + 时间筛选。
func (s *Service) WrongBookData(tr fenbi.TimeRange, excludeArchived bool) ([]Group, error) {
	tree, err := s.Fenbi.ErrorTreeNode(tr)
	if err != nil {
		return nil, err
	}
	var leafs []struct {
		ID   int64
		Name string
		IDs  []int64
	}
	walkTree(tree, "", &leafs)
	return s.buildGroups(leafs, excludeArchived)
}

// CollectData 收藏：知识点分组 + 复盘信息。
func (s *Service) CollectData(tr fenbi.TimeRange, excludeArchived bool) ([]Group, error) {
	tree, err := s.Fenbi.CollectTreeNode(tr)
	if err != nil {
		return nil, err
	}
	var leafs []struct {
		ID   int64
		Name string
		IDs  []int64
	}
	walkTree(tree, "", &leafs)
	return s.buildGroups(leafs, excludeArchived)
}

func (s *Service) buildGroups(leafs []struct {
	ID   int64
	Name string
	IDs  []int64
}, excludeArchived bool) ([]Group, error) {
	reviews := s.Store.AllQuestions()

	// 并发拉取各分组题目
	type leafResult struct {
		leaf struct {
			ID   int64
			Name string
			IDs  []int64
		}
		byID map[int64]fenbi.Question
	}
	results := make(chan leafResult, len(leafs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // 并发上限，避免触发粉笔 API 频率限制
	for _, leaf := range leafs {
		if len(leaf.IDs) == 0 {
			continue
		}
		wg.Add(1)
		go func(leaf struct {
			ID   int64
			Name string
			IDs  []int64
		}) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			qs, err := s.Fenbi.Questions(leaf.IDs)
			if err != nil {
				return
			}
			byID := map[int64]fenbi.Question{}
			for _, q := range qs {
				byID[q.ID] = q
			}
			results <- leafResult{leaf: leaf, byID: byID}
		}(leaf)
	}
	wg.Wait()
	close(results)

	var groups []Group
	for r := range results {
		leaf := r.leaf
		byID := r.byID
		var items []QuestionItem
		for _, id := range leaf.IDs {
			q, ok := byID[id]
			if !ok {
				continue
			}
			rv := reviews[strconv.FormatInt(id, 10)]
			if rv == nil {
				rv = &store.QuestionReview{ReviewState: store.StatePending}
			}
			if excludeArchived && rv.Archived {
				continue
			}
			items = append(items, QuestionItem{
				ID:            q.ID,
				Title:         stripHTML(q.Content),
				Type:          typeName(q.Type),
				Difficulty:    q.Difficulty,
				ErrorCategory: rv.ErrorCategory,
				ReviewState:   rv.ReviewState,
				HasNote4:      rv.Note4 != nil,
				Archived:      rv.Archived,
				RedoCount:     len(rv.RedoHistory),
			})
		}
		if len(items) == 0 {
			continue
		}
		segments := strings.Split(leaf.Name, " / ")
		groups = append(groups, Group{
			ID:     leaf.ID,
			Name:   leaf.Name,
			Module: segments[0],
			Sub:    segments[1],
			Count:  len(items),
			Items:  items,
		})
	}
	return groups, nil
}

// ReviewEntry 复习队列条目。
type ReviewEntry struct {
	QuestionID    int64  `json:"questionId"`
	Title         string `json:"title"`
	ErrorCategory string `json:"errorCategory,omitempty"`
	ReviewState   string `json:"reviewState"`
	RedoCount     int    `json:"redoCount"`
	ConsecutiveOK int    `json:"consecutiveOK"`
}

// ReviewQueue 复习队列：待复习 + 重点。
func (s *Service) ReviewQueue() ([]ReviewEntry, error) {
	reviews := s.Store.AllQuestions()
	var ids []int64
	for qid, rv := range reviews {
		if rv.Archived {
			continue
		}
		if rv.ReviewState == store.StatePending || rv.ReviewState == store.StateFlagged {
			id, err := strconv.ParseInt(qid, 10, 64)
			if err == nil {
				ids = append(ids, id)
			}
		}
	}
	entries := s.enrichEntries(ids, reviews)
	// 重点优先
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].ReviewState != entries[j].ReviewState {
			return entries[i].ReviewState == store.StateFlagged
		}
		return false
	})
	return entries, nil
}

func (s *Service) enrichEntries(ids []int64, reviews map[string]*store.QuestionReview) []ReviewEntry {
	var entries []ReviewEntry
	if len(ids) == 0 {
		return entries
	}
	qs, err := s.Fenbi.Questions(ids)
	if err != nil {
		return entries
	}
	byID := map[int64]fenbi.Question{}
	for _, q := range qs {
		byID[q.ID] = q
	}
	for _, id := range ids {
		q, ok := byID[id]
		if !ok {
			continue
		}
		rv := reviews[strconv.FormatInt(id, 10)]
		consec := 0
		if rv != nil {
			for i := len(rv.RedoHistory) - 1; i >= 0; i-- {
				if rv.RedoHistory[i].Result != "correct" {
					break
				}
				consec++
			}
		}
		entries = append(entries, ReviewEntry{
			QuestionID:    q.ID,
			Title:         stripHTML(q.Content),
			ErrorCategory: rv.ErrorCategory,
			ReviewState:   rv.ReviewState,
			RedoCount:     len(rv.RedoHistory),
			ConsecutiveOK: consec,
		})
	}
	return entries
}

// Today 今天是第几天用于（暂保留，供后续扩展）。
func Today() string {
	return time.Now().Format("2006-01-02")
}

// UsedCategories 收集已用过的错误归类标签（预设 + 自定义，去重）。
func (s *Service) UsedCategories() []string {
	all := s.Store.AllQuestions()
	seen := map[string]bool{}
	for _, name := range store.ErrorCategories {
		seen[name] = true
	}
	var out []string
	for _, name := range store.ErrorCategories {
		out = append(out, name)
	}
	for _, rv := range all {
		if rv.ErrorCategory == "" || seen[rv.ErrorCategory] {
			continue
		}
		seen[rv.ErrorCategory] = true
		out = append(out, rv.ErrorCategory)
	}
	return out
}
