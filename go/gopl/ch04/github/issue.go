package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// authToken reads the GitHub personal access token from the environment.
// Write operations require it; reads work anonymously but rate-limit sooner without it.
func authToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// doRequest sends a JSON request to the GitHub API and decodes the JSON
// response into out. body and out may be nil.
func doRequest(method, url string, body, out any) error {
	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = new(bytes.Buffer)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := authToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if method != http.MethodGet {
		return fmt.Errorf("GITHUB_TOKEN environment variable must be set for %s requests", method)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github request failed: %s", resp.Status)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// CreateIssue creates a new issue in owner/repo.
// See https://developer.github.com/v3/issues/#create-an-issue.
func CreateIssue(owner, repo string, req *IssueRequest) (*Issue, error) {
	u := fmt.Sprintf(IssuesURLFmt, owner, repo)
	var issue Issue
	if err := doRequest(http.MethodPost, u, req, &issue); err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}
	return &issue, nil
}

// GetIssue reads a single issue from owner/repo.
// See https://developer.github.com/v3/issues/#get-an-issue.
func GetIssue(owner, repo string, number int) (*Issue, error) {
	u := fmt.Sprintf(IssueURLFmt, owner, repo, number)
	var issue Issue
	if err := doRequest(http.MethodGet, u, nil, &issue); err != nil {
		return nil, fmt.Errorf("getting issue #%d: %w", number, err)
	}
	return &issue, nil
}

// UpdateIssue updates title, body, state, labels, or assignees of an issue.
// See https://developer.github.com/v3/issues/#update-an-issue.
func UpdateIssue(owner, repo string, number int, req *IssueRequest) (*Issue, error) {
	u := fmt.Sprintf(IssueURLFmt, owner, repo, number)
	var issue Issue
	if err := doRequest(http.MethodPatch, u, req, &issue); err != nil {
		return nil, fmt.Errorf("updating issue #%d: %w", number, err)
	}
	return &issue, nil
}

// CloseIssue closes an issue. Closing is just an update of its state.
func CloseIssue(owner, repo string, number int) (*Issue, error) {
	issue, err := UpdateIssue(owner, repo, number, &IssueRequest{State: "closed"})
	if err != nil {
		return nil, fmt.Errorf("closing issue #%d: %w", number, err)
	}
	return issue, nil
}

// AddComment posts a comment on an issue.
// See https://developer.github.com/v3/issues/comments/#create-a-comment.
func AddComment(owner, repo string, number int, body string) (*Comment, error) {
	u := fmt.Sprintf(CommentsURLFmt, owner, repo, number)
	payload := struct {
		Body string `json:"body"`
	}{Body: body}

	var comment Comment
	if err := doRequest(http.MethodPost, u, payload, &comment); err != nil {
		return nil, fmt.Errorf("commenting on issue #%d: %w", number, err)
	}
	return &comment, nil
}
