package fenbi

import (
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

// TimeRange 时间筛选：All 或 Days(-1 为自定义)
type TimeRange struct {
	All        bool      // true: timeRange=0 全部
	Start, End time.Time // 自定义起止
}

func (tr TimeRange) params() string {
	if tr.All {
		return "timeRange=0"
	}
	start := tr.Start.Format("20060102")
	end := tr.End.Format("20060102")
	return "timeRange=-1&startDate=" + start + "&endDate=" + end
}

// KeypointNode 知识点树节点。
type KeypointNode struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	QuestionIDs []int64        `json:"questionIds"`
	Children    []KeypointNode `json:"children"`
}

// Question 题目数据。
type Question struct {
	ID          int64  `json:"id"`
	Content     string `json:"content"`
	Material    string `json:"material"`
	Type        int    `json:"type"`
	Difficulty  int    `json:"difficulty"`
	Accessories []struct {
		Options []string `json:"options"`
		Type    int      `json:"type"`
	} `json:"accessories"`
	CorrectAnswer struct {
		Choice string `json:"choice"`
		Type   int    `json:"type"`
	} `json:"correctAnswer"`
}

// ErrorTreeNode 获取错题本知识点树。
func (c *Client) ErrorTreeNode(tr TimeRange) ([]KeypointNode, error) {
	return c.keypointTree("errors", tr)
}

// CollectTreeNode 获取收藏知识点树。
func (c *Client) CollectTreeNode(tr TimeRange) ([]KeypointNode, error) {
	return c.keypointTree("collects", tr)
}

func (c *Client) keypointTree(kind string, tr TimeRange) ([]KeypointNode, error) {
	u := fmt.Sprintf("https://tiku.fenbi.com/api/xingce/%s/keypoint-tree?%s&order=1&%s", kind, tr.params(), apiParams)
	var nodes []KeypointNode
	if _, err := c.GetJSON(u, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// Questions 批量获取题目。
func (c *Client) Questions(ids []int64) ([]Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	u := "https://tiku.fenbi.com/api/xingce/questions?ids=" + strings.Join(idStrs, ",") + "&" + apiParams
	var qs []Question
	if _, err := c.GetJSON(u, &qs); err != nil {
		return nil, err
	}
	return qs, nil
}

// ExerciseHistoryItem 练习历史条目。
type ExerciseHistoryItem struct {
	ID          int64 `json:"id"`
	Status      int   `json:"status"`
	UpdatedTime int64 `json:"updatedTime"`
	Sheet       struct {
		Name string `json:"name"`
	} `json:"sheet"`
	AnswerCount int     `json:"answerCount"`
	ElapsedTime int     `json:"elapsedTime"`
	CorrectRate float64 `json:"correctRate"`
	Client      string  `json:"client"`
}

// ExerciseReport 练习报告。
type ExerciseReport struct {
	ElapsedTime  int `json:"elapsedTime"`
	AnswerCount  int `json:"answerCount"`
	CorrectCount int `json:"correctCount"`
	Answers      []struct {
		QuestionID int64 `json:"questionId"`
		Correct    bool  `json:"correct"`
		Status     int   `json:"status"`
	} `json:"answers"`
}

// UserAnswer 单题作答。
type UserAnswer struct {
	QuestionID    int64 `json:"questionId"`
	QuestionIndex int   `json:"questionIndex"`
	Time          int   `json:"time"`
}

// ExerciseItem 练习详情中的题目级数据。
type ExerciseItem struct {
	Idx          int     `json:"idx"`
	QuestionID   int64   `json:"questionId"`
	Correct      bool    `json:"correct"`
	Cost         int     `json:"cost"`
	Status       int     `json:"status"`
	Difficulty   int     `json:"difficulty"`
	CorrectRatio float64 `json:"correctRatio"`
	Title        string  `json:"title"`
}

// ExerciseDetail 练习详情：每题对错/耗时 + 难度。
func (c *Client) ExerciseDetail(exerciseID int64) ([]ExerciseItem, error) {
	var ex struct {
		UserAnswers map[string]UserAnswer `json:"userAnswers"`
	}
	u := fmt.Sprintf("https://tiku.fenbi.com/api/xingce/exercises/%d?%s", exerciseID, apiParams)
	if _, err := c.GetJSON(u, &ex); err != nil {
		return nil, err
	}
	rep, err := c.ExerciseReport(exerciseID)
	if err != nil {
		return nil, err
	}

	type doneItem struct {
		ua         UserAnswer
		correct    bool
		status     int
		questionID int64
	}
	var done []doneItem
	for _, a := range rep.Answers {
		if a.Status == 10 {
			continue
		}
		ua := UserAnswer{QuestionID: a.QuestionID, QuestionIndex: -1}
		for _, v := range ex.UserAnswers {
			if v.QuestionID == a.QuestionID {
				ua = v
				break
			}
		}
		done = append(done, doneItem{ua: ua, correct: a.Correct, status: a.Status, questionID: a.QuestionID})
	}
	if len(done) == 0 {
		return nil, nil
	}

	// 批量补难度/标题
	ids := make([]int64, 0, len(done))
	for _, d := range done {
		ids = append(ids, d.questionID)
	}
	sols, err := c.Solutions(ids)
	if err != nil {
		return nil, err
	}

	var items []ExerciseItem
	for i, d := range done {
		var sol Solution
		if i < len(sols) {
			sol = sols[i]
		}
		idx := d.ua.QuestionIndex + 1
		if idx <= 0 {
			idx = i + 1
		}
		items = append(items, ExerciseItem{
			Idx:          idx,
			QuestionID:   d.questionID,
			Correct:      d.correct,
			Cost:         d.ua.Time,
			Status:       d.status,
			Difficulty:   sol.Difficulty,
			CorrectRatio: sol.QuestionMeta.CorrectRatio,
			Title:        stripHTMLContent(sol.Content),
		})
	}
	return items, nil
}

func stripHTMLContent(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 60 {
		out = out[:60] + "…"
	}
	return out
}

// History 获取练习历史（categoryId=2 行测模块，两个游标合并）。
func (c *Client) History() ([]ExerciseHistoryItem, error) {
	var all []ExerciseHistoryItem
	for _, cursor := range []int{0, 30} {
		u := fmt.Sprintf("https://tiku.fenbi.com/api/xingce/category-exercises?categoryId=2&cursor=%d&count=30&%s", cursor, apiParams)
		var resp struct {
			Datas []ExerciseHistoryItem `json:"datas"`
		}
		if _, err := c.GetJSON(u, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Datas...)
	}

	// 并发拉报告填充耗时/正确率
	type reportResult struct {
		id  int64
		rep *ExerciseReport
		err error
	}
	var out []ExerciseHistoryItem
	var pending []int64
	valid := map[int64]ExerciseHistoryItem{}
	for _, item := range all {
		if item.Status != 1 {
			continue
		}
		valid[item.ID] = item
		pending = append(pending, item.ID)
	}
	results := make(chan reportResult, len(pending))
	var wg sync.WaitGroup
	for _, id := range pending {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			rep, err := c.ExerciseReport(id)
			results <- reportResult{id: id, rep: rep, err: err}
		}(id)
	}
	wg.Wait()
	close(results)
	for r := range results {
		item := valid[r.id]
		if r.err == nil && r.rep != nil && r.rep.AnswerCount > 0 {
			item.AnswerCount = r.rep.AnswerCount
			item.ElapsedTime = r.rep.ElapsedTime
			item.CorrectRate = float64(r.rep.CorrectCount) / float64(r.rep.AnswerCount) * 100
		}
		if item.AnswerCount > 0 {
			out = append(out, item)
		}
	}
	return out, nil
}

// ExerciseReport 单次练习报告。
func (c *Client) ExerciseReport(exerciseID int64) (*ExerciseReport, error) {
	u := fmt.Sprintf("https://tiku.fenbi.com/api/xingce/exercises/%d/report/v2?%s", exerciseID, apiParams)
	var rep ExerciseReport
	if _, err := c.GetJSON(u, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

// Solution 解析。
type Solution struct {
	Content       string `json:"content"`
	Solution      string `json:"solution"`
	Difficulty    int    `json:"difficulty"`
	CorrectAnswer struct {
		Choice string `json:"choice"`
	} `json:"correctAnswer"`
	Accessories []struct {
		Options []string `json:"options"`
	} `json:"accessories"`
	QuestionMeta struct {
		CorrectRatio float64 `json:"correctRatio"`
	} `json:"questionMeta"`
}

// Solutions 批量获取解析。
func (c *Client) Solutions(ids []int64) ([]Solution, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	u := "https://tiku.fenbi.com/api/xingce/solutions?ids=" + strings.Join(idStrs, ",") + "&" + apiParams
	var sols []Solution
	if _, err := c.GetJSON(u, &sols); err != nil {
		return nil, err
	}
	return sols, nil
}

// SaveNote 同步笔记到粉笔。
func (c *Client) SaveNote(questionID int64, content string) error {
	form := url.Values{}
	form.Set("questionId", strconv.FormatInt(questionID, 10))
	form.Set("content", content)
	u := "https://tiku.fenbi.com/api/xingce/notes?" + apiParams
	req, err := http.NewRequest(http.MethodPost, c.attachDeviceID(u), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	c.applyHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.Code != 1 {
		return fmt.Errorf("笔记同步失败: %s", string(body))
	}
	return nil
}
