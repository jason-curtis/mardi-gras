package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/matt-wright86/mardi-gras/internal/app"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/tmux"
)

// Alias SourceMode constants for convenience.
const (
	SourceJSONL = data.SourceJSONL
	SourceCLI   = data.SourceCLI
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	path := flag.String("path", "", "Path to .beads/issues.jsonl file")
	blockTypesFlag := flag.String("block-types", "", "Comma-separated dependency types that count as blockers (default: blocks)")
	statusMode := flag.Bool("status", false, "Output tmux status line and exit")
	showVersion := flag.Bool("version", false, "Print version and exit")
	rigFlag := flag.String("rig", "", "View beads from specific rigs (comma-separated, or 'all' for all rigs in town)")
	flag.Parse()

	if *showVersion {
		fmt.Println("mg", version)
		return
	}

	// Parse blocking types from flag, env var, or default
	blockingTypes := parseBlockingTypes(*blockTypesFlag)

	// Resolve data source: multi-rig, JSONL file, or bd CLI fallback
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	var source data.Source
	if *rigFlag != "" {
		source, err = resolveMultiRigSource(cwd, *rigFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		source = resolveSource(cwd, *path)
	}

	if source.Mode == SourceJSONL && source.Path == "" {
		fmt.Fprintf(os.Stderr, "No .beads/issues.jsonl found and bd not on PATH.\n\n")
		fmt.Fprintf(os.Stderr, "Run mg from inside a project with Beads, or specify a path:\n")
		fmt.Fprintf(os.Stderr, "  mg --path /path/to/.beads/issues.jsonl\n")
		os.Exit(1)
	}

	// Load issues
	var issues []data.Issue
	switch source.Mode {
	case data.SourceMultiRig:
		issues, err = data.FetchIssuesMultiRig(source.Rigs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading cross-rig issues: %v\n\n", err)
			fmt.Fprintf(os.Stderr, "Ensure the Dolt server is running and bd is working.\n")
			os.Exit(1)
		}
	case SourceCLI:
		issues, err = data.FetchIssuesCLI(source.ProjectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading issues via bd list: %v\n\n", err)
			fmt.Fprintf(os.Stderr, "Ensure the Dolt server is running (dolt sql-server) and bd is working.\n")
			os.Exit(1)
		}
	default:
		var skipped int
		issues, skipped, err = data.LoadIssues(source.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading issues from %s: %v\n", source.Path, err)
			os.Exit(1)
		}
		if skipped > 0 {
			fmt.Fprintf(os.Stderr, "Warning: skipped %d malformed line(s) in %s\n", skipped, source.Path)
		}
	}

	if *statusMode {
		groups := data.GroupByParade(issues, blockingTypes)
		fmt.Print(tmux.StatusLine(groups))
		return
	}

	// Run TUI
	guard := app.NewOSCGuard()
	model := app.NewWithGuard(issues, source, blockingTypes, guard)
	p := tea.NewProgram(model, tea.WithFilter(guard.Filter()))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseBlockingTypes builds the blocking types set from flag, env var, or default.
func parseBlockingTypes(flagVal string) map[string]bool {
	raw := flagVal
	if raw == "" {
		raw = os.Getenv("MG_BLOCK_TYPES")
	}
	if raw == "" {
		return data.DefaultBlockingTypes
	}
	types := make(map[string]bool)
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			types[t] = true
		}
	}
	if len(types) == 0 {
		return data.DefaultBlockingTypes
	}
	return types
}

// findBeadsFile walks up from dir looking for .beads/issues.jsonl.
func findBeadsFile(dir string) string {
	for {
		candidate := filepath.Join(dir, ".beads", "issues.jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findBeadsDir walks up from dir looking for a .beads/ directory (even without issues.jsonl).
func findBeadsDir(dir string) string {
	for {
		candidate := filepath.Join(dir, ".beads")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// bdOnPath returns true if the bd command is available.
func bdOnPath() bool {
	_, err := exec.LookPath("bd")
	return err == nil
}

// resolveMultiRigSource builds a multi-rig source from --rig flag value.
func resolveMultiRigSource(cwd, rigFlag string) (data.Source, error) {
	if !bdOnPath() {
		return data.Source{}, fmt.Errorf("bd not on PATH (required for --rig)")
	}

	townRoot := data.FindTownRoot(cwd)
	if townRoot == "" {
		return data.Source{}, fmt.Errorf("not inside a Gas Town directory (no mayor/town.json found)")
	}

	allRigs, err := data.LoadRigsFromTown(townRoot)
	if err != nil {
		return data.Source{}, fmt.Errorf("loading rig list: %w", err)
	}

	names := strings.Split(rigFlag, ",")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}
	rigs := data.FilterRigs(allRigs, names)
	if len(rigs) == 0 {
		return data.Source{}, fmt.Errorf("no rigs matched %q (available: %s)", rigFlag, rigNames(allRigs))
	}

	return data.Source{
		Mode: data.SourceMultiRig,
		Rigs: rigs,
	}, nil
}

// rigNames returns a comma-separated list of rig names.
func rigNames(rigs []data.RigInfo) string {
	names := make([]string, len(rigs))
	for i, r := range rigs {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

// resolveSource determines how mg should load issues.
//
//	--path flag → SourceJSONL with explicit path
//	.beads/ dir exists, bd on PATH → SourceCLI (preferred)
//	.beads/issues.jsonl exists → SourceJSONL (legacy fallback)
//	neither → empty Source (caller should exit with error)
func resolveSource(cwd, pathFlag string) data.Source {
	if pathFlag != "" {
		return data.Source{
			Mode:       data.SourceJSONL,
			Path:       pathFlag,
			ProjectDir: filepath.Dir(filepath.Dir(pathFlag)),
			Explicit:   true,
		}
	}

	// Prefer CLI when bd is available (JSONL removed upstream in beads v0.56+)
	if projectDir := findBeadsDir(cwd); projectDir != "" && bdOnPath() {
		return data.Source{
			Mode:       data.SourceCLI,
			ProjectDir: projectDir,
		}
	}

	// Legacy fallback: JSONL file exists but bd not on PATH
	if jsonlPath := findBeadsFile(cwd); jsonlPath != "" {
		return data.Source{
			Mode:       data.SourceJSONL,
			Path:       jsonlPath,
			ProjectDir: filepath.Dir(filepath.Dir(jsonlPath)),
		}
	}

	// Nothing found
	return data.Source{}
}
