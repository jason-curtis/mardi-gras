package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// RigInfo describes a rig from the town's rigs.json registry.
type RigInfo struct {
	Name   string // e.g. "gastown", "mardigras"
	Prefix string // e.g. "gt", "ma"
}

// MultiRigConfig holds the configuration for cross-rig bead aggregation.
type MultiRigConfig struct {
	TownRoot string    // Absolute path to the Gas Town root directory
	Rigs     []RigInfo // Rigs to query (all or a subset)
}

// FindTownRoot walks up from dir looking for mayor/town.json.
func FindTownRoot(dir string) string {
	for {
		candidate := filepath.Join(dir, "mayor", "town.json")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadRigsFromTown reads mayor/rigs.json and returns all configured rigs.
func LoadRigsFromTown(townRoot string) ([]RigInfo, error) {
	path := filepath.Join(townRoot, "mayor", "rigs.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rigs.json: %w", err)
	}

	var registry struct {
		Rigs map[string]struct {
			Beads struct {
				Prefix string `json:"prefix"`
			} `json:"beads"`
		} `json:"rigs"`
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, fmt.Errorf("parse rigs.json: %w", err)
	}

	var rigs []RigInfo
	for name, info := range registry.Rigs {
		rigs = append(rigs, RigInfo{
			Name:   name,
			Prefix: info.Beads.Prefix,
		})
	}
	sort.Slice(rigs, func(i, j int) bool {
		return rigs[i].Name < rigs[j].Name
	})
	return rigs, nil
}

// FilterRigs returns only the rigs matching the given names.
// If names is empty or contains "all", returns all rigs.
func FilterRigs(all []RigInfo, names []string) []RigInfo {
	if len(names) == 0 {
		return all
	}
	for _, n := range names {
		if strings.EqualFold(n, "all") {
			return all
		}
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[strings.ToLower(n)] = true
	}
	var result []RigInfo
	for _, r := range all {
		if want[strings.ToLower(r.Name)] {
			result = append(result, r)
		}
	}
	return result
}

// FetchIssuesMultiRig queries bd list --json --rig <name> for each rig
// concurrently, tags each issue with its source rig, and merges results.
func FetchIssuesMultiRig(rigs []RigInfo) ([]Issue, error) {
	type result struct {
		rig    RigInfo
		issues []Issue
		err    error
	}

	results := make([]result, len(rigs))
	var wg sync.WaitGroup
	for i, rig := range rigs {
		wg.Add(1)
		go func(idx int, r RigInfo) {
			defer wg.Done()
			out, err := runWithTimeout(timeoutMedium, "bd", "list", "--json", "--limit", "0", "--all", "--rig", r.Name)
			if err != nil {
				results[idx] = result{rig: r, err: fmt.Errorf("bd list --rig %s: %w", r.Name, wrapExitError("bd list", err))}
				return
			}
			var issues []Issue
			if err := json.Unmarshal(out, &issues); err != nil {
				results[idx] = result{rig: r, err: fmt.Errorf("bd list --rig %s parse: %w", r.Name, err)}
				return
			}
			// Tag each issue with its source rig
			for j := range issues {
				issues[j].Rig = r.Name
			}
			results[idx] = result{rig: r, issues: issues}
		}(i, rig)
	}
	wg.Wait()

	// Collect all issues, skip rigs with errors (log them)
	var all []Issue
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err.Error())
			continue
		}
		all = append(all, r.issues...)
	}

	if len(all) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all rig queries failed: %s", strings.Join(errs, "; "))
	}

	SortIssues(all)
	return all, nil
}

// PollMultiRig polls bd list across multiple rigs on a timer.
func PollMultiRig(rigs []RigInfo) tea.Cmd {
	return tea.Tick(cliPollInterval, func(time.Time) tea.Msg {
		issues, err := FetchIssuesMultiRig(rigs)
		if err != nil {
			return FileWatchErrorMsg{Err: err}
		}
		return FileChangedMsg{Issues: issues, LastMod: time.Now()}
	})
}

// FetchIssuesMultiRigNow returns a tea.Cmd that fetches multi-rig issues immediately.
func FetchIssuesMultiRigNow(rigs []RigInfo) tea.Cmd {
	return func() tea.Msg {
		issues, err := FetchIssuesMultiRig(rigs)
		if err != nil {
			return FileWatchErrorMsg{Err: err}
		}
		return FileChangedMsg{Issues: issues, LastMod: time.Now()}
	}
}
