package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/wlnyx/fenbi-helper-go/internal/auth"
	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/review"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

//go:embed templates/* static/*
var assets embed.FS

// Server 聚合依赖。
type Server struct {
	Auth    *auth.Store
	Fenbi   *fenbi.Client
	QR      *fenbi.QRLogin
	Store   *store.Store
	Review  *review.Service
	DataDir string
}

func NewServer(dataDir string) (*Server, error) {
	if err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		return nil
	}); err != nil {
		return nil, err
	}
	client := fenbi.NewClient()
	// 恢复持久化凭据
	authStore := auth.NewStore(dataDir)
	dev, cookies := authStore.LoadCredentials()
	if dev != nil && dev.DeviceID != "" {
		client.SetDeviceID(dev.DeviceID)
		client.LoadSession(cookies)
	}
	dataStore := store.NewStore(dataDir)
	if dev != nil && dev.UserID != 0 {
		dataStore.SetUser(dev.UserID)
	}
	return &Server{
		Auth:    authStore,
		Fenbi:   client,
		QR:      fenbi.NewQRLogin(client),
		Store:   dataStore,
		Review:  review.NewService(client, dataStore),
		DataDir: dataDir,
	}, nil
}

// Routes 注册全部路由。
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// 页面
	mux.HandleFunc("GET /setup", s.handleSetup)
	mux.HandleFunc("GET /dashboard", s.requireAuth(s.handleDashboard))
	mux.HandleFunc("GET /history", s.requireAuth(s.handleHistory))
	mux.HandleFunc("GET /wrong", s.requireAuth(s.handleWrong))
	mux.HandleFunc("GET /collects", s.requireAuth(s.handleCollects))
	mux.HandleFunc("GET /review", s.requireAuth(s.handleReviewQueue))
	mux.HandleFunc("GET /question/{id}", s.requireAuth(s.handleQuestion))
	mux.HandleFunc("GET /exercise/{id}", s.requireAuth(s.handleExercise))
	mux.HandleFunc("GET /tools", s.requireAuth(s.handleTools))

	// 扫码登录 API
	mux.HandleFunc("POST /api/qr/start", s.handleQRStart)
	mux.HandleFunc("GET /api/qr/status", s.handleQRStatus)
	mux.HandleFunc("POST /api/qr/finish", s.handleQRFinish)
	mux.HandleFunc("GET /api/qr.png", s.handleQRPNG)
	mux.HandleFunc("POST /api/session/apply", s.handleSessionApply)

	// 复盘 API
	mux.HandleFunc("POST /api/review/update", s.requireAuth(s.handleReviewUpdate))
	mux.HandleFunc("POST /api/review/note4", s.requireAuth(s.handleNote4))
	mux.HandleFunc("POST /api/review/summary", s.requireAuth(s.handleSummary))

	// 静态资源
	mux.HandleFunc("GET /static/", s.handleStatic)

	// 健康检查
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	return logMiddleware(mux)
}

// --- 中间件 ---

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isLoggedIn(r) {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) isLoggedIn(r *http.Request) bool {
	c, err := r.Cookie("fb_device")
	if err != nil || c.Value != "1" {
		return false
	}
	return true
}

// --- 页面 ---

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	tpl := mustTemplate("setup")
	tpl.Execute(w, map[string]interface{}{})
}

// --- 扫码登录 API ---

func (s *Server) handleQRStart(w http.ResponseWriter, r *http.Request) {
	qrText, _, err := s.QR.Start()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	png, err := qrcode.Encode(qrText, qrcode.Medium, 240)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": "二维码生成失败"})
		return
	}
	if err := s.Auth.SaveQrPNG(png); err != nil {
		log.Printf("保存二维码失败: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200})
}

func (s *Server) handleQRStatus(w http.ResponseWriter, r *http.Request) {
	state, msg, err := s.QR.Status()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"state": "error", "msg": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": state, "msg": msg})
}

func (s *Server) handleQRFinish(w http.ResponseWriter, r *http.Request) {
	userID, cookies, err := s.QR.Finish()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"code": 500, "msg": err.Error()})
		return
	}
	// 持久化 deviceId + 会话 Cookie
	dev := auth.DeviceID{DeviceID: s.Fenbi.DeviceID(), UserID: userID, Time: time.Now().UnixMilli()}
	if err := s.Auth.SaveCredentials(dev, cookies); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"code": 500, "msg": "凭据保存失败: " + err.Error()})
		return
	}
	s.Fenbi.LoadSession(cookies)
	s.Store.SetUser(userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200, "userId": userID})
}

func (s *Server) handleQRPNG(w http.ResponseWriter, r *http.Request) {
	png, err := s.Auth.LoadQrPNG()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}

func (s *Server) handleSessionApply(w http.ResponseWriter, r *http.Request) {
	if !s.Auth.HasSession() {
		writeJSON(w, http.StatusOK, map[string]interface{}{"code": 500, "msg": "未获取到登录凭证"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "fb_device",
		Value:   "1",
		Path:    "/",
		Expires: time.Date(2099, 7, 6, 0, 0, 0, 0, time.UTC),
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"code": 200, "redirectPath": "/dashboard"})
}

// --- 静态资源 ---

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	content, err := assets.ReadFile("static" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := "text/plain"
	switch {
	case hasSuffix(name, ".css"):
		ct = "text/css"
	case hasSuffix(name, ".js"):
		ct = "application/javascript"
	case hasSuffix(name, ".png"):
		ct = "image/png"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(content)
}

// --- 工具 ---

// NavItem 导航项。
type NavItem struct {
	Label  string
	Href   string
	Active bool
}

func navItems(active string) []NavItem {
	items := []struct {
		key, label, href string
	}{
		{"dashboard", "🎯 复盘工作台", "/dashboard"},
		{"history", "📋 练习历史", "/history"},
		{"wrong", "❌ 错题本", "/wrong"},
		{"collects", "⭐ 我的收藏", "/collects"},
		{"review", "📅 复习队列", "/review"},
		{"tools", "🧮 小工具", "/tools"},
	}
	var out []NavItem
	for _, it := range items {
		out = append(out, NavItem{Label: it.label, Href: it.href, Active: it.key == active})
	}
	return out
}

func renderPage(w http.ResponseWriter, name string, data map[string]interface{}) {
	t := mustTemplate(name)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := t.Execute(w, data); err != nil {
		log.Printf("模板渲染 %s 失败: %v", name, err)
	}
}

// mustJSON 把值序列化为 JSON 字符串（用于模板内嵌）。
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func mustTemplate(name string) *template.Template {
	t, err := templateNew(name)
	if err != nil {
		panic(fmt.Sprintf("template %s: %v", name, err))
	}
	return t
}
