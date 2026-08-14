package fenbi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TimeRange 时间筛选：All 或 Days(-1 为自定义)
type TimeRange struct {
	All       bool      // true: timeRange=0 全部
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
	ID       int64  `json:"id"`
	Content  string `json:"content"`
	Material string `json:"material"`
	Type     int    `json:"type"`
	Difficulty int  `json:"difficulty"`
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

// QuestionIDsPage 分页查询题目 ID（type=1 错题；按知识点 + 时间筛选）。
type QuestionIDsPage struct {
	Total   int   `json:"total"`
	Results []int64 `json:"results"`
}

func (c *Client) QuestionIDsPage(questionType, categoryID int, offset, limit int, tr TimeRange) (*QuestionIDsPage, error) {
	u := fmt.Sprintf("https://tiku.fenbi.com/api/xingce/questionIds?type=%d&categoryId=%d&offset=%d&limit=%d&order=1&%s&%s",
		questionType, categoryID, offset, limit, tr.params(), apiParams)
	var page QuestionIDsPage
	if _, err := c.GetJSON(u, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// ExerciseHistoryItem 练习历史条目。
type ExerciseHistoryItem struct {
	ID          int64  `json:"id"`
	Status      int    `json:"status"`
	UpdatedTime int64  `json:"updatedTime"`
	Sheet       struct {
		Name string `json:"name"`
	} `json:"sheet"`
	AnswerCount int `json:"answerCount"`
	ElapsedTime int `json:"elapsedTime"`
	CorrectRate float64 `json:"correctRate"`
	Client      string `json:"client"`
}

// ExerciseReport 练习报告。
type ExerciseReport struct {
	ElapsedTime int `json:"elapsedTime"`
	AnswerCount int `json:"answerCount"`
	CorrectCount int `json:"correctCount"`
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

	// 逐条拉报告填充耗时/正确率
	var out []ExerciseHistoryItem
	for i := range all {
		item := all[i]
		if item.Status != 1 {
			continue
		}
		rep, err := c.ExerciseReport(item.ID)
		if err == nil && rep.AnswerCount > 0 {
			item.AnswerCount = rep.AnswerCount
			item.ElapsedTime = rep.ElapsedTime
			item.CorrectRate = float64(rep.CorrectCount) / float64(rep.AnswerCount) * 100
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
	Content     string `json:"content"`
	Solution    string `json:"solution"`
	Difficulty  int    `json:"difficulty"`
	CorrectAnswer struct {
		Choice string `json:"choice"`
	} `json:"correctAnswer"`
	Accessories []struct {
		Options []string `json:"options"`
	} `json:"accessories"`
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
