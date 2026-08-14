package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/wlnyx/fenbi-helper-go/internal/auth"
	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
	"github.com/wlnyx/fenbi-helper-go/internal/review"
	"github.com/wlnyx/fenbi-helper-go/internal/store"
)

// Server 聚合依赖。
type Server struct {
	Auth          *auth.Store
	Fenbi         *fenbi.Client
	QR            *fenbi.QRLogin
	Store         *store.Store
	Review        *review.Service
	DataDir       string
	cache         *memCache
	questionCache *memCache
}

func NewServer(dataDir string) (*Server, error) {
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
		Auth:          authStore,
		Fenbi:         client,
		QR:            fenbi.NewQRLogin(client),
		Store:         dataStore,
		Review:        review.NewService(client, dataStore),
		DataDir:       dataDir,
		cache:         newMemCache(5 * time.Minute),
		questionCache: newMemCache(time.Hour),
	}, nil
}

// Routes 注册全部路由。
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

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

	// JSON API（Vue SPA）
	s.apiRoutes(mux)

	// 健康检查
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// SPA 前端（Vue 构建产物，已 embed 进二进制）
	mux.HandleFunc("GET /assets/", s.handleAsset)
	mux.HandleFunc("GET /", s.handleSPA)

	return logMiddleware(mux)
}

// handleSPA 所有非 API 路径返回前端入口（前端路由接管）。
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	index, err := frontend.ReadFile("dist/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(index)
}

// handleAsset 前端静态资源。
func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := "dist/" + strings.TrimPrefix(r.URL.Path, "/")
	content, err := frontend.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := "text/plain"
	switch {
	case strings.HasSuffix(name, ".css"):
		ct = "text/css"
	case strings.HasSuffix(name, ".js"):
		ct = "application/javascript"
	case strings.HasSuffix(name, ".png"):
		ct = "image/png"
	case strings.HasSuffix(name, ".svg"):
		ct = "image/svg+xml"
	case strings.HasSuffix(name, ".woff2"):
		ct = "font/woff2"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(content)
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
			// API 返回 401 JSON；页面重定向到登录页
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"code": 401, "msg": "未登录"})
				return
			}
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// isLoggedIn 双重校验：客户端 cookie 标记 + 服务端持久化凭据必须同时存在。
// 仅伪造 Cookie: fb_device=1 无法通过（服务端 device.json 不存在时拒绝）。
func (s *Server) isLoggedIn(r *http.Request) bool {
	c, err := r.Cookie("fb_device")
	if err != nil || c.Value != "1" {
		return false
	}
	return s.Auth.HasSession()
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
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": "二维码保存失败"})
		return
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

// --- 工具 ---

// mustJSON 把值序列化为 JSON 字符串。
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
