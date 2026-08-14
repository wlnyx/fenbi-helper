package web

import (
	"net/http"
	"strconv"

	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// handleExercise 练习详情：报告 + 宏观总结卡（六步第五步）。
func (s *Server) handleExercise(w http.ResponseWriter, r *http.Request) {
	eidStr := r.PathValue("id")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rep, err := s.Fenbi.ExerciseReport(eid)
	if err != nil {
		http.Error(w, "报告加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	correctRate := 0.0
	if rep.AnswerCount > 0 {
		correctRate = float64(rep.CorrectCount) / float64(rep.AnswerCount) * 100
	}
	ex := s.Store.Exercise(eidStr)

	renderPage(w, "exercise", map[string]interface{}{
		"EID":            eidStr,
		"Report":         rep,
		"CorrectRate":    correctRate,
		"Macro":          ex.Macro,
		"HasMacro":       ex.Macro != nil,
		"Nav":            navItems("history"),
		"ErrorCategories": store.ErrorCategories,
	})
}
