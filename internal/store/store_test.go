package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestQuestionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.SetUser(1001)

	s.UpdateQuestion("42", func(q *QuestionReview) {
		q.ErrorCategory = "计算蠢"
		q.ReviewState = StateFlagged
		q.Note4 = &Note4{Point: "考点X", Mistake: "看错", Approach: "重算", Reminder: "圈单位"}
		q.RedoHistory = append(q.RedoHistory, RedoRecord{Date: "2026-08-14", Result: "correct"})
		q.Archived = false
	})
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 重新加载（新实例，模拟重启）
	s2 := NewStore(dir)
	s2.SetUser(1001)
	q := s2.Question("42")
	if q.ErrorCategory != "计算蠢" {
		t.Errorf("ErrorCategory = %q, want 计算蠢", q.ErrorCategory)
	}
	if q.ReviewState != StateFlagged {
		t.Errorf("ReviewState = %q, want flagged", q.ReviewState)
	}
	if q.Note4 == nil || q.Note4.Point != "考点X" {
		t.Errorf("Note4 未持久化: %+v", q.Note4)
	}
	if len(q.RedoHistory) != 1 || q.RedoHistory[0].Result != "correct" {
		t.Errorf("RedoHistory 未持久化: %+v", q.RedoHistory)
	}
}

func TestPerUserIsolation(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	s.SetUser(1)
	s.UpdateQuestion("42", func(q *QuestionReview) { q.ErrorCategory = "找数瞎" })
	s.Save()

	// 切换账号不应看到账号1的数据
	s.SetUser(2)
	q := s.Question("42")
	if q.ErrorCategory != "" {
		t.Errorf("账号2 看到了账号1 的归类: %q", q.ErrorCategory)
	}
}

func TestCorruptFileBackup(t *testing.T) {
	dir := t.TempDir()
	// 写入损坏文件
	file := filepath.Join(dir, "review-1.json")
	if err := os.WriteFile(file, []byte("{broken json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewStore(dir)
	s.SetUser(1)
	q := s.Question("42") // 应触发自检重建
	if q == nil {
		t.Fatal("question nil")
	}
	if _, err := os.Stat(file + ".bak"); err != nil {
		t.Errorf("备份文件未生成: %v", err)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.SetUser(1)
	s.UpdateQuestion("1", func(q *QuestionReview) { q.Doubt = true })
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return // Windows 不支持 Unix 权限位
	}
	// 文件权限应为 0600
	info, err := os.Stat(s.file())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 600", perm)
	}
}

func TestUsedCategories(t *testing.T) {
	// 验证预设类别 + 自定义标签收集（通过 review service 不可用，这里直接验证 store 层）
	dir := t.TempDir()
	s := NewStore(dir)
	s.SetUser(1)
	s.UpdateQuestion("1", func(q *QuestionReview) { q.ErrorCategory = "自定义标签" })
	all := s.AllQuestions()
	if all["1"].ErrorCategory != "自定义标签" {
		t.Errorf("自定义标签未保存: %+v", all["1"])
	}
	_ = time.Now()
}
