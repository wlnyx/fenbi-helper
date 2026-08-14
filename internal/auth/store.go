package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
)

// DeviceID 持久化的 deviceId 凭据。
type DeviceID struct {
	DeviceID string `json:"deviceId"`
	UserID   int64  `json:"userId"`
	Time     int64  `json:"time"`
}

// Store 负责凭据与本地文件的读写。
type Store struct {
	mu    sync.Mutex
	dir   string
	creds *DeviceID
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Dir() string { return s.dir }

// SaveCredentials 持久化 deviceId + 会话 Cookie。
func (s *Store) SaveCredentials(deviceID DeviceID, cookies []fenbi.Cookie) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	// deviceId
	devBytes, _ := json.Marshal(deviceID)
	if err := atomicWrite(filepath.Join(s.dir, "device.json"), devBytes); err != nil {
		return err
	}
	// cookies
	cBytes, _ := json.Marshal(cookies)
	return atomicWrite(filepath.Join(s.dir, "cookies.json"), cBytes)
}

// LoadCredentials 读取持久化凭据（deviceId + cookie）。
func (s *Store) LoadCredentials() (*DeviceID, []fenbi.Cookie) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var dev DeviceID
	devBytes, err := os.ReadFile(filepath.Join(s.dir, "device.json"))
	if err == nil {
		json.Unmarshal(devBytes, &dev)
	}
	var cookies []fenbi.Cookie
	cBytes, err := os.ReadFile(filepath.Join(s.dir, "cookies.json"))
	if err == nil {
		json.Unmarshal(cBytes, &cookies)
	}
	return &dev, cookies
}

// ClearCredentials 删除本地凭据（退出登录）。
func (s *Store) ClearCredentials() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err1 := os.Remove(filepath.Join(s.dir, "device.json"))
	err2 := os.Remove(filepath.Join(s.dir, "cookies.json"))
	if err1 != nil && !os.IsNotExist(err1) {
		return err1
	}
	if err2 != nil && !os.IsNotExist(err2) {
		return err2
	}
	return nil
}

// HasSession 是否已有持久化登录凭据。
func (s *Store) HasSession() bool {
	dev, _ := s.LoadCredentials()
	return dev != nil && dev.DeviceID != ""
}

// SaveQrPNG 保存二维码图片（供 /api/qr.png 读取）。
func (s *Store) SaveQrPNG(png []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(s.dir, "qr.png"), png)
}

// LoadQrPNG 读取二维码图片。
func (s *Store) LoadQrPNG() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.ReadFile(filepath.Join(s.dir, "qr.png"))
}

// atomicWrite 原子写入：tmp + rename。
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
