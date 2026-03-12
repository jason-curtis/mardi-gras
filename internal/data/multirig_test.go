package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFindTownRoot(t *testing.T) {
	// Create temp directory structure: tmpdir/mayor/town.json
	tmp := t.TempDir()
	mayorDir := filepath.Join(tmp, "mayor")
	if err := os.MkdirAll(mayorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mayorDir, "town.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// From the town root itself
	if got := FindTownRoot(tmp); got != tmp {
		t.Errorf("FindTownRoot(%q) = %q, want %q", tmp, got, tmp)
	}

	// From a subdirectory
	sub := filepath.Join(tmp, "rigs", "myrig")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindTownRoot(sub); got != tmp {
		t.Errorf("FindTownRoot(%q) = %q, want %q", sub, got, tmp)
	}

	// From a directory with no town root
	noTown := t.TempDir()
	if got := FindTownRoot(noTown); got != "" {
		t.Errorf("FindTownRoot(%q) = %q, want empty", noTown, got)
	}
}

func TestLoadRigsFromTown(t *testing.T) {
	tmp := t.TempDir()
	mayorDir := filepath.Join(tmp, "mayor")
	if err := os.MkdirAll(mayorDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rigsJSON := map[string]interface{}{
		"version": 1,
		"rigs": map[string]interface{}{
			"gastown": map[string]interface{}{
				"beads": map[string]interface{}{"prefix": "gt"},
			},
			"mardigras": map[string]interface{}{
				"beads": map[string]interface{}{"prefix": "ma"},
			},
			"beads": map[string]interface{}{
				"beads": map[string]interface{}{"prefix": "bd"},
			},
		},
	}
	raw, err := json.Marshal(rigsJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mayorDir, "rigs.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	rigs, err := LoadRigsFromTown(tmp)
	if err != nil {
		t.Fatalf("LoadRigsFromTown: %v", err)
	}
	if len(rigs) != 3 {
		t.Fatalf("expected 3 rigs, got %d", len(rigs))
	}
	// Should be sorted by name
	if rigs[0].Name != "beads" || rigs[0].Prefix != "bd" {
		t.Errorf("rigs[0] = %+v, want beads/bd", rigs[0])
	}
	if rigs[1].Name != "gastown" || rigs[1].Prefix != "gt" {
		t.Errorf("rigs[1] = %+v, want gastown/gt", rigs[1])
	}
	if rigs[2].Name != "mardigras" || rigs[2].Prefix != "ma" {
		t.Errorf("rigs[2] = %+v, want mardigras/ma", rigs[2])
	}
}

func TestFilterRigs(t *testing.T) {
	all := []RigInfo{
		{Name: "beads", Prefix: "bd"},
		{Name: "gastown", Prefix: "gt"},
		{Name: "mardigras", Prefix: "ma"},
	}

	// Empty names returns all
	result := FilterRigs(all, nil)
	if len(result) != 3 {
		t.Errorf("FilterRigs(nil) returned %d rigs, want 3", len(result))
	}

	// "all" returns all
	result = FilterRigs(all, []string{"all"})
	if len(result) != 3 {
		t.Errorf("FilterRigs([all]) returned %d rigs, want 3", len(result))
	}

	// Specific rigs
	result = FilterRigs(all, []string{"gastown", "beads"})
	if len(result) != 2 {
		t.Fatalf("FilterRigs([gastown,beads]) returned %d rigs, want 2", len(result))
	}

	// Case-insensitive
	result = FilterRigs(all, []string{"GasTown"})
	if len(result) != 1 || result[0].Name != "gastown" {
		t.Errorf("FilterRigs([GasTown]) = %v, want [gastown]", result)
	}

	// No match
	result = FilterRigs(all, []string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("FilterRigs([nonexistent]) returned %d rigs, want 0", len(result))
	}
}

func TestIssueRigField(t *testing.T) {
	issue := Issue{
		ID:    "gt-001",
		Title: "Test issue",
		Rig:   "gastown",
	}

	// Verify Rig field roundtrips through JSON
	raw, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Issue
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Rig != "gastown" {
		t.Errorf("decoded.Rig = %q, want %q", decoded.Rig, "gastown")
	}
}
