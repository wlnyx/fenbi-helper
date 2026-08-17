package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 错误归类（六步法第三步）
var ErrorCategories = []string{"找数瞎", "列式乱", "计算蠢", "掉坑快", "死磕傻", "读不懂"}

// 复习状态（六步法第六步）
const (
	StatePending  = "pending"  // 待复习
	StateMastered = "mastered" // 已掌握
	StateFlagged  = "flagged"  // 重点
)

// Note4 四句话记录（六步法第四步）。
type Note4 struct {
	Point    string `json:"point"`    // ① 考点是什么
	Mistake  string `json:"mistake"`  // ② 错在哪里
	Approach string `json:"approach"` // ③ 正确计算思路
	Reminder string `json:"reminder"` // ④ 长效提醒/举一反三
}

// RedoRecord 重做记录（六步法第二步）。
type RedoRecord struct {
	Date   string `json:"date"`
	Result string `json:"result"` // correct / wrong
}

// QuestionReview 单题的复盘数据。
type QuestionReview struct {
	Doubt         bool         `json:"doubt"`                   // ① 存疑标记
	ErrorCategory string       `json:"errorCategory,omitempty"` // ③
	RedoHistory   []RedoRecord `json:"redoHistory,omitempty"`   // ②
	ReviewState   string       `json:"reviewState,omitempty"`   // ⑥ pending/mastered/flagged
	Note4         *Note4       `json:"note4,omitempty"`         // ④
	Archived      bool         `json:"archived,omitempty"`      // 归档
	UpdatedAt     int64        `json:"updatedAt"`
}

// MacroSummary 套卷宏观总结（六步法第五步）。
type MacroSummary struct {
	TopErrorType    string `json:"topErrorType,omitempty"`    // 占比最高的错误类型
	SlowestMaterial string `json:"slowestMaterial,omitempty"` // 耗时最长材料及原因
	Trap            string `json:"trap,omitempty"`            // 本次最容易踩的陷阱
	Rules           string `json:"rules,omitempty"`           // 可执行做题规范
}

// ExerciseReview 一次练习的复盘数据。
type ExerciseReview struct {
	Macro *MacroSummary `json:"macro,omitempty"`
}

// Data 是单个账号的完整复盘存储。
type Data struct {
	Questions map[string]*QuestionReview `json:"questions"`
	Exercises map[string]*ExerciseReview `json:"exercises"`
}

// Store 按 userId 隔离的本地复盘存储。
type Store struct {
	mu     sync.Mutex
	dir    string
	userID int64
	cache  *Data
	dirty  bool
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// SetUser 切换账号（切换后重新加载对应文件）。
func (s *Store) SetUser(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.userID == userID && s.cache != nil {
		return
	}
	s.userID = userID
	s.cache = nil
	s.loadLocked()
}

func (s *Store) file() string {
	return filepath.Join(s.dir, fmt.Sprintf("review-%d.json", s.userID))
}

func (s *Store) loadLocked() {
	if s.cache != nil {
		return
	}
	s.cache = &Data{Questions: map[string]*QuestionReview{}, Exercises: map[string]*ExerciseReview{}}
	b, err := os.ReadFile(s.file())
	if err != nil {
		return
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		// 损坏自检：备份后重建
		if rerr := os.Rename(s.file(), s.file()+".bak"); rerr != nil {
			log.Printf("复盘数据损坏且备份失败: %v (原始错误: %v)", rerr, err)
		} else {
			log.Printf("复盘数据损坏，已备份为 .bak 并重建: %v", err)
		}
		return
	}
	if d.Questions == nil {
		d.Questions = map[string]*QuestionReview{}
	}
	if d.Exercises == nil {
		d.Exercises = map[string]*ExerciseReview{}
	}
	s.cache = &d
}

// flush 原子写入。
func (s *Store) flush() error {
	if !s.dirty {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.file()); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

// Save 手动触发保存（通常在请求结束时调用）。
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flush()
}

func (s *Store) mark() {
	s.dirty = true
}

// Question 获取单题复盘数据（无则初始化）。返回拷贝，避免锁外数据竞争。
func (s *Store) Question(qid string) *QuestionReview {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	q, ok := s.cache.Questions[qid]
	if !ok {
		q = &QuestionReview{ReviewState: StatePending}
		s.cache.Questions[qid] = q
	}
	cp := *q
	return &cp
}

// UpdateQuestion 更新单题复盘字段。
func (s *Store) UpdateQuestion(qid string, fn func(q *QuestionReview)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	q, ok := s.cache.Questions[qid]
	if !ok {
		q = &QuestionReview{ReviewState: StatePending}
		s.cache.Questions[qid] = q
	}
	fn(q)
	q.UpdatedAt = time.Now().UnixMilli()
	s.mark()
}

// Exercise 获取练习复盘数据。
func (s *Store) Exercise(eid string) *ExerciseReview {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	e, ok := s.cache.Exercises[eid]
	if !ok {
		e = &ExerciseReview{}
		s.cache.Exercises[eid] = e
	}
	return e
}

// UpdateExercise 更新练习复盘。
func (s *Store) UpdateExercise(eid string, fn func(e *ExerciseReview)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	e, ok := s.cache.Exercises[eid]
	if !ok {
		e = &ExerciseReview{}
		s.cache.Exercises[eid] = e
	}
	fn(e)
	s.mark()
}

// AllQuestions 返回全部题目复盘数据（快照）。
func (s *Store) AllQuestions() map[string]*QuestionReview {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make(map[string]*QuestionReview, len(s.cache.Questions))
	for k, v := range s.cache.Questions {
		out[k] = v
	}
	return out
}
