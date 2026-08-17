package fenbi

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// SessionJar 包装 http.CookieJar，支持 dump 会话 Cookie 用于持久化。
type SessionJar struct {
	jar *cookiejar.Jar
	hc  *http.Client
}

func NewSessionJar() *SessionJar {
	jar, _ := cookiejar.New(nil)
	return &SessionJar{jar: jar, hc: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
}

func (s *SessionJar) Client() *http.Client {
	return s.hc
}

// DumpCookies 收集主要粉笔域上的 Cookie。
func (s *SessionJar) DumpCookies() []Cookie {
	domains := []string{
		"https://login.fenbi.com/",
		"https://www.fenbi.com/",
		"https://ke.fenbi.com/",
		"https://tiku.fenbi.com/",
		"https://fenbi.com/",
	}
	seen := map[string]bool{}
	var out []Cookie
	for _, d := range domains {
		u, err := url.Parse(d)
		if err != nil {
			continue
		}
		for _, c := range s.jar.Cookies(u) {
			key := c.Name + "@" + c.Domain
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Cookie{Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path})
		}
	}
	return out
}

// LoadCookies 把持久化的 Cookie 注入 jar。
func (s *SessionJar) LoadCookies(cookies []Cookie) {
	for _, c := range cookies {
		domain := c.Domain
		if domain == "" {
			domain = "login.fenbi.com"
		}
		u := &url.URL{Scheme: "https", Host: domain}
		hc := &http.Cookie{Name: c.Name, Value: c.Value, Path: c.Path}
		if hc.Path == "" {
			hc.Path = "/"
		}
		s.jar.SetCookies(u, []*http.Cookie{hc})
	}
}
