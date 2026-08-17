package fenbi

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	genCodeURL  = "https://ke.fenbi.com/qrcode-login/api/gen_code"
	statusURL   = "https://ke.fenbi.com/qrcode-login/api/query_code_status"
	infoURL     = "https://login.fenbi.com/api/users/info?" + apiParams
	sidCreate   = "https://login.fenbi.com/api/users/device/sid/create?app=web&av=100&hav=100&kav=100&gav=2&apcid=0&client_context_id="
	qrExpirySec = 100
)

// Cookie 是持久化的会话 Cookie。
type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// QRLogin 管理单次扫码登录会话。
type QRLogin struct {
	mu       sync.Mutex
	client   *Client
	jar      *SessionJar
	running  bool
	lgtoken  string
	qrText   string
	deviceID string
	started  time.Time
}

func NewQRLogin(client *Client) *QRLogin {
	return &QRLogin{client: client, jar: NewSessionJar()}
}

// Start 创建 deviceId 并生成二维码。
func (q *QRLogin) Start() (qrText string, lgtoken string, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 已有会话且超时则重置
	if q.running && time.Since(q.started) > qrExpirySec*time.Second {
		q.reset()
	}
	if q.running {
		return q.qrText, q.lgtoken, nil
	}

	// 1. 创建 deviceId
	deviceID, err := q.createDeviceID()
	if err != nil {
		return "", "", fmt.Errorf("创建 deviceId 失败: %w", err)
	}
	q.deviceID = deviceID
	q.client.setDeviceID(deviceID)

	// 2. gen_code
	u := genCodeURL + "?random=0." + strconv.FormatInt(time.Now().UnixNano(), 36) + "&deviceId=" + url.QueryEscape(deviceID)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := q.jar.Client().Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var gen struct {
		Code int `json:"code"`
		Data struct {
			Lgtoken     string `json:"lgtoken"`
			CodeContent string `json:"codeContent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &gen); err != nil {
		return "", "", err
	}
	if gen.Data.Lgtoken == "" {
		return "", "", fmt.Errorf("gen_code 失败: %s", string(body))
	}

	q.lgtoken = gen.Data.Lgtoken
	q.qrText = gen.Data.CodeContent
	q.started = time.Now()
	q.running = true
	return q.qrText, q.lgtoken, nil
}

// Status 查询扫码状态：waiting/scanned/authorized/expired/canceled/failed。
func (q *QRLogin) Status() (state string, msg string, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.running || q.lgtoken == "" {
		return "error", "尚未开始扫码登录", nil
	}
	if time.Since(q.started) > qrExpirySec*time.Second {
		q.reset()
		return "expired", "二维码已失效", nil
	}

	payload := fmt.Sprintf(`{"lgtoken":%q}`, q.lgtoken)
	req, _ := http.NewRequest(http.MethodPost, statusURL, strings.NewReader(payload))
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.jar.Client().Do(req)
	if err != nil {
		return "error", err.Error(), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var st struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data int    `json:"data"`
	}
	if err := json.Unmarshal(body, &st); err != nil {
		return "error", "响应解析失败", nil
	}
	switch st.Data {
	case 0:
		q.reset()
		return "expired", "二维码已失效", nil
	case 1:
		return "waiting", "等待扫码", nil
	case 2:
		return "scanned", "已扫码，请在手机上确认", nil
	case 3:
		return "authorized", "授权成功", nil
	case 4:
		q.reset()
		return "canceled", "已取消", nil
	case 5:
		q.reset()
		return "failed", "登录失败", nil
	}
	return "error", st.Msg, nil
}

// Finish 授权后确认登录，返回 userId 与会话 Cookie。
func (q *QRLogin) Finish() (userID int64, cookies []Cookie, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.running || q.deviceID == "" {
		return 0, nil, fmt.Errorf("无进行中的扫码会话")
	}

	// info 确认登录（响应会下发会话 Cookie 到 jar）
	req, _ := http.NewRequest(http.MethodGet, q.client.attachDeviceID(infoURL), nil)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := q.jar.Client().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var info struct {
		UserID int64 `json:"userId"`
	}
	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("登录态确认失败(HTTP %d): %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &info); err != nil || info.UserID == 0 {
		return 0, nil, fmt.Errorf("登录态确认失败: %s", string(body))
	}

	// 收集 jar 中的会话 Cookie
	cookies = q.jar.DumpCookies()
	q.running = false
	return info.UserID, cookies, nil
}

func (q *QRLogin) createDeviceID() (string, error) {
	payload := fmt.Sprintf(`{"pf":"web","startupId":%q,"extras":{}}`, strconv.FormatInt(time.Now().UnixMilli(), 10))
	req, _ := http.NewRequest(http.MethodPost, sidCreate+randomHex(6), strings.NewReader(payload))
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.jar.Client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Data struct {
			DeviceID string `json:"deviceId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.Data.DeviceID == "" {
		return "", fmt.Errorf("device/sid/create 失败: %s", string(body))
	}
	return r.Data.DeviceID, nil
}

func (q *QRLogin) reset() {
	q.running = false
	q.lgtoken = ""
	q.qrText = ""
	q.jar = NewSessionJar()
}

// randomHex 生成随机 hex 串，用作粉笔 API 的 client_context_id 参数。
func randomHex(n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = hexChars[b[i]&0xf]
	}
	return string(b)
}
