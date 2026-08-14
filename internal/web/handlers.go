package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/review"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// parseTimeRange 解析查询参数 timeRange=90|30|180|all|custom(start,end)。
// 默认近 90 天。
func parseTimeRange(r *http.Request) fenbi.TimeRange {
	q := r.URL.Query()
	switch q.Get("range") {
	case "all":
		return fenbi.TimeRange{All: true}
	case "30":
		return fenbi.TimeRange{Start: time.Now().AddDate(0, 0, -30), End: time.Now()}
	case "180":
		return fenbi.TimeRange{Start: time.Now().AddDate(0, 0, -180), End: time.Now()}
	case "custom":
		if st, err := time.Parse("2006-01-02", q.Get("start")); err == nil {
			en, err2 := time.Parse("2006-01-02", q.Get("end"))
			if err2 == nil {
				return fenbi.TimeRange{Start: st, End: en}
			}
		}
		return fenbi.TimeRange{Start: time.Now().AddDate(0, 0, -90), End: time.Now()}
	default: // 90
		return fenbi.TimeRange{Start: time.Now().AddDate(0, 0, -90), End: time.Now()}
	}
}

// filterGroups 按 module/sub 过滤分组。
func filterGroups(groups []review.Group, module, sub string) []review.Group {
	var out []review.Group
	for _, g := range groups {
		if module != "" && g.Module != module {
			continue
		}
		if sub != "" && g.Sub != sub {
			continue
		}
		out = append(out, g)
	}
	return out
}

func (s *Server) handleWrong(w http.ResponseWriter, r *http.Request) {
	groups, err := s.Review.WrongBookData(parseTimeRange(r), true)
	if err != nil {
		http.Error(w, "错题数据加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	groups = filterGroups(groups, r.URL.Query().Get("module"), r.URL.Query().Get("sub"))
	modules := collectModules(groups)
	renderPage(w, "wrong", map[string]interface{}{
		"Title":         "错题本",
		"Groups":        groups,
		"Modules":       modules,
		"Current":       r.URL.Query().Get("module"),
		"Sub":           r.URL.Query().Get("sub"),
		"Range":         r.URL.Query().Get("range"),
		"Nav":           navItems("wrong"),
		"Categories":    store.ErrorCategories,
		"CategoriesJSON": mustJSON(store.ErrorCategories),
	})
}

func (s *Server) handleCollects(w http.ResponseWriter, r *http.Request) {
	groups, err := s.Review.CollectData(parseTimeRange(r), true)
	if err != nil {
		http.Error(w, "收藏数据加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	groups = filterGroups(groups, r.URL.Query().Get("module"), r.URL.Query().Get("sub"))
	renderPage(w, "wrong", map[string]interface{}{
		"Title":         "我的收藏",
		"Groups":        groups,
		"Modules":       collectModules(groups),
		"Current":       r.URL.Query().Get("module"),
		"Sub":           r.URL.Query().Get("sub"),
		"Range":         r.URL.Query().Get("range"),
		"Nav":           navItems("collects"),
		"Categories":    store.ErrorCategories,
		"CategoriesJSON": mustJSON(store.ErrorCategories),
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	history, err := s.Fenbi.History()
	if err != nil {
		http.Error(w, "历史数据加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderPage(w, "history", map[string]interface{}{
		"History": history,
		"Nav":     navItems("history"),
	})
}

func (s *Server) handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Review.ReviewQueue()
	if err != nil {
		http.Error(w, "复习队列加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var pending, flagged, mastered []review.ReviewEntry
	for _, e := range entries {
		switch e.ReviewState {
		case store.StateFlagged:
			flagged = append(flagged, e)
		case store.StateMastered:
			mastered = append(mastered, e)
		default:
			pending = append(pending, e)
		}
	}
	renderPage(w, "review", map[string]interface{}{
		"Pending":  pending,
		"Flagged":  flagged,
		"Mastered": mastered,
		"Nav":      navItems("review"),
	})
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "tools", map[string]interface{}{"Nav": navItems("tools")})
}

func collectModules(groups []review.Group) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range groups {
		if !seen[g.Module] {
			seen[g.Module] = true
			out = append(out, g.Module)
		}
	}
	return out
}

// --- 复盘更新 API ---

func (s *Server) handleReviewUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		QuestionID string `json:"questionId"`
		Field      string `json:"field"`
		Value      string `json:"value"`
		Sub        string `json:"sub,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}
	if body.QuestionID == "" || body.Field == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": "参数缺失"})
		return
	}

	s.Store.UpdateQuestion(body.QuestionID, func(q *store.QuestionReview) {
		switch body.Field {
		case "categorize":
			q.ErrorCategory = body.Value
		case "doubt":
			q.Doubt = body.Value == "1"
		case "state":
			q.ReviewState = body.Value
		case "archive":
			q.Archived = body.Value == "1"
		case "redo":
			// value: correct|wrong（重做打卡）
			q.RedoHistory = append(q.RedoHistory, store.RedoRecord{
				Date:   time.Now().Format("2006-01-02"),
				Result: body.Value,
			})
		}
	})
	s.Store.Save()
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200})
}

func (s *Server) handleNote4(w http.ResponseWriter, r *http.Request) {
	var body struct {
		QuestionID string       `json:"questionId"`
		Note4      store.Note4  `json:"note4"`
		SyncFenbi  bool         `json:"syncFenbi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}
	n := body.Note4
	s.Store.UpdateQuestion(body.QuestionID, func(q *store.QuestionReview) {
		q.Note4 = &n
	})
	s.Store.Save()

	if body.SyncFenbi {
		qid, err := strconv.ParseInt(body.QuestionID, 10, 64)
		if err == nil {
			content := fmt.Sprintf("【考点】%s\n【错在哪里】%s\n【正确思路】%s\n【长效提醒】%s",
				n.Point, n.Mistake, n.Approach, n.Reminder)
			_ = s.Fenbi.SaveNote(qid, content)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200})
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExerciseID string            `json:"exerciseId"`
		Summary    store.MacroSummary `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}
	s.Store.UpdateExercise(body.ExerciseID, func(e *store.ExerciseReview) {
		e.Macro = &body.Summary
	})
	s.Store.Save()
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200})
}
