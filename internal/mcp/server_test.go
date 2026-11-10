package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func testServer(run Runner) *Server {
	return &Server{
		Name: "odoo-work-cli", Version: "test", Run: run, MaxBytes: 200,
		Tools: []Tool{{
			Name: "search", Description: "d", Command: []string{"search"}, FixedFlags: []string{"--json"},
			Params: []Param{
				{Name: "query", Type: "string", Required: true, Positional: true},
				{Name: "actor", Type: "string", Enum: []string{"human", "agent"}},
				{Name: "limit", Type: "integer", Max: 50},
				{Name: "by_session", Type: "boolean"},
				{Name: "type", Type: "list", Enum: []string{"event", "session"}},
			},
			Defaults: map[string]int{"limit": 20}, ReadOnly: true, Idempotent: true,
		}, {
			Name: "show", ReadOnly: true, Idempotent: true, Command: []string{"show"}, FixedFlags: []string{"--json"}, Units: []string{"events"},
			Params: []Param{{Name: "session", Type: "string", Required: true, Positional: true}, {Name: "offset", Type: "integer"}, {Name: "cursor", Type: "string", Server: true}},
		}},
	}
}

func rpc(t *testing.T, s *Server, msg string) map[string]any {
	t.Helper()
	out := s.Handle(context.Background(), []byte(msg))
	if out == nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("response is not JSON: %v: %s", err, out)
	}
	if m["jsonrpc"] != "2.0" {
		t.Fatalf("jsonrpc = %v", m["jsonrpc"])
	}
	return m
}

func TestInitializeNegotiatesVersion(t *testing.T) {
	s := testServer(nil)
	for _, tc := range []struct{ client, want string }{
		{"2025-06-18", "2025-06-18"},
		{"2024-11-05", "2024-11-05"},
		{"2099-01-01", ProtocolVersions[0]},
		{"", ProtocolVersions[0]},
	} {
		m := rpc(t, s, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":%q,"capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`, tc.client))
		res := m["result"].(map[string]any)
		if res["protocolVersion"] != tc.want {
			t.Errorf("client %q: version %v, want %s", tc.client, res["protocolVersion"], tc.want)
		}
		if _, ok := res["capabilities"].(map[string]any)["tools"]; !ok {
			t.Errorf("no tools capability: %v", res)
		}
		if res["serverInfo"].(map[string]any)["name"] != "odoo-work-cli" {
			t.Errorf("serverInfo = %v", res["serverInfo"])
		}
		if m["id"] != float64(1) {
			t.Errorf("id = %v", m["id"])
		}
	}
}

func TestJSONRPCErrors(t *testing.T) {
	s := testServer(nil)
	for _, tc := range []struct {
		name, msg string
		code      float64
		id        any
	}{
		{"parse", `{not json`, -32700, nil},
		{"batch", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, -32600, nil},
		{"not 2.0", `{"jsonrpc":"1.0","id":7,"method":"ping"}`, -32600, float64(7)},
		{"unknown method", `{"jsonrpc":"2.0","id":"a","method":"resources/list"}`, -32601, "a"},
		{"unknown tool", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"sql"}}`, -32602, float64(2)},
		{"no name", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{}}`, -32602, float64(3)},
	} {
		m := rpc(t, s, tc.msg)
		e, ok := m["error"].(map[string]any)
		if !ok || e["code"] != tc.code {
			t.Errorf("%s: %v, want code %v", tc.name, m, tc.code)
			continue
		}
		if m["id"] != tc.id {
			t.Errorf("%s: id = %v, want %v", tc.name, m["id"], tc.id)
		}
	}
}

func TestNotificationsGetNoResponse(t *testing.T) {
	s := testServer(nil)
	for _, msg := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`,
		`{"jsonrpc":"2.0","method":"unknown/thing"}`,
	} {
		if out := s.Handle(context.Background(), []byte(msg)); out != nil {
			t.Errorf("%s: got response %s", msg, out)
		}
	}
}

func TestPingAndToolsList(t *testing.T) {
	s := testServer(nil)
	if m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"ping"}`); len(m["result"].(map[string]any)) != 0 {
		t.Errorf("ping = %v", m)
	}
	m := rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := m["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %d", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "search" {
		t.Errorf("name = %v", tool["name"])
	}
	if tool["annotations"].(map[string]any)["readOnlyHint"] != true {
		t.Errorf("not read-only: %v", tool["annotations"])
	}
	schema := tool["inputSchema"].(map[string]any)
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Errorf("schema = %v", schema)
	}
	if req := schema["required"].([]any); len(req) != 1 || req[0] != "query" {
		t.Errorf("required = %v", req)
	}
	props := schema["properties"].(map[string]any)
	if props["limit"].(map[string]any)["maximum"] != float64(50) || props["actor"].(map[string]any)["enum"] == nil {
		t.Errorf("properties = %v", props)
	}
}

func TestToolCallArgv(t *testing.T) {
	var got []string
	s := testServer(func(_ context.Context, argv []string) (string, bool) {
		got = argv
		return `[]`, false
	})
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"--help x","actor":"human","by_session":true}}}`)
	res := m["result"].(map[string]any)
	if res["isError"] != false {
		t.Fatalf("isError: %v", res)
	}
	want := "search --actor=human --by-session --limit=20 --json -- --help x"
	if strings.Join(got, " ") != want {
		t.Errorf("argv = %q, want %q", strings.Join(got, " "), want)
	}
	c := res["content"].([]any)
	if len(c) != 1 || c[0].(map[string]any)["text"] != "[]" || c[0].(map[string]any)["type"] != "text" {
		t.Errorf("content = %v", c)
	}
}

func TestListParam(t *testing.T) {
	var got []string
	s := testServer(func(_ context.Context, argv []string) (string, bool) { got = argv; return "[]", false })
	for args, want := range map[string]string{
		`{"query":"q","type":["event","session"]}`: "--type=event,session",
		`{"query":"q","type":"session, event"}`:    "--type=session,event",
	} {
		rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":`+args+`}}`)
		if !strings.Contains(strings.Join(got, " "), want) {
			t.Errorf("%s: argv %v lacks %s", args, got, want)
		}
	}
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"q","type":["memory"]}}}`)
	if m["result"].(map[string]any)["isError"] != true {
		t.Errorf("enum not checked: %v", m)
	}
	list := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)["result"].(map[string]any)["tools"].([]any)
	typ := list[0].(map[string]any)["inputSchema"].(map[string]any)["properties"].(map[string]any)["type"].(map[string]any)
	if typ["anyOf"] == nil || typ["type"] != nil {
		t.Errorf("schema of a list = %v", typ)
	}
}

func TestToolCallValidation(t *testing.T) {
	ran := false
	s := testServer(func(context.Context, []string) (string, bool) { ran = true; return "", false })
	for name, args := range map[string]string{
		"missing":  `{}`,
		"unknown":  `{"query":"x","sql":"drop"}`,
		"enum":     `{"query":"x","actor":"robot"}`,
		"max":      `{"query":"x","limit":51}`,
		"negative": `{"query":"x","limit":-1}`,
		"type":     `{"query":5}`,
		"bool":     `{"query":"x","by_session":"yes"}`,
		"notobj":   `[1]`,
	} {
		m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":`+args+`}}`)
		res, ok := m["result"].(map[string]any)
		if !ok || res["isError"] != true {
			t.Errorf("%s: %v", name, m)
			continue
		}
		var e struct {
			Error struct{ Kind, Message string }
		}
		if err := json.Unmarshal([]byte(res["content"].([]any)[0].(map[string]any)["text"].(string)), &e); err != nil || e.Error.Kind != "validation" || e.Error.Message == "" {
			t.Errorf("%s: error object = %v", name, res["content"])
		}
	}
	if ran {
		t.Error("runner ran on invalid arguments")
	}
}

func TestToolCallFailureKeepsCLIError(t *testing.T) {
	s := testServer(func(context.Context, []string) (string, bool) {
		return `{"error":{"kind":"db","message":"no archive"}}` + "\n", true
	})
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"show","arguments":{"session":"x"}}}`)
	res := m["result"].(map[string]any)
	if res["isError"] != true || res["content"].([]any)[0].(map[string]any)["text"] != `{"error":{"kind":"db","message":"no archive"}}` {
		t.Errorf("result = %v", res)
	}
}

func TestSmallResultIsVerbatim(t *testing.T) {
	s := testServer(func(context.Context, []string) (string, bool) { return "[\n  {\"a\": 1}\n]\n", false })
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"x"}}}`)
	c := m["result"].(map[string]any)["content"].([]any)
	if len(c) != 1 || c[0].(map[string]any)["text"] != "[\n  {\"a\": 1}\n]" {
		t.Errorf("content = %v", c)
	}
}

func TestServe(t *testing.T) {
	s := testServer(nil)
	in := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"ping"}` // no trailing newline
	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"id":1`) || !strings.Contains(lines[1], `"id":2`) {
		t.Errorf("output = %q", out.String())
	}
}

// TestUsageErrorNamesArguments: a validation error of the CLI names its flags
// (--by-session); the MCP caller knows arguments (by_session), so the
// server says argument "by_session". Only flags of the tool are renamed,
// other kinds of errors are kept as they are.
func TestUsageErrorNamesArguments(t *testing.T) {
	for _, tt := range []struct {
		name, out, want string
	}{
		{"flag of the tool", `{"error":{"kind":"validation","message":"--actor does not apply to type session"}}`,
			`{"error":{"kind":"validation","message":"argument \"actor\" does not apply to type session"}}`},
		{"hyphen becomes underscore, several flags", `{"error":{"kind":"validation","message":"--limit (3) must be below --by-session"}}`,
			`{"error":{"kind":"validation","message":"argument \"limit\" (3) must be below argument \"by_session\""}}`},
		{"longer flag name is not cut", `{"error":{"kind":"validation","message":"--limits is unknown, --limit is fine"}}`,
			`{"error":{"kind":"validation","message":"--limits is unknown, argument \"limit\" is fine"}}`},
		{"flag the tool has no argument for", `{"error":{"kind":"validation","message":"--db is wrong"}}`,
			`{"error":{"kind":"validation","message":"--db is wrong"}}`},
		{"other kinds stay", `{"error":{"kind":"sql","message":"--actor in the SQL"}}`,
			`{"error":{"kind":"sql","message":"--actor in the SQL"}}`},
		{"not an error object", "plain text --actor", "plain text --actor"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(func(context.Context, []string) (string, bool) { return tt.out + "\n", true })
			m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"x"}}}`)
			got := m["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"]
			if got != tt.want {
				t.Errorf("text = %v\nwant   %v", got, tt.want)
			}
		})
	}
}

// writeServer has one read tool and three write tools with different
// annotations, a positional integer and a number-or-string argument.
func writeServer(run Runner) *Server {
	return &Server{
		Name: "odoo-work-cli", Version: "test", Run: run,
		Tools: []Tool{
			{Name: "projects", Command: []string{"projects"}, FixedFlags: []string{"--json"}, ReadOnly: true, Idempotent: true},
			{Name: "entries_add", Command: []string{"entries", "add"}, FixedFlags: []string{"--json"},
				Params: []Param{
					{Name: "project_id", Type: "integer", Required: true},
					{Name: "hours", Type: "string", Number: true, Required: true},
					{Name: "dry_run", Type: "boolean"},
				}},
			{Name: "entries_update", Command: []string{"entries", "update"}, FixedFlags: []string{"--json"}, Destructive: true, Idempotent: true,
				Params: []Param{{Name: "id", Type: "integer", Required: true, Positional: true}}},
			{Name: "entries_delete", Command: []string{"entries", "delete"}, FixedFlags: []string{"--json"}, Destructive: true, Idempotent: true,
				Params: []Param{{Name: "id", Type: "integer", Required: true, Positional: true}}},
		},
	}
}

// TestAnnotations: read tools are read-only and idempotent; write tools
// carry their own destructive and idempotent hints.
func TestAnnotations(t *testing.T) {
	s := writeServer(nil)
	tools := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)["result"].(map[string]any)["tools"].([]any)
	want := map[string][3]bool{ // readOnly, destructive, idempotent
		"projects":       {true, false, true},
		"entries_add":    {false, false, false},
		"entries_update": {false, true, true},
		"entries_delete": {false, true, true},
	}
	for _, x := range tools {
		tool := x.(map[string]any)
		a := tool["annotations"].(map[string]any)
		w := want[tool["name"].(string)]
		got := [3]bool{a["readOnlyHint"].(bool), a["destructiveHint"].(bool), a["idempotentHint"].(bool)}
		if got != w {
			t.Errorf("%s: annotations %v, want %v", tool["name"], got, w)
		}
		if a["openWorldHint"] != false {
			t.Errorf("%s: openWorldHint = %v", tool["name"], a["openWorldHint"])
		}
	}
}

// TestPositionalIntegerAndNumber: an integer can be positional, and a
// Number string argument accepts a JSON number as well (hours: 2.5).
func TestPositionalIntegerAndNumber(t *testing.T) {
	var got []string
	s := writeServer(func(_ context.Context, argv []string) (string, bool) { got = argv; return "{}", false })
	for args, want := range map[string]string{
		`{"name":"entries_update","arguments":{"id":42}}`:                                "entries update --json -- 42",
		`{"name":"entries_add","arguments":{"project_id":3,"hours":2.5,"dry_run":true}}`: "entries add --project-id=3 --hours=2.5 --dry-run --json",
		`{"name":"entries_add","arguments":{"project_id":3,"hours":"2:30"}}`:             "entries add --project-id=3 --hours=2:30 --json",
	} {
		m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":`+args+`}`)
		if m["result"].(map[string]any)["isError"] != false {
			t.Errorf("%s: %v", args, m)
		}
		if strings.Join(got, " ") != want {
			t.Errorf("%s: argv %q, want %q", args, strings.Join(got, " "), want)
		}
	}
	m := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"entries_add","arguments":{"project_id":3,"hours":true}}}`)
	if m["result"].(map[string]any)["isError"] != true {
		t.Errorf("hours=true accepted: %v", m)
	}
	tools := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)["result"].(map[string]any)["tools"].([]any)
	hours := tools[1].(map[string]any)["inputSchema"].(map[string]any)["properties"].(map[string]any)["hours"].(map[string]any)
	if fmt.Sprint(hours["type"]) != "[string number]" {
		t.Errorf("hours type = %v", hours["type"])
	}
}
