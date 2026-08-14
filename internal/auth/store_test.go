package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wlnyx/fenbi-helper-go/internal/fenbi"
)

func TestCredentialsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	dev := DeviceID{DeviceID: "v1_test", UserID: 123, Time: 1786000000000}
	cookies := []fenbi.Cookie{{Name: "sess", Value: "abc", Domain: "login.fenbi.com", Path: "/"}}
	if err := s.SaveCredentials(dev, cookies); err != nil {
		t.Fatal(err)
	}
	gotDev, gotCookies := s.LoadCredentials()
	if gotDev.DeviceID != "v1_test" || gotDev.UserID != 123 {
		t.Errorf("device 往返失败: %+v", gotDev)
	}
	if len(gotCookies) != 1 || gotCookies[0].Name != "sess" {
		t.Errorf("cookies 往返失败: %+v", gotCookies)
	}
	if !s.HasSession() {
		t.Error("HasSession 应为 true")
	}
	if err := s.ClearCredentials(); err != nil {
		t.Fatal(err)
	}
	if s.HasSession() {
		t.Error("ClearCredentials 后 HasSession 应为 false")
	}
	if _, err := os.Stat(filepath.Join(dir, "device.json")); !os.IsNotExist(err) {
		t.Error("device.json 应被删除")
	}
}

func TestCredentialFilePermission(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	dev := DeviceID{DeviceID: "v1_test", UserID: 1, Time: 1}
	if err := s.SaveCredentials(dev, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "device.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("device.json perm = %o, want 600", perm)
	}
}
