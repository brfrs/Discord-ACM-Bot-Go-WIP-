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

type submissionEntry struct {
	Status string `json:"statusDisplay"`
	Title  string `json:"title"`
	Slug   string `json:"titleSlug"`
	Time   string `json:"timestamp"`
}

type leetcodeResponseData struct {
	EntryList []submissionEntry `json:"recentSubmissionList"`
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

func FindIfUserCompletedLeetCodeProblem(user string, problem string) (bool, error) {
	graphqlVars := map[string]string{
		"username": user,
	}
	data := map[string]interface{}{
		"operationName": "getRecentSubmissionList",
		"query":         USER_SUBMISSION_GQL_QUERY,
		"variables":     graphqlVars,
	}
	postBody, _ := json.Marshal(data)
	resp, err := http.Post(LEETCODE_GQL_URI, "application/json", bytes.NewBuffer(postBody))

	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("LeetCode Error: Response to GraphQL request %d!=200", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)

	var decodedData leetcodeResponseBody
	err = decoder.Decode(&decodedData)

	if err != nil {
		return false, err
	}

	for _, entry := range decodedData.Data.EntryList {
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
