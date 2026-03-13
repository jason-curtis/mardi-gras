package app

import (
	"testing"

	"github.com/matt-wright86/mardi-gras/internal/data"
)

// ---------------------------------------------------------------------------
// plural
// ---------------------------------------------------------------------------

func TestPlural(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "s"},
		{1, ""},
		{5, "s"},
	}
	for _, tt := range tests {
		got := plural(tt.n)
		if got != tt.want {
			t.Errorf("plural(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// diffIssues
// ---------------------------------------------------------------------------

func TestDiffIssuesEmptyPrev(t *testing.T) {
	m := Model{
		prevIssueMap: map[string]data.Status{},
		changedIDs:   make(map[string]bool),
	}
	issues := []data.Issue{testIssue("a", data.StatusOpen)}
	if got := m.diffIssues(issues); got != 0 {
		t.Errorf("empty prevIssueMap: got %d changes, want 0", got)
	}
}

func TestDiffIssuesStatusChanged(t *testing.T) {
	m := Model{
		prevIssueMap: map[string]data.Status{
			"a": data.StatusOpen,
		},
		changedIDs: make(map[string]bool),
	}
	issues := []data.Issue{testIssue("a", data.StatusInProgress)}
	got := m.diffIssues(issues)
	if got != 1 {
		t.Errorf("status changed: got %d changes, want 1", got)
	}
	if !m.changedIDs["a"] {
		t.Error("expected changedIDs to contain 'a'")
	}
}

func TestDiffIssuesNewAndRemoved(t *testing.T) {
	m := Model{
		prevIssueMap: map[string]data.Status{
			"old": data.StatusOpen,
		},
		changedIDs: make(map[string]bool),
	}
	issues := []data.Issue{testIssue("new", data.StatusOpen)}
	got := m.diffIssues(issues)
	// 1 new issue + 1 removed issue = 2
	if got != 2 {
		t.Errorf("new+removed: got %d changes, want 2", got)
	}
	if !m.changedIDs["new"] {
		t.Error("expected changedIDs to contain 'new'")
	}
}

func TestDiffIssuesNoChange(t *testing.T) {
	m := Model{
		prevIssueMap: map[string]data.Status{
			"a": data.StatusOpen,
			"b": data.StatusClosed,
		},
		changedIDs: make(map[string]bool),
	}
	issues := []data.Issue{
		testIssue("a", data.StatusOpen),
		testIssue("b", data.StatusClosed),
	}
	if got := m.diffIssues(issues); got != 0 {
		t.Errorf("no change: got %d changes, want 0", got)
	}
}
