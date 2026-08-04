package github

// Package github provides a Go API for the GitHub issue tracker.
// See https://developer.github.com/v3/issues/.
const (
	// SearchIssuesURL is the issue search endpoint.
	// See https://developer.github.com/v3/search/#search-issues.
	SearchIssuesURL = "https://api.github.com/search/issues"

	// IssuesURLFmt lists or creates issues for a repo.
	// Use fmt.Sprintf(IssuesURLFmt, owner, repo).
	IssuesURLFmt = "https://api.github.com/repos/%s/%s/issues"

	// IssueURLFmt reads, updates, or closes a single issue.
	// Use fmt.Sprintf(IssueURLFmt, owner, repo, number).
	IssueURLFmt = "https://api.github.com/repos/%s/%s/issues/%d"

	// CommentsURLFmt adds a comment to an issue.
	// Use fmt.Sprintf(CommentsURLFmt, owner, repo, number).
	CommentsURLFmt = "https://api.github.com/repos/%s/%s/issues/%d/comments"
)
