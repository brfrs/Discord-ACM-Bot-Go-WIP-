package leetcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const LEETCODE_GQL_URI = "https://leetcode.com/graphql"

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
