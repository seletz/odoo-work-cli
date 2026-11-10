package docsgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seletz/odoo-work-cli/internal/cli"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

// example is one item of a command's EXAMPLES: a description and the
// command lines that belong to it.
type example struct {
	text  string
	lines []string
}

// parseExamples reads a cobra Example: "# text" lines describe the
// command lines that follow them; a blank line ends an example.
func parseExamples(s string) []example {
	var out []example
	var cur example
	flush := func() {
		if cur.text != "" || len(cur.lines) > 0 {
			out = append(out, cur)
		}
		cur = example{}
	}
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		switch {
		case l == "":
			flush()
		case strings.HasPrefix(l, "#"):
			if len(cur.lines) > 0 {
				flush()
			}
			t := strings.TrimSpace(strings.TrimPrefix(l, "#"))
			if cur.text != "" {
				t = cur.text + " " + t
			}
			cur.text = t
		default:
			cur.lines = append(cur.lines, l)
		}
	}
	flush()
	return out
}

// usageFailures are messages of a stale example: the command line no
// longer parses.
var usageFailures = []string{"unknown flag", "unknown command", "unknown shorthand", "required flag", "accepts "}

// runExamples renders the examples of cmd as console blocks. Unless the
// command has StaticExamples, every odoo-work-cli line runs in-process
// against one fresh copy of the demo state (so the examples of a page
// build on each other) with the fake backend, and its output follows it.
//
// :param cmd: the command
// :returns: the EXAMPLES blocks, or an error for an example that no
//
//	longer parses
func runExamples(cmd *cobra.Command) ([]block, error) {
	examples := parseExamples(cmd.Example)
	run := cmd.Annotations[manual.StaticExamples] != "true"
	var state string
	if run {
		dir, err := os.MkdirTemp("", "odoo-work-cli-examples-")
		if err != nil {
			return nil, err
		}
		defer func() { _ = os.RemoveAll(dir) }()
		state = filepath.Join(dir, "state.json")
		if err := WriteDemoState(state); err != nil {
			return nil, err
		}
		restore := setenv(map[string]string{
			fake.BackendEnvVar:    "fake",
			"ODOO_WORK_CLI_NO_OP": "1",
			fake.StateEnvVar:      state,
		})
		defer restore()
	}

	var out []block
	for _, ex := range examples {
		if ex.text != "" {
			t := ex.text
			if !strings.ContainsAny(t[len(t)-1:], ".:!?") {
				t += ":"
			}
			out = append(out, para(t))
		}
		var console []string
		for _, l := range ex.lines {
			console = append(console, "$ "+l)
			if !run || !strings.HasPrefix(l, "odoo-work-cli ") {
				continue
			}
			args, err := splitWords(strings.TrimPrefix(l, "odoo-work-cli "))
			if err != nil {
				return nil, fmt.Errorf("%s: example %q: %w", cmd.CommandPath(), l, err)
			}
			var stdout, stderr bytes.Buffer
			cli.Run(args, &stdout, &stderr)
			text := strings.TrimRight(stdout.String()+stderr.String(), "\n")
			for _, f := range usageFailures {
				if strings.Contains(text, f) {
					return nil, fmt.Errorf("%s: example %q no longer parses:\n%s", cmd.CommandPath(), l, text)
				}
			}
			if text != "" {
				console = append(console, text)
			}
		}
		if len(console) > 0 {
			out = append(out, pre(strings.Join(console, "\n")))
		}
	}
	return out, nil
}

// splitWords splits a command line into words like a POSIX shell for the
// simple cases examples use: blanks, single and double quotes.
func splitWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		case r == '|' || r == '>' || r == ';' || r == '&':
			return nil, fmt.Errorf("shell operator %q: examples that run must be a single command", r)
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}
