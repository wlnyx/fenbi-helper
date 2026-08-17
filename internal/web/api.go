package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/review"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// JSON API：供 Vue SPA 使用（页面数据 + 复盘操作）。

type apiGroup struct {
	ID     int64         `json:"id"`
	Name   string        `json:"name"`
	Module string        `json:"module"`
	Sub    string        `json:"sub"`
	Count  int           `json:"count"`
	Items  []apiQuestion `json:"items"`
}

type apiQuestion struct {
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

func toAPIGroups(groups []review.Group) []apiGroup {
	var out []apiGroup
	for _, g := range groups {
		var items []apiQuestion
		for _, it := range g.Items {
			items = append(items, apiQuestion{
				ID:            it.ID,
				Title:         it.Title,
				Type:          it.Type,
				Difficulty:    it.Difficulty,
				ErrorCategory: it.ErrorCategory,
				ReviewState:   it.ReviewState,
				HasNote4:      it.HasNote4,
				Archived:      it.Archived,
				RedoCount:     it.RedoCount,
			})
		}
		out = append(out, apiGroup{ID: g.ID, Name: g.Name, Module: g.Module, Sub: g.Sub, Count: g.Count, Items: items})
	}
	return out
}

// 注册 JSON API 路由。
func (s *Server) apiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/wrong", s.requireAuth(s.apiWrong))
	mux.HandleFunc("GET /api/collects", s.requireAuth(s.apiCollects))
	mux.HandleFunc("GET /api/history", s.requireAuth(s.apiHistory))
	mux.HandleFunc("GET /api/question/{id}", s.requireAuth(s.apiQuestion))
	mux.HandleFunc("GET /api/exercise/{id}", s.requireAuth(s.apiExercise))
	mux.HandleFunc("GET /api/dashboard", s.requireAuth(s.apiDashboard))
	mux.HandleFunc("GET /api/session/info", s.requireAuth(s.apiSessionInfo))
	mux.HandleFunc("POST /api/session/logout", s.requireAuth(s.apiSessionLogout))
	mux.HandleFunc("GET /api/review/queue", s.requireAuth(s.apiReviewQueue))
}

// apiReviewQueue 复习队列（重点/待复习/已掌握 分组）。
func (s *Server) apiReviewQueue(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Review.ReviewQueue()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	var pending, flagged, mastered []map[string]interface{}
	pending = make([]map[string]interface{}, 0)
	flagged = make([]map[string]interface{}, 0)
	mastered = make([]map[string]interface{}, 0)
	for _, e := range entries {
		item := map[string]interface{}{
			"questionId": e.QuestionID, "title": e.Title,
			"errorCategory": e.ErrorCategory, "reviewState": e.ReviewState,
			"redoCount": e.RedoCount, "consecutiveOK": e.ConsecutiveOK,
		}
		switch e.ReviewState {
		case "flagged":
			flagged = append(flagged, item)
		case "mastered":
			mastered = append(mastered, item)
		default:
			pending = append(pending, item)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 200, "pending": pending, "flagged": flagged, "mastered": mastered,
	})
}

// apiSessionInfo 当前登录账号信息。
func (s *Server) apiSessionInfo(w http.ResponseWriter, r *http.Request) {
	dev, cookies := s.Auth.LoadCredentials()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":        200,
		"userId":      dev.UserID,
		"deviceId":    dev.DeviceID,
		"loginTime":   dev.Time,
		"cookieCount": len(cookies),
	})
}

// apiSessionLogout 退出登录：清除浏览器会话标记 + 服务端凭据。
func (s *Server) apiSessionLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "fb_device",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0), // 兼容不支持 MaxAge 的旧代理
		SameSite: http.SameSiteLaxMode,
	})
	_ = s.Auth.ClearCredentials()
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200})
}

func (s *Server) apiWrong(w http.ResponseWriter, r *http.Request) {
	key := "wrong:" + r.URL.RawQuery
	if v, ok := s.cache.Get(key); ok {
		writeJSON(w, http.StatusOK, v)
		return
	}
	tr := parseTimeRange(r)
	groups, err := s.Review.WrongBookData(tr, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	groups = filterGroups(groups, r.URL.Query().Get("module"), r.URL.Query().Get("sub"))
	resp := map[string]interface{}{"code": 200, "groups": toAPIGroups(groups)}
	s.cache.Set(key, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiCollects(w http.ResponseWriter, r *http.Request) {
	key := "collects:" + r.URL.RawQuery
	if v, ok := s.cache.Get(key); ok {
		writeJSON(w, http.StatusOK, v)
		return
	}
	tr := parseTimeRange(r)
	groups, err := s.Review.CollectData(tr, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	groups = filterGroups(groups, r.URL.Query().Get("module"), r.URL.Query().Get("sub"))
	resp := map[string]interface{}{"code": 200, "groups": toAPIGroups(groups)}
	s.cache.Set(key, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiHistory(w http.ResponseWriter, r *http.Request) {
	key := "history"
	if v, ok := s.cache.Get(key); ok {
		writeJSON(w, http.StatusOK, v)
		return
	}
	history, err := s.Fenbi.History()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	resp := map[string]interface{}{"code": 200, "history": history}
	s.cache.Set(key, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiQuestion(w http.ResponseWriter, r *http.Request) {
	qid, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": "id 无效"})
		return
	}
	key := "question:" + strconv.FormatInt(qid, 10)
	if v, ok := s.questionCache.Get(key); ok {
		writeJSON(w, http.StatusOK, v)
		return
	}
	qs, err := s.Fenbi.Questions([]int64{qid})
	if err != nil || len(qs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"code": 404, "msg": "题目不存在"})
		return
	}
	q := qs[0]
	sols, _ := s.Fenbi.Solutions([]int64{qid})
	sol := fenbi.Solution{}
	if len(sols) > 0 {
		sol = sols[0]
	}
	rv := s.Store.Question(strconv.FormatInt(qid, 10))
	resp := map[string]interface{}{
		"code":           200,
		"question":       q,
		"solution":       sol,
		"review":         rv,
		"categories":     store.ErrorCategories,
		"usedCategories": s.Review.UsedCategories(),
	}
	s.questionCache.Set(key, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiExercise(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": "id 无效"})
		return
	}
	rep, err := s.Fenbi.ExerciseReport(eid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"code": 404, "msg": "报告不存在"})
		return
	}
	items, _ := s.Fenbi.ExerciseDetail(eid)
	ex := s.Store.Exercise(strconv.FormatInt(eid, 10))
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200, "report": rep, "items": items, "macro": ex.Macro})
}

func (s *Server) apiDashboard(w http.ResponseWriter, r *http.Request) {
	if v, ok := s.cache.Get("dashboard"); ok {
		writeJSON(w, http.StatusOK, v)
		return
	}
	data := map[string]interface{}{
		"totalReview": 0, "flagged": 0, "mastered": 0, "unreviewed": 0,
		"categoryCount": map[string]int{},
		"queue":         []interface{}{},
		"recentHistory": []interface{}{},
	}
	entries, err := s.Review.ReviewQueue()
	if err == nil {
		queue := make([]map[string]interface{}, 0)
		for _, e := range entries {
			queue = append(queue, map[string]interface{}{
				"questionId": e.QuestionID, "title": e.Title,
				"errorCategory": e.ErrorCategory, "reviewState": e.ReviewState,
				"consecutiveOK": e.ConsecutiveOK,
			})
			if e.ReviewState == store.StateFlagged {
				data["flagged"] = data["flagged"].(int) + 1
			} else {
				data["totalReview"] = data["totalReview"].(int) + 1
			}
		}
		data["queue"] = queue
	}
	all := s.Store.AllQuestions()
	cc := data["categoryCount"].(map[string]int)
	for _, rv := range all {
		if rv.Archived {
			continue
		}
		switch rv.ReviewState {
		case store.StateMastered:
			data["mastered"] = data["mastered"].(int) + 1
		case "":
			data["unreviewed"] = data["unreviewed"].(int) + 1
		}
		if rv.ErrorCategory != "" {
			cc[rv.ErrorCategory]++
		}
	}
	// recentHistory 是粉笔侧数据（用户完成新练习会变化），用 60s 短缓存，
	// 避免与 5 分钟统计缓存绑定导致"近期练习"长期陈旧。
	recentHistory := []interface{}{}
	if v, ok := s.cache.Get("dashboard-recent"); ok {
		recentHistory = v.([]interface{})
	} else {
		hist, err := s.Fenbi.History()
		if err == nil {
			recent := make([]map[string]interface{}, 0)
			for i, h := range hist {
				if i >= 5 {
					break
				}
				recent = append(recent, map[string]interface{}{
					"id": h.ID, "name": h.Sheet.Name,
					"answerCount": h.AnswerCount, "correctRate": h.CorrectRate,
				})
			}
			recentHistory = make([]interface{}, len(recent))
			for i, r := range recent {
				recentHistory[i] = r
			}
			s.cache.SetWithTTL("dashboard-recent", recentHistory, 60*time.Second)
		}
	}
	data["recentHistory"] = recentHistory
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200, "data": data, "categories": store.ErrorCategories})
}

// invalidateDataCache 复盘写操作后清空列表类缓存与对应题目缓存。
func (s *Server) invalidateDataCache(questionID string) {
	for _, prefix := range []string{"wrong:", "collects:", "history", "dashboard", "review:"} {
		s.cache.Invalidate(prefix)
	}
	if questionID != "" {
		s.questionCache.Invalidate("question:" + questionID)
	}
}
