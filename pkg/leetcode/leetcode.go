package leetcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const LEETCODE_GQL_URI = "https://leetcode.com/graphql"
const LEETCODE_PROBLEMS_URI = "https://leetcode.com/api/problems/all/"

const USER_SUBMISSION_GQL_QUERY = `query getRecentSubmissionList($username: String!, $limit: Int) {
  recentSubmissionList(username: $username, limit: $limit) {
    title
    titleSlug
    timestamp
    statusDisplay
    lang
    __typename
  }
  languageList {
    id
    name
    verboseName
    __typename
  }
}
`

const PROBLEM_DESCRIPTION_GQL_QUERY = `query questionData($titleSlug: String!) {
	question(titleSlug: $titleSlug) {
	  title
	  titleSlug
	  content
	}
  }
`

const LEETCODE_PROBLEM_URI_FORMAT = "https://leetcode.com/problems/%s/"

type submissionEntry struct {
	Status string `json:"statusDisplay"`
	Title  string `json:"title"`
	Slug   string `json:"titleSlug"`
	Time   string `json:"timestamp"`
}

type ProblemDesc struct {
	Title   string `json:"title"`
	Slug    string `json:"titleSlug"`
	Content string `json:"content"`
}

type leetcodeResponseData struct {
	EntryList    []submissionEntry `json:"recentSubmissionList,omitempty"`
	QuestionData *ProblemDesc      `json:"question,omitempty"`
}

type leetcodeResponseBody struct {
	Data leetcodeResponseData `json:"data"`
}

type ProblemMetadata struct {
	Title            string `json:"question__title"`
	Slug             string `json:"question__title_slug"`
	TotalAccepts     int    `json:"total_acs"`
	TotalSubmissions int    `json:"total_submitted"`
}

const (
	DIFFICULTY_EASY   = 1
	DIFFICULTY_MEDIUM = 2
	DIFFICULTY_HARD   = 3
)

type ProblemDifficulty struct {
	Level int `json:"level"`
}

type Problem struct {
	Stat       ProblemMetadata   `json:"stat"`
	Difficulty ProblemDifficulty `json:"difficulty"`
	PaidOnly   bool              `json:"paid_only"`
}

type apiResponse struct {
	StatStatusPairs []Problem `json:"stat_status_pairs"`
}

func performGQLQuery(opName, query string, vars map[string]interface{}) ([]byte, error) {
	toSend := map[string]interface{}{
		"operationName": opName,
		"query":         query,
		"variables":     vars,
	}

	postBody, err := json.Marshal(toSend)

	if err != nil {
		return nil, err
	}

	resp, err := http.Post(LEETCODE_GQL_URI, "application/json", bytes.NewBuffer(postBody))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LeetCode Error: Response to GraphQL request %d!=200", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func FindIfUserCompletedLeetCodeProblem(user string, problem string) (bool, error) {
	graphqlVars := map[string]interface{}{
		"username": user,
	}

	body, err := performGQLQuery("getRecentSubmissionList", USER_SUBMISSION_GQL_QUERY, graphqlVars)

	if err != nil {
		return false, err
	}

	var decodedBody leetcodeResponseBody
	json.Unmarshal(body, &decodedBody)

	for _, entry := range decodedBody.Data.EntryList {
		if entry.Slug == problem && entry.Status == "Accepted" {
			return true, nil
		}
	}

	return false, nil
}

func GetLeetCodeProblems() ([]Problem, error) {
	resp, err := http.Get(LEETCODE_PROBLEMS_URI)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("leetcode error: response code for %s is %d", LEETCODE_PROBLEMS_URI, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var blob apiResponse // LeetCode's big blob of problem data
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return nil, err
	}

	return blob.StatStatusPairs, nil
}

func GetProblemDesc(slug string) (ProblemDesc, error) {
	vars := map[string]interface{}{
		"titleSlug": slug,
	}

	body, err := performGQLQuery("questionData", PROBLEM_DESCRIPTION_GQL_QUERY, vars)

	if err != nil {
		return ProblemDesc{}, err
	}

	var decodedBody leetcodeResponseBody
	err = json.Unmarshal(body, &decodedBody)

	if err != nil {
		return ProblemDesc{}, err
	}

	if decodedBody.Data.QuestionData == nil {
		return ProblemDesc{}, fmt.Errorf("no question data")
	}

	return *decodedBody.Data.QuestionData, nil
}

func GetProblemURL(slug string) string {
	return fmt.Sprintf(LEETCODE_PROBLEM_URI_FORMAT, slug)
}
