package docsgen

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seletz/odoo-work-cli/internal/cli"
	"github.com/seletz/odoo-work-cli/internal/manual"
)

var (
	refOnce  sync.Once
	refFiles *ReferenceFiles
	refErr   error
)

// reference renders the reference once per test binary: it runs every
// example against the fake backend.
func reference(t *testing.T) *ReferenceFiles {
	t.Helper()
	refOnce.Do(func() { refFiles, refErr = Reference(cli.NewRootCmd()) })
	if refErr != nil {
		t.Fatal(refErr)
	}
	return refFiles
}

func TestReference_OnePagePerCommand(t *testing.T) {
	files := reference(t)
	root := cli.NewRootCmd()
	root.InitDefaultCompletionCmd()
	documented(root, func(c *cobra.Command) {
		name := ManName(c)
		if c == root {
			if _, ok := files.Markdown["index.md"]; !ok {
				t.Error("no overview page index.md")
			}
		} else if _, ok := files.Markdown[name+".md"]; !ok {
			t.Errorf("no reference page %s.md for %q", name, c.CommandPath())
		}
		if _, ok := files.Man[name+".1"]; !ok {
			t.Errorf("no man page %s.1 for %q", name, c.CommandPath())
		}
	})
	for _, n := range []string{"odoo-work-cli-completion-bash.md", "odoo-work-cli-completion-zsh.1"} {
		if _, ok := files.Markdown[n]; ok {
			t.Errorf("%s: the shells belong on the one completion page", n)
		}
		if _, ok := files.Man[n]; ok {
			t.Errorf("%s: the shells belong on the one completion page", n)
		}
	}
}

func TestReference_PageSections(t *testing.T) {
	page := string(reference(t).Markdown["odoo-work-cli-entries-add.md"])
	for _, s := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "GLOBAL OPTIONS", "OUTPUT",
		"ENVIRONMENT", "FILES", "EXIT STATUS", "EXAMPLES", "SEE ALSO"} {
		if !strings.Contains(page, "\n## "+s+"\n") {
			t.Errorf("entries add page lacks section %s", s)
		}
	}
	if !strings.Contains(page, "odoo-work-cli-entries-add - Create a new timesheet entry") {
		t.Errorf("NAME line missing:\n%s", page)
	}
}

func TestReference_Options(t *testing.T) {
	page := string(reference(t).Markdown["odoo-work-cli-entries-add.md"])
	for _, want := range []string{
		"`--hours` *H*",              // value placeholder
		"string; required",           // type and mandatory
		"default: today",             // default from SetFlagDefault
		"env: ODOO_WORK_CLI_BACKEND", // env of a global option
		"`--dry-run`\n:",             // bool flag: no value
		"`--format` *FORMAT*",        // global option
		"[`odoo-work-cli entries`](odoo-work-cli-entries.md)", // see also
	} {
		if !strings.Contains(page, want) {
			t.Errorf("entries add page lacks %q:\n%s", want, page)
		}
	}
}

func TestReference_ExamplesRunAgainstTheFakeBackend(t *testing.T) {
	page := string(reference(t).Markdown["odoo-work-cli-entries-add.md"])
	if !strings.Contains(page, "Dry run: would create this entry") {
		t.Errorf("entries add examples lack the dry-run output:\n%s", page)
	}
	list := string(reference(t).Markdown["odoo-work-cli-clock-list.md"])
	if !strings.Contains(list, "Total:") {
		t.Errorf("clock list examples lack the command output:\n%s", list)
	}
}

func TestReference_JSONFieldsFromSchema(t *testing.T) {
	page := string(reference(t).Markdown["odoo-work-cli-entries.md"])
	for _, want := range []string{"`project_id`", "integer or null", "entry.schema.json"} {
		if !strings.Contains(page, want) {
			t.Errorf("entries OUTPUT lacks %q:\n%s", want, page)
		}
	}
}

func TestReference_ManPage(t *testing.T) {
	man := string(reference(t).Man["odoo-work-cli-entries-add.1"])
	for _, want := range []string{
		`.TH "ODOO-WORK-CLI-ENTRIES-ADD" "1"`,
		".SH NAME\nodoo\\-work\\-cli\\-entries\\-add \\- Create a new timesheet entry",
		".SH SYNOPSIS",
		".SH EXAMPLES",
		`\fB\-\-hours\fR \fIH\fR`,
		`\fBodoo\-work\-cli\-entries\fR(1)`,
	} {
		if !strings.Contains(man, want) {
			t.Errorf("man page lacks %q:\n%s", want, man)
		}
	}
	// A line starting with a dot or quote would be read as a request.
	for _, line := range strings.Split(man, "\n") {
		if strings.HasPrefix(line, "'") {
			t.Errorf("unescaped line %q", line)
		}
	}
}

func TestReference_NoDateStamp(t *testing.T) {
	files := reference(t)
	date := regexp.MustCompile(`\b20\d\d-\d\d-\d\d\b`)
	for name, data := range files.Man {
		th := strings.SplitN(string(data), "\n", 2)[0]
		if date.MatchString(th) || strings.Contains(string(data), "Auto generated") {
			t.Errorf("%s carries a date; the check would fail every day: %s", name, th)
		}
	}
	for name, data := range files.Markdown {
		if strings.Contains(string(data), "Auto generated") {
			t.Errorf("%s carries cobra's dated auto-gen tag", name)
		}
	}
}

func TestReference_OverviewGroupsCommands(t *testing.T) {
	index := string(reference(t).Markdown["index.md"])
	for _, g := range manual.Groups() {
		if !strings.Contains(index, "\n### "+g+"\n") {
			t.Errorf("overview lacks group %q", g)
		}
	}
	for _, want := range []string{"(odoo-work-cli-entries-add.md)", "(odoo-work-cli-clock-edit.md)",
		"(odoo-work-cli-llm.md)", "\n## ENVIRONMENT\n", "\n## FILES\n", "\n## EXIT STATUS\n", "`secrets_locked`"} {
		if !strings.Contains(index, want) {
			t.Errorf("overview lacks %q:\n%s", want, index)
		}
	}
}

func TestCheckDocumented_TreeIsComplete(t *testing.T) {
	if problems := CheckDocumented(cli.NewRootCmd()); len(problems) > 0 {
		t.Fatalf("commands without complete documentation:\n%s", strings.Join(problems, "\n"))
	}
}

func TestCheckDocumented_NamesWhatIsMissing(t *testing.T) {
	root := &cobra.Command{Use: "odoo-work-cli", Short: "s", Long: "l", Example: "  odoo-work-cli x"}
	bare := &cobra.Command{Use: "mcp", Short: "Serve MCP", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(bare)
	got := strings.Join(CheckDocumented(root), "\n")
	for _, want := range []string{"odoo-work-cli mcp", "Long (DESCRIPTION)", "Example (EXAMPLES",
		"manual.Group", manual.GroupAgents} {
		if !strings.Contains(got, want) {
			t.Errorf("message lacks %q:\n%s", want, got)
		}
	}
}

func TestGoreleaser_ListsEveryManPage(t *testing.T) {
	data, err := os.ReadFile("../../.goreleaser.yml")
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for name := range reference(t).Man {
		if !strings.Contains(string(data), "- man/man1/"+name+"\n") {
			missing = append(missing, "      - man/man1/"+name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf(".goreleaser.yml homebrew_casks manpages lacks:\n%s", strings.Join(missing, "\n"))
	}
}

func TestSitePages_MatchTheDocs(t *testing.T) {
	for path, title := range manual.SitePages {
		data, err := os.ReadFile("../../docs/" + path)
		if err != nil {
			t.Errorf("manual.SitePages names %s: %v", path, err)
			continue
		}
		if !strings.Contains(string(data), "\ntitle: "+title+"\n") {
			t.Errorf("manual.SitePages[%q] = %q, but the page has another title", path, title)
		}
	}
}
