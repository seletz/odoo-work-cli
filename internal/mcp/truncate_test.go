package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// statsDoc builds a stats-like object: days with categories, each with a
// list of sessions.
func statsDoc(days, cats, sessions int) string {
	var d []string
	for i := 0; i < days; i++ {
		var c []string
		for j := 0; j < cats; j++ {
			var s []string
			for k := 0; k < sessions; k++ {
				s = append(s, fmt.Sprintf(`{"id":"claude:%08d-aaaa-bbbb","title":"session %d title","minutes":%d}`, k, k, k+1))
			}
			c = append(c, fmt.Sprintf(`{"category":"c%d","minutes":%d,"sessions":[%s]}`, j, j*10, strings.Join(s, ",")))
		}
		d = append(d, fmt.Sprintf(`{"day":"2026-10-%02d","minutes":480,"categories":[%s]}`, i+1, strings.Join(c, ",")))
	}
	return `{"worktime":{"from":"2026-10-01","days":[` + strings.Join(d, ",") + `],"total_minutes":9999}}`
}

func TestTruncateObjectNestedLists(t *testing.T) {
	doc := statsDoc(20, 4, 30)
	const max = 6000
	if len(doc) < 4*max {
		t.Fatalf("fixture too small: %d", len(doc))
	}
	got, h, err := truncate(doc, max, []string{"sessions"}, "sessions:false or a smaller range")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > max {
		t.Errorf("got %d bytes, max %d", len(got), max)
	}
	var v struct {
		Worktime struct {
			From string
			Days []struct {
				Day        string
				Categories []struct {
					Category string
					Sessions []json.RawMessage
				}
			}
			Total int `json:"total_minutes"`
		}
	}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, got)
	}
	if len(v.Worktime.Days) != 20 || v.Worktime.Total != 9999 {
		t.Errorf("structure lost: %d days, total %d (nested lists go first)", len(v.Worktime.Days), v.Worktime.Total)
	}
	if !strings.HasPrefix(got, `{"worktime":{"from"`) {
		t.Errorf("key order lost: %.40s", got)
	}
	if h == nil || !h.Truncated || h.Next != nil || len(h.Cut) == 0 {
		t.Fatalf("hint = %+v", h)
	}
	c := h.Cut[0]
	if c.Path != "worktime.days[].categories[].sessions" || c.Total != 20*4*30 || c.Kept >= c.Total || c.Lists != 80 {
		t.Errorf("cut = %+v", c)
	}
	if !strings.Contains(h.Hint, "sessions:false or a smaller range") || !strings.Contains(h.Hint, c.Path) {
		t.Errorf("hint text = %q", h.Hint)
	}
}

func TestTruncateObjectIndentedFitsCompact(t *testing.T) {
	// Pretty-printed output over the bound that fits once compacted: no cut.
	var buf strings.Builder
	var v any
	_ = json.Unmarshal([]byte(statsDoc(2, 2, 3)), &v)
	b, _ := json.MarshalIndent(v, "", "    ")
	buf.Write(b)
	got, h, _ := truncate(buf.String(), len(b)/2, nil, "")
	if h != nil || len(got) > len(b)/2 || !json.Valid([]byte(got)) {
		t.Errorf("hint %+v, %d bytes, valid %v", h, len(got), json.Valid([]byte(got)))
	}
}

func TestTruncateObjectShortensStrings(t *testing.T) {
	doc := `{"content":"` + strings.Repeat("ä", 5000) + `","n":1}`
	got, h, _ := truncate(doc, 1000, nil, "")
	if len(got) > 1000 || !json.Valid([]byte(got)) {
		t.Fatalf("%d bytes, valid %v", len(got), json.Valid([]byte(got)))
	}
	if h == nil || h.Shortened != 1 || !strings.Contains(h.Hint, "shortened") {
		t.Errorf("hint = %+v", h)
	}
}

func TestTruncateObjectNothingToCut(t *testing.T) {
	// Numbers that cannot be cut: an error instead of a wrong partial result.
	var parts []string
	for i := 0; i < 400; i++ {
		parts = append(parts, fmt.Sprintf(`"k%d":%d`, i, i))
	}
	doc := "{" + strings.Join(parts, ",") + "}"
	if _, _, err := truncate(doc, 100, nil, ""); err == nil || !strings.Contains(err.Error(), "narrow") {
		t.Errorf("err = %v, want a narrow-the-query error", err)
	}
}

func TestTruncateNotJSON(t *testing.T) {
	out := strings.Repeat("plain text ", 100)
	got, h, _ := truncate(out, 50, nil, "")
	if got != out || h != nil {
		t.Errorf("non-JSON must pass through unchanged")
	}
}

func TestDefaultBoundOnLargeStats(t *testing.T) {
	doc := statsDoc(31, 6, 40) // about the 160,000 characters of the bug report and more
	if len(doc) < 160000 {
		t.Fatalf("fixture %d bytes", len(doc))
	}
	s := testServer(func(context.Context, []string) (string, bool) { return doc, false })
	s.MaxBytes = 0
	s.Tools[0].Detail = []string{"sessions"}
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"x"}}}`)
	c := m["result"].(map[string]any)["content"].([]any)
	first := c[0].(map[string]any)["text"].(string)
	if len(c) != 2 || len(first) > DefaultMaxBytes || !json.Valid([]byte(first)) {
		t.Errorf("%d items, first %d bytes, valid %v", len(c), len(first), json.Valid([]byte(first)))
	}
}
