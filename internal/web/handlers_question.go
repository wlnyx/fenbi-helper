package web

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// handleQuestion 单题页：答题模式 + 六步复盘面板。
func (s *Server) handleQuestion(w http.ResponseWriter, r *http.Request) {
	qidStr := r.PathValue("id")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	qs, err := s.Fenbi.Questions([]int64{qid})
	if err != nil || len(qs) == 0 {
		http.Error(w, "题目加载失败", http.StatusInternalServerError)
		return
	}
	q := qs[0]
	sols, _ := s.Fenbi.Solutions([]int64{qid})
	sol := fenbi.Solution{}
	if len(sols) > 0 {
		sol = sols[0]
	}
	rv := s.Store.Question(qidStr)

	renderPage(w, "question", map[string]interface{}{
		"Q":             q,
		"Sol":           sol,
		"QID":           qid,
		"Options":       optionLetters(q),
		"RV":            rv,
		"Categories":    store.ErrorCategories,
		"CategoriesJSON": mustJSON(store.ErrorCategories),
		"Nav":           navItems(""),
	})
}

// optionLetters 生成 A/B/C/D 选项标签。
func optionLetters(q fenbi.Question) []string {
	var letters []string
	for _, acc := range q.Accessories {
		for i := range acc.Options {
			letters = append(letters, string(rune('A'+i)))
		}
		break
	}
	return letters
}

var _ = template.HTMLEscapeString
