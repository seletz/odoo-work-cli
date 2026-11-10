package mcp

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"
)

// minString is the length below which a text value is not shortened further.
const minString = 64

// node is a JSON value with its serialized (compact) size, kept current
// while entries are dropped.
type node struct {
	kind   byte // 'o' object, 'a' array, 's' string, 'l' other leaf
	keys   []string
	kids   []*node
	raw    []byte // leaves
	size   int
	parent *node
	path   string // arrays: path with [] for list levels
	tier   int    // arrays: 0 without nested lists, else 1 + the highest tier below
	total  int    // arrays: original entry count
}

func parse(raw []byte, parent *node, path string) *node {
	n := &node{parent: parent, raw: raw, size: len(raw)}
	switch raw[0] {
	case '{':
		n.kind = 'o'
		var m map[string]json.RawMessage
		_ = json.Unmarshal(raw, &m)
		n.size = 2
		for i, k := range orderedKeys(raw) {
			kb, _ := json.Marshal(k)
			child := parse(m[k], n, joinPath(path, k))
			n.keys = append(n.keys, k)
			n.kids = append(n.kids, child)
			n.size += len(kb) + 1 + child.size
			if i > 0 {
				n.size++
			}
		}
	case '[':
		n.kind = 'a'
		n.path = path
		var items []json.RawMessage
		_ = json.Unmarshal(raw, &items)
		n.size = 2
		for i, it := range items {
			child := parse(it, n, path+"[]")
			n.kids = append(n.kids, child)
			n.size += child.size
			if i > 0 {
				n.size++
			}
			if t := child.height(); t+1 > n.tier {
				n.tier = t + 1
			}
		}
		n.total = len(items)
	case '"':
		n.kind = 's'
	default:
		n.kind = 'l'
	}
	return n
}

// height is the array tier of an array, else the highest tier below it, so
// that a list in an object in a list counts as nested.
func (n *node) height() int {
	switch n.kind {
	case 'a':
		return n.tier
	case 'o':
		h := 0
		for _, k := range n.kids {
			if t := k.height(); t > h {
				h = t
			}
		}
		return h
	}
	return 0
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// grow changes the size of n and of all its ancestors.
func (n *node) grow(delta int) {
	for p := n; p != nil; p = p.parent {
		p.size += delta
	}
}

func (n *node) write(b *bytes.Buffer) {
	switch n.kind {
	case 'o':
		b.WriteByte('{')
		for i, k := range n.kids {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, _ := json.Marshal(n.keys[i])
			b.Write(kb)
			b.WriteByte(':')
			k.write(b)
		}
		b.WriteByte('}')
	case 'a':
		b.WriteByte('[')
		for i, k := range n.kids {
			if i > 0 {
				b.WriteByte(',')
			}
			k.write(b)
		}
		b.WriteByte(']')
	default:
		b.Write(n.raw)
	}
}

// walk calls f for every node below and including n.
func (n *node) walk(f func(*node)) {
	f(n)
	for _, k := range n.kids {
		k.walk(f)
	}
}

// cutNested shortens b, a compact JSON document, to at most max bytes when
// it can: it drops trailing entries of detail lists, deepest tier first and
// the largest list within a tier first, one entry at a time, then shortens
// the longest strings. Detail names the lists that may be cut (the last
// element of their path, e.g. "sessions"); every other list carries sums
// or is the data itself and stays whole, and numbers are never touched.
// It is the last resort for a page that holds a single entry over the
// bound (see paginate); the caller must check the size of the result.
func cutNested(b []byte, max int, detail []string) ([]byte, *Hint) {
	root := parse(b, nil, "")
	var lists []*node
	var strs []*node
	root.walk(func(n *node) {
		if n.kind == 'a' {
			lists = append(lists, n)
		}
	})
	maxTier := 0
	for _, l := range lists {
		maxTier = max2(maxTier, l.tier)
	}
	for tier := 0; tier <= maxTier && root.size > max; tier++ {
		var cand []*node
		for _, l := range lists {
			if l.tier == tier && len(l.kids) > 0 && isDetail(l.path, detail) {
				cand = append(cand, l)
			}
		}
		for root.size > max {
			var best *node
			for _, l := range cand {
				if len(l.kids) > 0 && (best == nil || l.size > best.size) {
					best = l
				}
			}
			if best == nil {
				break
			}
			last := best.kids[len(best.kids)-1]
			delta := last.size
			if len(best.kids) > 1 {
				delta++
			}
			best.kids = best.kids[:len(best.kids)-1]
			best.grow(-delta)
		}
	}
	live := map[*node]bool{}
	root.walk(func(n *node) { live[n] = true })
	h := &Hint{}
	byPath := map[string]*Cut{}
	cutPath := map[string]bool{}
	for _, l := range lists {
		c := byPath[l.path]
		if c == nil {
			c = &Cut{Path: l.path}
			byPath[l.path] = c
			h.Cut = append(h.Cut, Cut{Path: l.path}) // first-appearance order
		}
		c.Lists++
		if live[l] {
			c.Kept += len(l.kids)
		}
		c.Total += l.total
		if !live[l] || len(l.kids) < l.total {
			cutPath[l.path] = true
		}
	}
	cuts := h.Cut[:0]
	for _, c := range h.Cut {
		if cutPath[c.Path] {
			cuts = append(cuts, *byPath[c.Path])
		}
	}
	h.Cut = cuts
	// Strings of dropped entries are gone; collect what is left.
	strs = strs[:0]
	root.walk(func(n *node) {
		if n.kind == 's' {
			strs = append(strs, n)
		}
	})
	for root.size > max && len(strs) > 0 {
		bi := 0
		for i, s := range strs {
			if len(s.raw) > len(strs[bi].raw) {
				bi = i
			}
		}
		best := strs[bi]
		if len(best.raw) <= minString {
			break
		}
		if shorten(best, root.size-max) {
			h.Shortened++
		} else {
			strs = append(strs[:bi], strs[bi+1:]...)
		}
	}
	if len(h.Cut) == 0 && h.Shortened == 0 {
		return b, nil
	}
	var out bytes.Buffer
	root.write(&out)
	h.Truncated = true
	return out.Bytes(), h
}

// isDetail reports whether the list at path (e.g. days[].categories[].refs)
// is named in detail.
func isDetail(path string, detail []string) bool {
	name := path
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return slices.Contains(detail, strings.TrimSuffix(name, "[]"))
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// shorten cuts the text of s by about need bytes (at least down to minString
// bytes of text) at a rune boundary and marks it with an ellipsis. It reports
// whether s changed.
func shorten(s *node, need int) bool {
	var text string
	if json.Unmarshal(s.raw, &text) != nil {
		return false
	}
	target := len(text) - need - len("…")
	if target < minString {
		target = minString
	}
	if target >= len(text) {
		return false
	}
	for target > 0 && !utf8.RuneStart(text[target]) {
		target--
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(text[:target] + "…")
	nb := bytes.TrimSpace(buf.Bytes())
	if len(nb) >= len(s.raw) {
		return false
	}
	s.grow(len(nb) - len(s.raw))
	s.raw = nb
	return true
}
