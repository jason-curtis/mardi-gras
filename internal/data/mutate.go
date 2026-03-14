package data

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SetStatus runs `bd update <id> --status=<status>` to change an issue's status.
func SetStatus(issueID string, status Status) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, "--status="+string(status))
}

// ClaimIssue runs `bd update <id> --claim` to atomically set assignee and status to in_progress.
// Fails if the issue is already claimed by another agent, preventing races in multi-agent workflows.
func ClaimIssue(issueID string) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, "--claim")
}

// CloseIssue runs `bd close <id>` to close an issue.
func CloseIssue(issueID string) error {
	return execWithTimeout(timeoutShort, "bd", "close", issueID)
}

// SetPriority runs `bd update <id> --priority=<n>` to change priority.
func SetPriority(issueID string, priority Priority) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, fmt.Sprintf("--priority=%d", priority))
}

// CreateIssue runs `bd create` with the given parameters and returns the new issue ID.
func CreateIssue(title string, issueType IssueType, priority Priority) (string, error) {
	args := []string{
		"create",
		"--title=" + title,
		"--type=" + string(issueType),
		fmt.Sprintf("--priority=%d", priority),
	}
	out, err := runWithTimeout(timeoutShort, "bd", args...)
	if err != nil {
		return "", wrapExitError("bd create", err)
	}
	// bd create prints the new issue ID
	return strings.TrimSpace(string(out)), nil
}

// UpdateTextField updates a text field on an issue.
// The field parameter should be one of: title, description, notes, design, acceptance.
// For description, content is piped via stdin to handle large/multiline text safely.
// Other fields are passed as command-line args (safe because exec.Command bypasses the shell).
func UpdateTextField(issueID, field, value string) error {
	if field == "description" {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bd", "update", issueID, "--stdin")
		cmd.Stdin = bytes.NewReader([]byte(value))
		return cmd.Run()
	}
	return execWithTimeout(timeoutShort, "bd", "update", issueID, "--"+field+"="+value)
}

// BranchName generates a git branch name from an issue.
func BranchName(issue Issue) string {
	prefix := "feat"
	switch issue.IssueType {
	case TypeBug:
		prefix = "fix"
	case TypeChore:
		prefix = "chore"
	case TypeTask:
		prefix = "task"
	}
	slug := slugify(issue.Title)
	return fmt.Sprintf("%s/%s-%s", prefix, issue.ID, slug)
}

// slugify converts a title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ', r == '-', r == '_', r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := b.String()
	result = strings.TrimRight(result, "-")
	if len(result) > 50 {
		result = result[:50]
		result = strings.TrimRight(result, "-")
	}
	return result
}
