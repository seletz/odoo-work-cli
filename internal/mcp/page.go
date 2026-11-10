package mcp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// doc is a JSON result split at its paged list: the natural unit of the
// result (the days of stats, the events of show, the rows of a top-level
// array). Everything else, the totals, stays whole and goes on every page.
type doc struct {
	keys  [][]byte          // raw keys of an object, with quotes
	vals  [][]byte          // raw values; vals[at] is the paged list
	at    int               // index of the paged list; -1 for a top-level array
	unit  string            // name of the paged list; "" for a top-level array
	units []json.RawMessage // raw entries of the paged list
	hash  string            // identifies everything but the entries (see cursor)
}

// splitDoc finds the paged list of the compact document b: the first of
// candidates that is a list in the top-level object ("" is a top-level
// array). ok is false when there is none.
func splitDoc(b []byte, candidates []string) (d *doc, ok bool) {
	if len(b) == 0 {
		return nil, false
	}
	if b[0] == '[' {
		for _, c := range candidates {
			if c == "" {
				d := &doc{at: -1}
				if json.Unmarshal(b, &d.units) != nil {
					return nil, false
				}
				d.hash = digest(b)
				return d, true
			}
		}
		return nil, false
	}
	if b[0] != '{' {
		return nil, false
	}
	d = &doc{at: -1}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		return nil, false
	}
	names := map[string]int{}
	for dec.More() {
		start := int(dec.InputOffset())
		if b[start] == ',' {
			start++
		}
		t, err := dec.Token()
		if err != nil {
			return nil, false
		}
		keyEnd := int(dec.InputOffset())
		var v json.RawMessage
		if dec.Decode(&v) != nil {
			return nil, false
		}
		names[t.(string)] = len(d.keys)
		d.keys = append(d.keys, b[start:keyEnd])
		d.vals = append(d.vals, v)
	}
	for _, c := range candidates {
		i, found := names[c]
		if c != "" && found && d.vals[i][0] == '[' {
			d.at, d.unit = i, c
			if json.Unmarshal(d.vals[i], &d.units) != nil {
				return nil, false
			}
			break
		}
	}
	if d.at < 0 {
		return nil, false
	}
	h := sha256.New()
	for i := range d.keys {
		if i != d.at {
			h.Write(d.keys[i])
			h.Write(d.vals[i])
		}
	}
	d.hash = hex.EncodeToString(h.Sum(nil))[:12]
	return d, true
}

func digest(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])[:12]
}

// page serializes the entries [from, to) of the paged list with all the
// rest of the document, in the original key order.
func (d *doc) page(from, to int) []byte {
	var buf bytes.Buffer
	list := func() {
		buf.WriteByte('[')
		for i := from; i < to; i++ {
			if i > from {
				buf.WriteByte(',')
			}
			buf.Write(d.units[i])
		}
		buf.WriteByte(']')
	}
	if d.at < 0 {
		list()
		return buf.Bytes()
	}
	buf.WriteByte('{')
	for i := range d.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(d.keys[i])
		buf.WriteByte(':')
		if i == d.at {
			list()
		} else {
			buf.Write(d.vals[i])
		}
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

// starts returns the first entry of every page: whole entries, as many as
// fit into max bytes together with the rest of the document, at least one.
// Starting from the same document it is always the same partition, which
// is what makes a cursor stateless.
func (d *doc) starts(max int) []int {
	base := len(d.page(0, 0))
	starts := []int{0}
	size := base
	n := 0
	for i, u := range d.units {
		add := len(u)
		if n > 0 {
			add++
		}
		if n > 0 && size+add > max {
			starts = append(starts, i)
			size, n, add = base, 0, len(u)
		}
		size += add
		n++
	}
	return starts
}

// cursorOf is the opaque token for the page starting at entry from of the
// query with the argument hash query: "from-query-hash".
func (d *doc) cursorOf(from int, query string) string {
	return strconv.Itoa(from) + "-" + query + "-" + d.hash
}

// splitCursor takes a cursor apart: the first entry, the hash of the
// arguments of the call that issued it, and the hash of the result.
func splitCursor(c string) (from int, query, hash string, err error) {
	parts := strings.Split(c, "-")
	if len(parts) == 3 {
		from, err = strconv.Atoi(parts[0])
		if err == nil && from >= 0 && parts[1] != "" && parts[2] != "" {
			return from, parts[1], parts[2], nil
		}
	}
	return 0, "", "", fmt.Errorf("invalid cursor %q: pass next.cursor of the previous page unchanged", c)
}

// checkCursor validates the cursor of a call with the argument hash query
// and returns the first entry it names. It tells apart a cursor that is
// broken, one of a different query (other arguments) and, for the same
// query, a result that changed since the cursor was issued. d may be nil
// when the result has no pages (only the first two are checked).
func checkCursor(c, query string, d *doc) (int, error) {
	from, q, hash, err := splitCursor(c)
	if err != nil {
		return 0, err
	}
	if q != query {
		return 0, errDifferentQuery
	}
	if d != nil && hash != d.hash {
		return 0, errChanged
	}
	return from, nil
}

var (
	errDifferentQuery = fmt.Errorf("cursor belongs to a different query: it only works with the arguments of the call that returned it; leave out cursor to start this query")
	errChanged        = fmt.Errorf("the result changed since this cursor was issued (entries or records changed in Odoo): start again without cursor")
)

// errNoPages is returned for a cursor on a result that has no pages.
var errNoPages = fmt.Errorf("this result has no pages: leave out cursor")

// paginate returns the page of the compact JSON document b that the cursor
// names (the first page for ""). Every page holds whole entries of the
// paged list plus everything else unchanged, so the totals are on every
// page and all pages together are exactly b. A page that is larger than
// max although it holds one entry only loses detail lists (t.Detail) and
// long texts, flagged in the hint, never sums; if that is not enough it is
// an error. ok is false when b has no paged list.
func paginate(b []byte, t Tool, cursor, query string, max int) (page []byte, hint *Hint, ok bool, err error) {
	d, ok := splitDoc(b, t.Units)
	if !ok {
		return nil, nil, false, nil
	}
	starts := d.starts(max)
	idx := 0
	if cursor != "" {
		from, err := checkCursor(cursor, query, d)
		if err != nil {
			return nil, nil, true, err
		}
		idx = -1
		for i, s := range starts {
			if s == from {
				idx = i
			}
		}
		if idx < 0 {
			return nil, nil, true, fmt.Errorf("invalid cursor %q: no page starts there; pass next.cursor of the previous page unchanged", cursor)
		}
	}
	from, to := starts[idx], len(d.units)
	if idx+1 < len(starts) {
		to = starts[idx+1]
	}
	page = d.page(from, to)
	hint = &Hint{}
	var text []string
	if len(starts) > 1 {
		hint.Page = &PageInfo{Index: idx + 1, Of: len(starts), Unit: d.unit, From: from, To: to, Total: len(d.units)}
		what := "entries"
		if d.unit != "" {
			what = d.unit
		}
		text = append(text, fmt.Sprintf("page %d of %d (%s %d to %d of %d); totals and all other fields are complete on every page", idx+1, len(starts), what, from+1, to, len(d.units)))
		if idx+1 < len(starts) {
			hint.Next = &Next{Cursor: d.cursorOf(starts[idx+1], query)}
			text = append(text, fmt.Sprintf("call again with cursor=%q for the next page", hint.Next.Cursor))
		} else {
			text = append(text, "last page")
		}
	}
	if len(page) > max {
		cut, h, err := cutToFit(page, max, t.Detail, t.Narrow)
		if err != nil {
			return nil, nil, true, err
		}
		page = cut
		if h != nil {
			hint.Truncated, hint.Cut, hint.Shortened = true, h.Cut, h.Shortened
			text = append(text, h.Hint)
		}
	}
	if len(text) == 0 {
		return page, nil, true, nil
	}
	hint.Hint = strings.Join(text, "; ")
	if !hint.Truncated && t.Narrow != "" {
		hint.Hint += "; or narrow the query first: " + t.Narrow
	}
	return page, hint, true, nil
}
