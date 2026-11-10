package docsgen

import (
	"fmt"
	"regexp"
	"strings"
)

// manPage is one page of the command reference, independent of its
// output format. renderMarkdown and renderRoff write it for the site and
// as a man page.
type manPage struct {
	name     string // man page name: odoo-work-cli-entries-add
	title    string // heading on the site: odoo-work-cli entries add
	short    string // NAME description
	weight   int    // position in the site navigation
	sections []section
}

type section struct {
	title  string
	blocks []block
}

// block is one of: para (text), pre (preformatted lines), defs, links,
// sub (a subsection heading).
type block struct {
	kind  blockKind
	text  string
	defs  []def
	links []link
}

type blockKind int

const (
	blockPara blockKind = iota
	blockPre
	blockDefs
	blockLinks
	blockSub
	blockSynopsis
)

// def is a definition list item. body holds paragraphs of plain text;
// meta is an optional italic line after them (type, default, env).
type def struct {
	term []span
	body []string
	meta string
}

type spanStyle int

const (
	styleCode spanStyle = iota // --hours, ODOO_URL: bold in roff
	styleVar                   // H, YYYY-MM-DD: italic
	stylePlain
)

type span struct {
	text  string
	style spanStyle
	// href and man make the span a link: href on the site, man as
	// name(1) in roff.
	href, man string
}

// link is a SEE ALSO item: a command (man set) or a site page (url set).
type link struct {
	text string
	href string // site link, relative to the commands directory
	man  string // man page name, for commands
	url  string // absolute URL, for site pages in roff
}

func para(s string) block          { return block{kind: blockPara, text: s} }
func pre(s string) block           { return block{kind: blockPre, text: s} }
func sub(s string) block           { return block{kind: blockSub, text: s} }
func defsBlock(d []def) block      { return block{kind: blockDefs, defs: d} }
func linksBlock(l []link) block    { return block{kind: blockLinks, links: l} }
func synopsisBlock(s string) block { return block{kind: blockSynopsis, text: s} }

var preLine = regexp.MustCompile(`^(  |\t)`)

// textBlocks splits help text (cobra Long, the Output annotation) into
// paragraphs at blank lines. A paragraph whose lines are all indented is
// preformatted; its common indentation is removed.
func textBlocks(text string) []block {
	var out []block
	for _, p := range splitParagraphs(text) {
		lines := strings.Split(p, "\n")
		indented := true
		for _, l := range lines {
			if !preLine.MatchString(l) {
				indented = false
				break
			}
		}
		if indented {
			out = append(out, pre(dedent(lines)))
			continue
		}
		out = append(out, para(p))
	}
	return out
}

func splitParagraphs(text string) []string {
	var out []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, l := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		cur = append(cur, strings.TrimRight(l, " \t"))
	}
	flush()
	return out
}

func dedent(lines []string) string {
	min := -1
	for _, l := range lines {
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if min < 0 || n < min {
			min = n
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l[min:]
	}
	return strings.Join(out, "\n")
}

// String is for test failure messages.
func (s span) String() string { return fmt.Sprintf("%q", s.text) }
