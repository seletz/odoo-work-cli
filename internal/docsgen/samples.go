package docsgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/cli"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

// Sample is one CLI invocation whose output the docs show. Only commands
// whose output does not depend on the wall clock qualify: explicit
// --week/--date flags, no running attendance in the listed range.
type Sample struct {
	Name string
	Args []string
	// JSON marks samples whose stdout is JSON (for highlighting and tests).
	JSON bool
}

// CommandLine returns the invocation as typed in a shell.
//
// :returns: the command line
func (s Sample) CommandLine() string {
	parts := []string{"odoo-work-cli"}
	for _, a := range s.Args {
		if strings.ContainsAny(a, " \"'$") {
			a = fmt.Sprintf("%q", a)
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

// Samples are the CLI outputs of the docs, written to docs/assets/cli.
var Samples = []Sample{
	{Name: "whoami", Args: []string{"whoami"}},
	{Name: "projects", Args: []string{"projects"}},
	{Name: "projects-search-json", Args: []string{"projects", "--search", "portal", "--json"}, JSON: true},
	{Name: "tasks", Args: []string{"tasks", "1"}},
	{Name: "timesheets", Args: []string{"timesheets", "--week", DemoWeek}},
	{Name: "entries-date", Args: []string{"entries", "--date", "2026-04-08"}},
	{Name: "entries-date-json", Args: []string{"entries", "--date", "2026-04-09", "--json"}, JSON: true},
	{Name: "entries-add-dry-run", Args: []string{"entries", "add", "--project-id", "1", "--task-id", "13",
		"--date", "2026-04-09", "--hours", "1:30", "--description", "Review fuzzy matching", "--dry-run"}},
	{Name: "entries-add-dry-run-json", Args: []string{"entries", "add", "--project-id", "1", "--task-id", "13",
		"--date", "2026-04-09", "--hours", "1:30", "--description", "Review fuzzy matching", "--dry-run", "--json"}, JSON: true},
	{Name: "clock-list-week", Args: []string{"clock", "list", "--week", "2026-W14"}},
	{Name: "clock-list-date-json", Args: []string{"clock", "list", "--date", "2026-04-08", "--json"}, JSON: true},
	{Name: "error-not-found", Args: []string{"entries", "delete", "999", "--json"}, JSON: true},
	{Name: "error-validation", Args: []string{"entries", "add", "--project-id", "1", "--hours", "abc", "--description", "x", "--json"}, JSON: true},
}

// SampleOutput is the result of running a Sample.
type SampleOutput struct {
	Sample Sample
	Output string // stdout, plus stderr for table output
	Code   int
}

// RunSamples runs every one of Samples in-process against a fresh copy of
// the demo state with the fake backend.
//
// :returns: the outputs in the order of Samples
func RunSamples() ([]SampleOutput, error) {
	dir, err := os.MkdirTemp("", "odoo-work-cli-samples-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	restore := setenv(map[string]string{
		fake.BackendEnvVar:    "fake",
		"ODOO_WORK_CLI_NO_OP": "1",
	})
	defer restore()

	out := make([]SampleOutput, 0, len(Samples))
	for i, s := range Samples {
		state := filepath.Join(dir, fmt.Sprintf("state-%d.json", i))
		if err := WriteDemoState(state); err != nil {
			return nil, err
		}
		if err := os.Setenv(fake.StateEnvVar, state); err != nil {
			return nil, err
		}
		var stdout, stderr bytes.Buffer
		code := cli.Run(s.Args, &stdout, &stderr)
		text := stdout.String()
		if !s.JSON {
			text += stderr.String()
		}
		out = append(out, SampleOutput{Sample: s, Output: text, Code: code})
	}
	return out, nil
}

// setenv sets vars (plus the state variable's old value) and returns a
// function that restores the previous environment.
func setenv(vars map[string]string) func() {
	old := map[string]*string{}
	for k := range vars {
		old[k] = nil
	}
	old[fake.StateEnvVar] = nil
	for k := range old {
		if v, ok := os.LookupEnv(k); ok {
			old[k] = &v
		}
	}
	for k, v := range vars {
		_ = os.Setenv(k, v)
	}
	return func() {
		for k, v := range old {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
	}
}

// WriteSamples writes one <name>.txt per sample into dir and removes
// stale ones.
//
// :param dir: the output directory (docs/assets/cli)
// :param outs: the sample outputs
// :returns: an error if writing fails
func WriteSamples(dir string, outs []SampleOutput) error {
	files := map[string][]byte{}
	for _, o := range outs {
		files[o.Sample.Name+".txt"] = []byte(strings.TrimRight(o.Output, "\n") + "\n")
	}
	return writeDir(dir, ".txt", files)
}
