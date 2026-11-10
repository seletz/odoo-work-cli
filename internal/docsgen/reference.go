package docsgen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
)

// ReferenceFiles is the generated command reference: Markdown pages for
// the site (docs/reference/commands) and man pages (man/man1), keyed by
// file name.
type ReferenceFiles struct {
	Markdown map[string][]byte
	Man      map[string][]byte
}

// ManName returns the man page name of cmd (see manual.ManName).
//
// :param cmd: the command
// :returns: the name, without section
func ManName(cmd *cobra.Command) string { return manual.ManName(cmd) }

// Reference renders the command reference of root: odoo-work-cli(1) as
// the overview (index.md on the site) and one page per command, all
// shells of completion on one page. The content comes from the commands
// (see package manual); the examples run against the fake backend.
//
// :param root: the root command (cli.NewRootCmd())
// :returns: the pages
func Reference(root *cobra.Command) (*ReferenceFiles, error) {
	root.InitDefaultCompletionCmd()
	if problems := CheckDocumented(root); len(problems) > 0 {
		return nil, fmt.Errorf("commands without complete documentation:\n%s", strings.Join(problems, "\n"))
	}
	files := &ReferenceFiles{Markdown: map[string][]byte{}, Man: map[string][]byte{}}
	var cmds []*cobra.Command
	documented(root, func(c *cobra.Command) {
		if c != root {
			cmds = append(cmds, c)
		}
	})
	weights := navWeights(cmds)
	for _, c := range cmds {
		p, err := commandPage(c)
		if err != nil {
			return nil, err
		}
		p.weight = weights[c]
		files.Markdown[p.name+".md"] = renderMarkdown(p)
		files.Man[p.name+".1"] = renderRoff(p)
	}
	p, err := overviewPage(root, cmds)
	if err != nil {
		return nil, err
	}
	files.Markdown["index.md"] = renderMarkdown(p)
	files.Man[p.name+".1"] = renderRoff(p)
	return files, nil
}

// documented calls fn for every command that has a page of its own:
// root, every available command, but not the shells below completion.
func documented(root *cobra.Command, fn func(c *cobra.Command)) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.HasParent() && (!c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand()) {
			return
		}
		fn(c)
		if isCompletion(c) {
			return
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

func isCompletion(c *cobra.Command) bool {
	return c.HasParent() && !c.Parent().HasParent() && c.Name() == "completion"
}

// navWeights orders the pages in the site navigation by group, then by
// command path.
func navWeights(cmds []*cobra.Command) map[*cobra.Command]int {
	rank := map[string]int{}
	for i, g := range manual.Groups() {
		rank[g] = i
	}
	sorted := append([]*cobra.Command(nil), cmds...)
	sort.SliceStable(sorted, func(i, j int) bool {
		gi, gj := rank[manual.GroupOf(sorted[i])], rank[manual.GroupOf(sorted[j])]
		if gi != gj {
			return gi < gj
		}
		return sorted[i].CommandPath() < sorted[j].CommandPath()
	})
	w := map[*cobra.Command]int{}
	for i, c := range sorted {
		w[c] = 100 + i
	}
	return w
}

// CheckDocumented lists what the commands of root lack for a complete
// reference page. An empty result means every page has NAME, SYNOPSIS,
// DESCRIPTION and EXAMPLES, and every top-level command a group.
//
// :param root: the root command
// :returns: one line per command with gaps
func CheckDocumented(root *cobra.Command) []string {
	root.InitDefaultCompletionCmd()
	var problems []string
	documented(root, func(c *cobra.Command) {
		var missing []string
		if strings.TrimSpace(c.Short) == "" {
			missing = append(missing, "Short (NAME)")
		}
		if strings.TrimSpace(c.Long) == "" {
			missing = append(missing, "Long (DESCRIPTION)")
		}
		if !hasExampleLine(c.Example) {
			missing = append(missing, `Example (EXAMPLES; at least one "odoo-work-cli …" line)`)
		}
		if c.HasParent() && !c.Parent().HasParent() && !validGroup(c.Annotations[manual.Group]) {
			missing = append(missing, fmt.Sprintf("Annotations[manual.Group] (its group in odoo-work-cli(1), one of: %s)",
				strings.Join(manual.Groups(), ", ")))
		}
		c.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
			if strings.TrimSpace(f.Usage) == "" {
				missing = append(missing, "a help text for --"+f.Name)
			}
		})
		if len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("%s: missing %s (see the package doc of internal/manual)",
				c.CommandPath(), strings.Join(missing, ", ")))
		}
	})
	return problems
}

func hasExampleLine(s string) bool {
	for _, ex := range parseExamples(s) {
		for _, l := range ex.lines {
			if strings.HasPrefix(l, "odoo-work-cli") {
				return true
			}
		}
	}
	return false
}

func validGroup(g string) bool {
	for _, v := range manual.Groups() {
		if g == v {
			return true
		}
	}
	return false
}

// commandPage builds the page of one command.
func commandPage(c *cobra.Command) (*manPage, error) {
	p := &manPage{name: ManName(c), title: c.CommandPath(), short: c.Short}
	add := func(title string, blocks ...block) {
		if len(blocks) > 0 {
			p.sections = append(p.sections, section{title: title, blocks: blocks})
		}
	}
	add("NAME", para(p.name+" - "+c.Short))
	add("SYNOPSIS", synopsisBlock(strings.Join(synopsis(c), "\n")))

	desc := textBlocks(c.Long)
	if isCompletion(c) {
		for _, sh := range c.Commands() {
			if sh.IsAvailableCommand() {
				desc = append(desc, sub(sh.Name()))
				desc = append(desc, textBlocks(sh.Long)...)
			}
		}
	}
	if len(c.Commands()) > 0 && !isCompletion(c) {
		desc = append(desc, para("Subcommands:"), defsBlock(subcommandDefs(c)))
	}
	add("DESCRIPTION", desc...)

	add("OPTIONS", optionsBlocks(c)...)
	add("GLOBAL OPTIONS", defsBlock(globalOptionDefs(c)))

	out, err := outputBlocks(c)
	if err != nil {
		return nil, err
	}
	add("OUTPUT", out...)
	add("ENVIRONMENT", envBlocks(c)...)
	add("FILES", fileBlocks(c)...)
	add("EXIT STATUS", exitBlocks(c, manual.KindsOf(c))...)

	ex, err := runExamples(c)
	if err != nil {
		return nil, err
	}
	add("EXAMPLES", ex...)

	links, err := seeAlso(c)
	if err != nil {
		return nil, err
	}
	add("SEE ALSO", linksBlock(links))
	return p, nil
}

// overviewPage builds odoo-work-cli(1): the commands by group and,
// once, the environment, files, exit status and error kinds.
func overviewPage(root *cobra.Command, cmds []*cobra.Command) (*manPage, error) {
	p := &manPage{name: ManName(root), title: "Commands", short: root.Short, weight: 10}
	add := func(title string, blocks ...block) {
		if len(blocks) > 0 {
			p.sections = append(p.sections, section{title: title, blocks: blocks})
		}
	}
	add("NAME", para(p.name+" - "+root.Short))
	add("SYNOPSIS", synopsisBlock("odoo-work-cli COMMAND [OPTIONS] [ARGUMENTS]\nodoo-work-cli --version\nodoo-work-cli --help"))
	add("DESCRIPTION", textBlocks(root.Long)...)

	var commands []block
	for _, g := range manual.Groups() {
		var defs []def
		for _, c := range cmds {
			if manual.GroupOf(c) == g {
				defs = append(defs, def{term: []span{commandLink(c)}, body: []string{c.Short + "."}})
			}
		}
		if len(defs) > 0 {
			commands = append(commands, sub(g), defsBlock(defs))
		}
	}
	add("COMMANDS", commands...)

	global := globalOptionDefs(root)
	global = append(global, def{term: []span{{text: "-v", style: styleCode}, {text: ", ", style: stylePlain},
		{text: "--version", style: styleCode}}, body: []string{"Print the version and exit."}})
	add("GLOBAL OPTIONS", para("Every command accepts --config, --backend and --help. Commands that print data also accept --json and --format."),
		defsBlock(global))
	add("ENVIRONMENT", envDefsBlocks(append(manual.OdooEnv(), manual.MCPSecretsEnv))...)
	add("FILES", fileDefsBlocks(manual.OnlineFiles())...)
	add("EXIT STATUS", exitBlocks(root, nil)...)
	ex, err := runExamples(root)
	if err != nil {
		return nil, err
	}
	add("EXAMPLES", ex...)
	links, err := seeAlso(root)
	if err != nil {
		return nil, err
	}
	add("SEE ALSO", linksBlock(links))
	return p, nil
}

func commandLink(c *cobra.Command) span {
	href := ManName(c) + ".md"
	if !c.HasParent() {
		href = "index.md"
	}
	return span{text: c.CommandPath(), style: styleCode, href: href, man: ManName(c)}
}

func subcommandDefs(c *cobra.Command) []def {
	var defs []def
	for _, s := range c.Commands() {
		if s.IsAvailableCommand() && !s.IsAdditionalHelpTopicCommand() {
			defs = append(defs, def{term: []span{commandLink(s)}, body: []string{s.Short + "."}})
		}
	}
	return defs
}

// Output flags and flags documented under GLOBAL OPTIONS.
var globalFlagNames = map[string]bool{"format": true, "json": true, "help": true, "config": true, "backend": true, "version": true}

// ownFlags returns the flags shown under OPTIONS: those of the command,
// for completion those of its shells.
func ownFlags(c *cobra.Command) []*pflag.Flag {
	seen := map[string]bool{}
	var out []*pflag.Flag
	collect := func(cmd *cobra.Command) {
		cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
			if !globalFlagNames[f.Name] && !f.Hidden && !seen[f.Name] {
				seen[f.Name] = true
				out = append(out, f)
			}
		})
	}
	collect(c)
	if isCompletion(c) {
		for _, sh := range c.Commands() {
			collect(sh)
		}
	}
	return out
}

func isRequired(f *pflag.Flag) bool { return manual.FlagRequired(f) }

// placeholder returns the value name of a flag: the `quoted` word of its
// help text, else one derived from its type; "" for a boolean flag.
func placeholder(f *pflag.Flag) string {
	if f.Value.Type() == "bool" {
		return ""
	}
	name, _ := pflag.UnquoteUsage(f)
	if strings.Contains(f.Usage, "`") {
		return name
	}
	switch f.Value.Type() {
	case "int", "int64":
		return "N"
	default:
		return "VALUE"
	}
}

func typeName(f *pflag.Flag) string {
	switch t := f.Value.Type(); {
	case t == "bool":
		return "flag"
	case t == "int" || t == "int64":
		return "integer"
	case strings.Contains(t, "|"):
		return "one of " + strings.ReplaceAll(t, "|", ", ")
	default:
		return t
	}
}

func flagDefault(f *pflag.Flag) string {
	if d := manual.FlagDefault(f); d != "" {
		return d
	}
	switch f.DefValue {
	case "", "0", "false", "[]":
		return ""
	}
	return f.DefValue
}

func flagTerm(f *pflag.Flag) []span {
	var t []span
	if f.Shorthand != "" {
		t = append(t, span{text: "-" + f.Shorthand, style: styleCode}, span{text: ", ", style: stylePlain})
	}
	t = append(t, span{text: "--" + f.Name, style: styleCode})
	if ph := placeholder(f); ph != "" {
		t = append(t, span{text: " ", style: stylePlain}, span{text: ph, style: styleVar})
	}
	return t
}

func flagDef(f *pflag.Flag) def {
	_, usage := pflag.UnquoteUsage(f)
	// --help shows required and default in the text; here they are in meta.
	usage = strings.TrimSuffix(usage, manual.HelpSuffix(f))
	meta := []string{typeName(f)}
	if isRequired(f) {
		meta = append(meta, "required")
	} else {
		meta = append(meta, "optional")
	}
	if d := flagDefault(f); d != "" {
		meta = append(meta, "default: "+d)
	}
	if e := manual.FlagEnv(f); e != "" {
		meta = append(meta, "env: "+e)
	}
	return def{term: flagTerm(f), body: []string{capitalize(usage) + "."}, meta: strings.Join(meta, "; ")}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func optionsBlocks(c *cobra.Command) []block {
	var defs []def
	for _, f := range ownFlags(c) {
		defs = append(defs, flagDef(f))
	}
	if len(defs) == 0 {
		return []block{para("None besides the global options.")}
	}
	return []block{defsBlock(defs)}
}

func lookupFlag(c *cobra.Command, name string) *pflag.Flag {
	if f := c.Flags().Lookup(name); f != nil {
		return f
	}
	return c.InheritedFlags().Lookup(name)
}

func hasJSON(c *cobra.Command) bool { return lookupFlag(c, "json") != nil }

// globalOptionDefs lists --format/--json (where the command has them),
// the root's persistent flags and --help.
func globalOptionDefs(c *cobra.Command) []def {
	var defs []def
	for _, name := range []string{"format", "json"} {
		if f := lookupFlag(c, name); f != nil {
			defs = append(defs, flagDef(f))
		}
	}
	if !c.HasParent() {
		defs = nil
		for _, name := range []string{"format", "json"} {
			if f := lookupFlag(findCommand(c, "entries"), name); f != nil {
				defs = append(defs, flagDef(f))
			}
		}
	}
	c.Root().PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			defs = append(defs, flagDef(f))
		}
	})
	defs = append(defs, def{term: []span{{text: "-h", style: styleCode}, {text: ", ", style: stylePlain},
		{text: "--help", style: styleCode}}, body: []string{"Print the help of the command and exit."}})
	return defs
}

func findCommand(root *cobra.Command, path string) *cobra.Command {
	c, _, err := root.Find(strings.Fields(path))
	if err != nil {
		return root
	}
	return c
}

// synopsis returns the forms of cmd: the Synopsis annotation, or one
// generated from the arguments in Use and the flags.
func synopsis(c *cobra.Command) []string {
	if s := strings.TrimSpace(c.Annotations[manual.Synopsis]); s != "" {
		var forms []string
		for _, l := range strings.Split(s, "\n") {
			forms = append(forms, strings.TrimSpace(l))
		}
		return forms
	}
	if isCompletion(c) {
		var forms []string
		for _, sh := range c.Commands() {
			if sh.IsAvailableCommand() {
				forms = append(forms, synopsisLine(sh))
			}
		}
		return forms
	}
	var forms []string
	if c.Runnable() {
		forms = append(forms, synopsisLine(c))
	}
	if c.HasAvailableSubCommands() {
		forms = append(forms, c.CommandPath()+" COMMAND [OPTIONS]")
	}
	return forms
}

func synopsisLine(c *cobra.Command) string {
	parts := []string{c.CommandPath()}
	var opt []string
	for _, f := range ownFlags(c) {
		item := "--" + f.Name
		if ph := placeholder(f); ph != "" {
			item += " " + ph
		}
		if isRequired(f) {
			parts = append(parts, item)
		} else {
			opt = append(opt, "["+item+"]")
		}
	}
	parts = append(parts, opt...)
	if hasJSON(c) {
		parts = append(parts, "[--json | --format FORMAT]")
	}
	if args := strings.Fields(c.Use); len(args) > 1 {
		parts = append(parts, args[1:]...)
	}
	return strings.Join(parts, " ")
}

func outputBlocks(c *cobra.Command) ([]block, error) {
	blocks := textBlocks(c.Annotations[manual.Output])
	// A later schema (entry_dry_run after entry) lists only the fields
	// that are new or differ.
	shown := map[string]string{}
	var first string
	for _, s := range manual.List(c, manual.Schemas) {
		fields, err := schemaFields(s)
		if err != nil {
			return nil, fmt.Errorf("%s: schema %s: %w", c.CommandPath(), s, err)
		}
		var news []def
		for _, d := range fields {
			key := d.term[0].text
			if prev, ok := shown[key]; ok && prev == d.meta+strings.Join(d.body, "") {
				continue
			}
			shown[key] = d.meta + strings.Join(d.body, "")
			news = append(news, d)
		}
		intro := fmt.Sprintf("Fields of %s.schema.json:", s)
		if first == "" {
			first = s
		} else {
			intro = fmt.Sprintf("Fields of %s.schema.json: those of %s.schema.json, except:", s, first)
		}
		blocks = append(blocks, para(intro), defsBlock(news))
	}
	return blocks, nil
}

// listOrDefault returns the values of annotation key, or def when the
// command does not set the key at all ("" in the annotation means none).
func listOrDefault(c *cobra.Command, key string, def []string) []string {
	if _, ok := c.Annotations[key]; ok {
		return manual.List(c, key)
	}
	if manual.IsOffline(c) {
		return nil
	}
	return def
}

func envBlocks(c *cobra.Command) []block {
	return envDefsBlocks(listOrDefault(c, manual.Env, manual.OdooEnv()))
}

func envDefsBlocks(vars []string) []block {
	if len(vars) == 0 {
		return nil
	}
	var defs []def
	for _, v := range vars {
		defs = append(defs, def{term: []span{{text: v, style: styleCode}}, body: []string{manual.EnvHelp[v]}})
	}
	return []block{defsBlock(defs)}
}

func fileBlocks(c *cobra.Command) []block {
	return fileDefsBlocks(listOrDefault(c, manual.Files, manual.OnlineFiles()))
}

func fileDefsBlocks(files []string) []block {
	if len(files) == 0 {
		return nil
	}
	var defs []def
	for _, f := range files {
		defs = append(defs, def{term: []span{{text: f, style: styleCode}}, body: []string{manual.FileHelp[f]}})
	}
	return []block{defsBlock(defs)}
}

// exitBlocks describes the exit status and the error kinds of c; nil
// kinds means all of them (the overview).
func exitBlocks(c *cobra.Command, kinds []apperr.Kind) []block {
	all := kinds == nil
	if all {
		kinds = apperr.Kinds()
	}
	text := "0 on success, 1 on any error, including usage errors."
	if all || hasJSON(c) {
		text += " With --json or --format json|jsonl the error is printed on standard output as one JSON object, " +
			`{"error": {"kind": KIND, "message": TEXT}}, and nothing on standard error; otherwise as "Error: TEXT" on ` +
			"standard error, after usage errors followed by the usage."
	} else {
		text += ` The error is printed as "Error: TEXT" on standard error.`
	}
	intro := "Error kinds this command can fail with:"
	if all {
		intro = "Error kinds (the kind field of the JSON error):"
	}
	var defs []def
	for _, k := range kinds {
		defs = append(defs, def{term: []span{{text: string(k), style: styleCode}}, body: []string{manual.KindHelp[k]}})
	}
	return []block{para(text), para(intro), defsBlock(defs)}
}

// seeAlso lists the parent, the subcommands, the SeeAlso annotation and
// the reference pages that apply.
func seeAlso(c *cobra.Command) ([]link, error) {
	var links []link
	seen := map[string]bool{}
	addCmd := func(t *cobra.Command) {
		if t == c || seen[t.CommandPath()] {
			return
		}
		seen[t.CommandPath()] = true
		s := commandLink(t)
		links = append(links, link{text: s.text, href: s.href, man: s.man})
	}
	addPage := func(path string) error {
		title, ok := manual.SitePages[path]
		if !ok {
			return fmt.Errorf("%s: SeeAlso names %q, which is not in manual.SitePages", c.CommandPath(), path)
		}
		if seen[path] {
			return nil
		}
		seen[path] = true
		links = append(links, link{text: title, href: "../../" + path,
			url: manual.SiteURL + strings.TrimSuffix(path, ".md") + "/"})
		return nil
	}

	if c.HasParent() {
		addCmd(c.Parent())
	}
	if !isCompletion(c) && c.HasParent() {
		for _, s := range c.Commands() {
			if s.IsAvailableCommand() && !s.IsAdditionalHelpTopicCommand() {
				addCmd(s)
			}
		}
	}
	var pages []string
	for _, item := range manual.List(c, manual.SeeAlso) {
		if strings.HasSuffix(item, ".md") {
			pages = append(pages, item)
			continue
		}
		t, rest, err := c.Root().Find(strings.Fields(strings.TrimPrefix(item, "odoo-work-cli")))
		if err != nil || len(rest) > 0 || t.CommandPath() != item {
			return nil, fmt.Errorf("%s: SeeAlso names %q, which is not a command", c.CommandPath(), item)
		}
		addCmd(t)
	}
	for _, p := range pages {
		if err := addPage(p); err != nil {
			return nil, err
		}
	}
	if len(manual.List(c, manual.Schemas)) > 0 {
		_ = addPage("reference/20-json-output.md")
	}
	if c.HasParent() {
		_ = addPage("reference/30-errors.md")
		if !manual.IsOffline(c) {
			_ = addPage("reference/40-configuration.md")
		}
	}
	return links, nil
}

// WriteReference writes the Markdown pages into mdDir (docs/reference/
// commands) and the man pages into manDir (man/man1), and removes pages
// of commands that no longer exist.
//
// :param mdDir: the directory of the site pages
// :param manDir: the directory of the man pages
// :param files: the pages from Reference
// :returns: an error if writing fails
func WriteReference(mdDir, manDir string, files *ReferenceFiles) error {
	if err := writeDir(mdDir, ".md", files.Markdown); err != nil {
		return err
	}
	return writeDir(manDir, ".1", files.Man)
}
