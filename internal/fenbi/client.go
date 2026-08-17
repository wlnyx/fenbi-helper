package fenbi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	apiParams = "app=web&kav=131&av=130&hav=128&version=3.0.0.0"
)

// Client 是粉笔 API 统一客户端：自动注入 deviceId 参数 + 转发会话 Cookie。
type Client struct {
	HTTP     *http.Client
	deviceID string
	session  []Cookie
}

func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

// LoadSession 载入持久化会话 Cookie（全域转发，与 Node 版一致）。
func (c *Client) LoadSession(cookies []Cookie) {
	c.session = cookies
}

// Session 返回当前会话 Cookie。
func (c *Client) Session() []Cookie {
	return c.session
}

func (c *Client) setDeviceID(deviceID string) {
	c.deviceID = deviceID
}

// SetDeviceID 公开设置 deviceId。
func (c *Client) SetDeviceID(deviceID string) {
	c.setDeviceID(deviceID)
}

// DeviceID 返回当前 deviceId。
func (c *Client) DeviceID() string {
	return c.deviceID
}

// attachDeviceID 给粉笔域 URL 追加 deviceId 参数（若未携带）。
func (c *Client) attachDeviceID(rawURL string) string {
	if c.deviceID == "" || !strings.Contains(rawURL, "fenbi.com") || strings.Contains(rawURL, "deviceId=") {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "deviceId=" + url.QueryEscape(c.deviceID)
}

// applyHeaders 设置 UA + 会话 Cookie 头（仅粉笔域）。
func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("User-Agent", UserAgent)
	if len(c.session) > 0 && strings.Contains(req.URL.Host, "fenbi.com") {
		parts := make([]string, 0, len(c.session))
		for _, ck := range c.session {
			if ck.Name == "" {
				continue
			}
			parts = append(parts, ck.Name+"="+ck.Value)
		}
		if len(parts) > 0 {
			req.Header.Set("Cookie", strings.Join(parts, "; "))
		}
	}
}

// GetJSON 发起 GET 并把响应解析到 out。
func (c *Client) GetJSON(rawURL string, out interface{}) (int, error) {
	req, err := http.NewRequest(http.MethodGet, c.attachDeviceID(rawURL), nil)
	if err != nil {
		return 0, err
	}
	c.applyHeaders(req)
	return c.doJSON(req, out)
}

// PostJSON 发起 POST（body 为 JSON 字符串），解析响应到 out。
func (c *Client) PostJSON(rawURL, jsonBody string, out interface{}) (int, error) {
	req, err := http.NewRequest(http.MethodPost, c.attachDeviceID(rawURL), strings.NewReader(jsonBody))
	if err != nil {
		return 0, err
	}
	c.applyHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req, out)
}

// PostForm 发起 POST（body 为 form 编码），解析响应到 out。
func (c *Client) PostForm(rawURL string, form url.Values, out interface{}) (int, error) {
	req, err := http.NewRequest(http.MethodPost, c.attachDeviceID(rawURL), strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	c.applyHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	return c.doJSON(req, out)
}

func (c *Client) doJSON(req *http.Request, out interface{}) (int, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}
