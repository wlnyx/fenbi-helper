package web

import (
	"net/http"

	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// DashboardData 工作台首页数据。
type DashboardData struct {
	TotalReview   int
	Flagged       int
	Mastered      int
	Unreviewed    int
	CategoryCount map[string]int
	Queue         []reviewEntryView
	RecentHistory []historyView
	Nav           []NavItem
}

type reviewEntryView struct {
	QuestionID    int64
	Title         string
	ErrorCategory string
	ReviewState   string
	ConsecutiveOK int
}

type historyView struct {
	ID          int64
	Name        string
	AnswerCount int
	CorrectRate float64
}

// handleDashboard 复盘工作台首页：今日复习 + 统计 + 近期练习。
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := DashboardData{
		CategoryCount: map[string]int{},
		Nav:           navItems("dashboard"),
	}

	// 复习队列 + 统计
	entries, err := s.Review.ReviewQueue()
	if err == nil {
		for _, e := range entries {
			if e.ReviewState == store.StateFlagged {
				data.Flagged++
			} else {
				data.TotalReview++
			}
			if e.ErrorCategory != "" {
				data.CategoryCount[e.ErrorCategory]++
			}
			data.Queue = append(data.Queue, reviewEntryView{
				QuestionID:    e.QuestionID,
				Title:         e.Title,
				ErrorCategory: e.ErrorCategory,
				ReviewState:   e.ReviewState,
				ConsecutiveOK: e.ConsecutiveOK,
			})
		}
	}

	// 全部题目复盘状态统计
	all := s.Store.AllQuestions()
	for _, rv := range all {
		if rv.Archived {
			continue
		}
		switch rv.ReviewState {
		case store.StateMastered:
			data.Mastered++
		case "":
			data.Unreviewed++
		}
		if rv.ErrorCategory != "" {
			data.CategoryCount[rv.ErrorCategory]++
		}
	}

	// 近期练习
	hist, err := s.Fenbi.History()
	if err == nil {
		for i, h := range hist {
			if i >= 5 {
				break
			}
			data.RecentHistory = append(data.RecentHistory, historyView{
				ID:          h.ID,
				Name:        h.Sheet.Name,
				AnswerCount: h.AnswerCount,
				CorrectRate: h.CorrectRate,
			})
		}
	}

	renderPage(w, "dashboard", map[string]interface{}{
		"Data":         data,
		"Categories":   store.ErrorCategories,
		"CategoriesJSON": mustJSON(store.ErrorCategories),
	})
}
