package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/mcp"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/odoo/odootest"
)

// mcpSession drives an MCP server in-process, one JSON-RPC message at a time.
type mcpSession struct {
	t  *testing.T
	s  *mcp.Server
	id int
}

// stagesOf adapts a one-piece setup for tests that never reach it.
func stagesOf(setup setupFunc) mcpSetup {
	var client odoo.Client
	return mcpSetup{
		loadConfig: func(f string) (*config.Config, error) {
			c, cl, err := setup(f)
			client = cl
			return c, err
		},
		newClient: func(*config.Config) (odoo.Client, error) { return client, nil },
	}
}

func newSession(t *testing.T, global []string, setup setupFunc, readOnly bool) *mcpSession {
	t.Helper()
	s, closeFn := newMCPServer(global, stagesOf(setup), readOnly, false)
	t.Cleanup(closeFn)
	return &mcpSession{t: t, s: s}
}

func (s *mcpSession) rpc(method, params string) map[string]any {
	s.t.Helper()
	s.id++
	msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q`, s.id, method)
	if params != "" {
		msg += `,"params":` + params
	}
	out := s.s.Handle(context.Background(), []byte(msg+"}"))
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		s.t.Fatalf("%s: response is not JSON: %v: %s", method, err, out)
	}
	return m
}

// call calls tool with args and returns the text of the first content item.
func (s *mcpSession) call(tool string, args any) (text string, isError bool) {
	s.t.Helper()
	b, _ := json.Marshal(map[string]any{"name": tool, "arguments": args})
	m := s.rpc("tools/call", string(b))
	res, ok := m["result"].(map[string]any)
	if !ok {
		s.t.Fatalf("%s: no result: %v", tool, m)
	}
	c := res["content"].([]any)
	return c[0].(map[string]any)["text"].(string), res["isError"] == true
}

// mustCall calls tool, requires success and decodes the JSON into out.
func (s *mcpSession) mustCall(tool string, args any, out any) {
	s.t.Helper()
	text, isErr := s.call(tool, args)
	if isErr {
		s.t.Fatalf("%s(%v): error %s", tool, args, text)
	}
	if out != nil {
		if err := json.Unmarshal([]byte(text), out); err != nil {
			s.t.Fatalf("%s: decoding %q: %v", tool, text, err)
		}
	}
}

func fakeSession(t *testing.T, readOnly bool) *mcpSession {
	t.Helper()
	useFakeBackend(t)
	return newSession(t, nil, noSetup(t), readOnly)
}

var (
	readTools  = []string{"whoami", "projects", "tasks", "entries", "timesheets", "clock_status", "clock_list"}
	writeTools = []string{"entries_add", "entries_update", "entries_delete", "clock_in", "clock_out", "clock_add", "clock_edit"}
)

func toolsByName(t *testing.T, s *mcpSession) map[string]map[string]any {
	t.Helper()
	tools := s.rpc("tools/list", "")["result"].(map[string]any)["tools"].([]any)
	byName := map[string]map[string]any{}
	for _, x := range tools {
		tool := x.(map[string]any)
		byName[tool["name"].(string)] = tool
	}
	return byName
}

func TestMCPHandshakeAndTools(t *testing.T) {
	s := fakeSession(t, false)
	res := s.rpc("initialize", `{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}`)["result"].(map[string]any)
	if res["serverInfo"].(map[string]any)["name"] != "odoo-work-cli" || res["protocolVersion"] != "2025-06-18" {
		t.Errorf("initialize = %v", res)
	}
	if !strings.Contains(res["instructions"].(string), "dry_run") {
		t.Errorf("instructions do not mention dry_run: %v", res["instructions"])
	}

	byName := toolsByName(t, s)
	if len(byName) != len(readTools)+len(writeTools) {
		t.Errorf("tools = %v", slices.Sorted(func(yield func(string) bool) {
			for k := range byName {
				if !yield(k) {
					return
				}
			}
		}))
	}
	for _, name := range readTools {
		a := byName[name]["annotations"].(map[string]any)
		if a["readOnlyHint"] != true || a["destructiveHint"] != false || a["idempotentHint"] != true {
			t.Errorf("%s: annotations %v", name, a)
		}
	}
	destructive := map[string]bool{"entries_update": true, "entries_delete": true, "clock_edit": true}
	for _, name := range writeTools {
		tool, ok := byName[name]
		if !ok {
			t.Errorf("missing write tool %s", name)
			continue
		}
		a := tool["annotations"].(map[string]any)
		if a["readOnlyHint"] != false || a["destructiveHint"] != destructive[name] {
			t.Errorf("%s: annotations %v", name, a)
		}
		if tool["description"] == "" || tool["inputSchema"] == nil {
			t.Errorf("%s: no description or schema", name)
		}
	}
}

func TestMCPReadOnlyServesNoWriteTools(t *testing.T) {
	s := fakeSession(t, true)
	byName := toolsByName(t, s)
	for _, name := range writeTools {
		if _, ok := byName[name]; ok {
			t.Errorf("read-only server lists %s", name)
		}
	}
	for _, name := range readTools {
		if _, ok := byName[name]; !ok {
			t.Errorf("read-only server lacks %s", name)
		}
	}
	m := s.rpc("tools/call", `{"name":"entries_delete","arguments":{"id":1001}}`)
	if m["error"] == nil {
		t.Errorf("read-only server ran entries_delete: %v", m)
	}
}

type mcpEntry struct {
	ID          *int64 `json:"id"`
	Date        string `json:"date"`
	ProjectID   int64  `json:"project_id"`
	Description string `json:"description"`
	Hours       float64
	DryRun      bool `json:"dry_run"`
}

// TestMCPFlow: the acceptance flow of #102 against the fake backend. It
// also proves that a dry run does not leak into the next call.
func TestMCPFlow(t *testing.T) {
	s := fakeSession(t, false)
	const date = "2026-01-07"

	var projects []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	s.mustCall("projects", map[string]any{"search": "beta"}, &projects)
	if len(projects) != 1 {
		t.Fatalf("projects search beta = %v", projects)
	}
	add := map[string]any{"project_id": projects[0].ID, "date": date, "hours": 1.5, "description": "mcp flow"}

	var dry mcpEntry
	s.mustCall("entries_add", withArg(add, "dry_run", true), &dry)
	if !dry.DryRun || dry.ID != nil {
		t.Fatalf("dry run = %+v", dry)
	}
	var before []mcpEntry
	s.mustCall("entries", map[string]any{"date": date}, &before)
	if len(before) != 0 {
		t.Fatalf("dry run wrote an entry: %+v", before)
	}

	var created mcpEntry
	s.mustCall("entries_add", add, &created)
	if created.ID == nil || created.DryRun || created.Hours != 1.5 {
		t.Fatalf("created = %+v", created)
	}
	var after []mcpEntry
	s.mustCall("entries", map[string]any{"date": date}, &after)
	if len(after) != 1 || *after[0].ID != *created.ID || after[0].Description != "mcp flow" {
		t.Fatalf("entries after add = %+v", after)
	}
}

func withArg(m map[string]any, k string, v any) map[string]any {
	c := map[string]any{k: v}
	for kk, vv := range m {
		c[kk] = vv
	}
	return c
}

// TestMCPFlagsDoNotLeak: every call gets a fresh command tree, so a flag of
// one call is not set in the next.
func TestMCPFlagsDoNotLeak(t *testing.T) {
	s := fakeSession(t, false)
	var all, alpha, again []json.RawMessage
	s.mustCall("projects", nil, &all)
	s.mustCall("projects", map[string]any{"search": "alpha"}, &alpha)
	s.mustCall("projects", map[string]any{}, &again)
	if len(alpha) != 1 || len(all) < 2 || len(again) != len(all) {
		t.Errorf("all %d, alpha %d, again %d", len(all), len(alpha), len(again))
	}

	var list1, list2 []json.RawMessage
	s.mustCall("clock_list", map[string]any{"week": "2026-W02"}, &list1)
	s.mustCall("clock_add", map[string]any{"date": "2026-01-05", "in": "08:00", "out": "12:00"}, nil)
	s.mustCall("clock_list", map[string]any{"date": "2026-01-05"}, &list2)
	if len(list1) != 0 || len(list2) != 1 {
		t.Errorf("week list %d (want 0), day list %d (want 1)", len(list1), len(list2))
	}
}

// TestMCPEveryTool calls every tool once against the fake backend.
func TestMCPEveryTool(t *testing.T) {
	s := fakeSession(t, false)
	var created struct {
		ID int64 `json:"id"`
	}
	s.mustCall("entries_add", map[string]any{"project_id": 1, "task_id": 11, "date": "2026-01-08", "hours": "2:30", "description": "every tool"}, &created)

	calls := []struct {
		tool string
		args map[string]any
	}{
		{"whoami", nil},
		{"projects", nil},
		{"tasks", map[string]any{"project_id": 1}},
		{"tasks", map[string]any{"search": "support"}},
		{"entries", map[string]any{"week": "2026-W02", "project": "alpha"}},
		{"timesheets", map[string]any{"week": "2026-W02"}},
		{"entries_update", map[string]any{"id": created.ID, "hours": 3, "dry_run": true}},
		{"entries_update", map[string]any{"id": created.ID, "description": "updated"}},
		{"clock_status", nil},
		{"clock_in", nil},
		{"clock_out", nil},
		{"clock_add", map[string]any{"date": "2026-01-09", "in": "08:00", "out": "12:00"}},
		{"clock_edit", map[string]any{"date": "2026-01-12", "in": "09:00", "out": "17:00", "create_only": true}},
		{"clock_list", map[string]any{"week": "2026-W03"}},
		{"entries_delete", map[string]any{"id": created.ID}},
	}
	for _, c := range calls {
		text, isErr := s.call(c.tool, c.args)
		if isErr {
			t.Errorf("%s(%v): %s", c.tool, c.args, text)
			continue
		}
		if !json.Valid([]byte(text)) {
			t.Errorf("%s: not JSON: %s", c.tool, text)
		}
	}
}

func errKind(t *testing.T, text string) string {
	t.Helper()
	var e errEnvelope
	if err := json.Unmarshal([]byte(text), &e); err != nil {
		t.Fatalf("not an error envelope: %s", text)
	}
	return e.Error.Kind
}

func TestMCPErrorEnvelope(t *testing.T) {
	s := fakeSession(t, false)
	for _, tc := range []struct {
		tool string
		args map[string]any
		kind string
		msg  string
	}{
		{"entries_delete", map[string]any{"id": 424242}, "not_found", ""},
		{"entries_add", map[string]any{"project_id": 1, "hours": 1}, "validation", `missing argument "description"`},
		{"entries_add", map[string]any{"project_id": 1, "hours": "x", "description": "d"}, "validation", ""},
		{"entries", map[string]any{"week": "nope"}, "validation", ""},
		{"clock_out", nil, "conflict", ""},
		{"projects", map[string]any{"bogus": 1}, "validation", "unknown argument"},
		{"clock_list", map[string]any{"date": "2026-01-05", "week": "2026-W02"}, "validation", ""},
	} {
		text, isErr := s.call(tc.tool, tc.args)
		if !isErr {
			t.Errorf("%s(%v): no error: %s", tc.tool, tc.args, text)
			continue
		}
		if k := errKind(t, text); k != tc.kind {
			t.Errorf("%s(%v): kind %s, want %s: %s", tc.tool, tc.args, k, tc.kind, text)
		}
		var e struct{ Error struct{ Message string } }
		_ = json.Unmarshal([]byte(text), &e)
		if !strings.Contains(e.Error.Message, tc.msg) {
			t.Errorf("%s: message %q lacks %q", tc.tool, e.Error.Message, tc.msg)
		}
		if strings.Contains(text, "--") {
			t.Errorf("%s: message names a flag, not an argument: %s", tc.tool, text)
		}
	}
}

// stageProbe counts the two setup stages and lets a test fail them.
type stageProbe struct {
	cfgCalls, clientCalls int
	locked                bool  // config stage fails with secrets_locked
	clientErr             error // client stage fails with this error
	client                *odootest.Fake
}

func (p *stageProbe) stages() mcpSetup {
	return mcpSetup{
		loadConfig: func(string) (*config.Config, error) {
			p.cfgCalls++
			if p.locked {
				return nil, apperr.New(apperr.KindSecretsLocked, "1Password is locked")
			}
			cfg := &config.Config{}
			cfg.ApplyDefaults()
			return cfg, nil
		},
		newClient: func(*config.Config) (odoo.Client, error) {
			p.clientCalls++
			if p.clientErr != nil {
				return nil, p.clientErr
			}
			return p.client, nil
		},
	}
}

func newProbe() *stageProbe {
	return &stageProbe{client: &odootest.Fake{User: odoo.UserInfo{ID: 7, Name: "Pat"}}}
}

// TestMCPLazySecrets: the server starts and lists its tools without loading
// config or secrets. A locked 1Password is reported per call as
// secrets_locked and retried; once it is unlocked the next call loads, and
// later calls reuse the loaded client (one load for the life of the server).
func TestMCPLazySecrets(t *testing.T) {
	t.Setenv("ODOO_WORK_CLI_BACKEND", "")
	p := newProbe()
	p.locked = true
	s, closeFn := newMCPServer(nil, p.stages(), false, false)
	sess := &mcpSession{t: t, s: s}
	sess.rpc("initialize", `{"protocolVersion":"2025-06-18"}`)
	sess.rpc("tools/list", "")
	if p.cfgCalls != 0 {
		t.Fatalf("config loaded %d times before the first tool call", p.cfgCalls)
	}

	text, isErr := sess.call("whoami", nil)
	if !isErr || errKind(t, text) != "secrets_locked" {
		t.Fatalf("locked: %v %s", isErr, text)
	}

	p.locked = false
	sess.mustCall("whoami", nil, nil)
	sess.mustCall("whoami", nil, nil)
	if p.cfgCalls != 2 {
		t.Errorf("config loaded %d times, want 2 (locked, then once)", p.cfgCalls)
	}
	if p.client.Closed {
		t.Error("client closed between calls")
	}
	closeFn()
	if !p.client.Closed {
		t.Error("client not closed when the server ends")
	}
}

// TestMCPKeepsSecretsWhenClientFails: op succeeded, the Odoo login failed
// (network): the resolved config is kept, so the retries do not call op
// again, only the client stage.
func TestMCPKeepsSecretsWhenClientFails(t *testing.T) {
	t.Setenv("ODOO_WORK_CLI_BACKEND", "")
	p := newProbe()
	p.clientErr = apperr.New(apperr.KindNetwork, "timeout")
	sess := &mcpSession{t: t}
	s, closeFn := newMCPServer(nil, p.stages(), false, false)
	t.Cleanup(closeFn)
	sess.s = s
	for range 2 {
		text, isErr := sess.call("whoami", nil)
		if !isErr || errKind(t, text) != "network" {
			t.Fatalf("want network error: %v %s", isErr, text)
		}
	}
	p.clientErr = nil
	sess.mustCall("whoami", nil, nil)
	sess.mustCall("whoami", nil, nil)
	if p.cfgCalls != 1 {
		t.Errorf("config (op) loaded %d times, want 1", p.cfgCalls)
	}
	if p.clientCalls != 3 {
		t.Errorf("client stage ran %d times, want 3 (2 failures, 1 success)", p.clientCalls)
	}
}

// TestMCPReResolvesOnAuth: an auth error discards config and client, the
// next call resolves again, once. Other error kinds keep the cache.
func TestMCPReResolvesOnAuth(t *testing.T) {
	t.Setenv("ODOO_WORK_CLI_BACKEND", "")
	p := newProbe()
	s, closeFn := newMCPServer(nil, p.stages(), false, false)
	t.Cleanup(closeFn)
	sess := &mcpSession{t: t, s: s}
	sess.mustCall("whoami", nil, nil)

	p.client.Err = apperr.New(apperr.KindNetwork, "boom")
	if text, isErr := sess.call("whoami", nil); !isErr || errKind(t, text) != "network" {
		t.Fatalf("want network: %s", text)
	}
	if p.cfgCalls != 1 {
		t.Fatalf("network error re-resolved: %d op calls", p.cfgCalls)
	}

	p.client.Err = apperr.New(apperr.KindAuth, "bad key")
	if text, isErr := sess.call("whoami", nil); !isErr || errKind(t, text) != "auth" {
		t.Fatalf("want auth: %s", text)
	}
	if p.cfgCalls != 1 {
		t.Fatalf("auth error re-resolved immediately: %d", p.cfgCalls)
	}
	p.client.Err = nil
	sess.mustCall("whoami", nil, nil)
	sess.mustCall("whoami", nil, nil)
	if p.cfgCalls != 2 {
		t.Errorf("op calls = %d, want 2 (start, one after auth)", p.cfgCalls)
	}
}

// TestMCPEagerSecrets: --secrets eager resolves before the first request;
// a failure does not stop the server and is retried by the first call.
func TestMCPEagerSecrets(t *testing.T) {
	t.Setenv("ODOO_WORK_CLI_BACKEND", "")
	p := newProbe()
	_, closeFn := newMCPServer(nil, p.stages(), false, true)
	t.Cleanup(closeFn)
	if p.cfgCalls != 1 {
		t.Fatalf("eager: %d op calls before the first request, want 1", p.cfgCalls)
	}
	if p.clientCalls != 0 {
		t.Errorf("eager must not connect: %d client calls", p.clientCalls)
	}

	q := newProbe()
	q.locked = true
	s, closeFn2 := newMCPServer(nil, q.stages(), false, true)
	t.Cleanup(closeFn2)
	sess := &mcpSession{t: t, s: s}
	q.locked = false
	sess.mustCall("whoami", nil, nil)
	if q.cfgCalls != 2 {
		t.Errorf("failed eager start, then first call: %d op calls, want 2", q.cfgCalls)
	}
}

// TestMCPEagerSkipsFakeBackend: the fake backend never needs secrets, so
// eager resolves nothing, by flag or by environment.
func TestMCPEagerSkipsFakeBackend(t *testing.T) {
	t.Run("flag", func(t *testing.T) {
		t.Setenv("ODOO_WORK_CLI_BACKEND", "")
		p := newProbe()
		_, closeFn := newMCPServer([]string{"--backend=fake"}, p.stages(), false, true)
		t.Cleanup(closeFn)
		if p.cfgCalls != 0 {
			t.Errorf("--backend fake + eager: %d op calls, want 0", p.cfgCalls)
		}
	})
	t.Run("env", func(t *testing.T) {
		t.Setenv("ODOO_WORK_CLI_BACKEND", "fake")
		p := newProbe()
		_, closeFn := newMCPServer(nil, p.stages(), false, true)
		t.Cleanup(closeFn)
		if p.cfgCalls != 0 {
			t.Errorf("env fake + eager: %d op calls, want 0", p.cfgCalls)
		}
	})
}

func TestParseSecretsMode(t *testing.T) {
	for in, want := range map[string]bool{"eager": true, "lazy": false, "": false, "EAGER": true} {
		got, err := parseSecretsMode(in)
		if err != nil || got != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	if _, err := parseSecretsMode("sometimes"); err == nil || apperr.KindOf(err) != apperr.KindValidation {
		t.Errorf("bad mode: %v", err)
	}
}

// TestMCPCommandOverStdio runs `odoo-work-cli --backend fake mcp` through
// the command tree with a scripted stdin.
func TestMCPCommandOverStdio(t *testing.T) {
	useFakeBackend(t)
	t.Setenv("ODOO_WORK_CLI_BACKEND", "")
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`,
	}, "\n") + "\n"
	root, _ := newRootCmd(noSetup(t))
	var out bytes.Buffer
	root.SetIn(strings.NewReader(in))
	root.SetOut(&out)
	root.SetArgs([]string{"--backend", "fake", "mcp", "--read-only"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("responses = %d: %s", len(lines), out.String())
	}
	if strings.Contains(lines[1], `"entries_add"`) {
		t.Error("--read-only lists entries_add")
	}
	if !strings.Contains(lines[2], `Fake User`) {
		t.Errorf("whoami over --backend fake: %s", lines[2])
	}
}

func TestMCPPrint(t *testing.T) {
	code, out, _ := execute(t, noSetup(t), "mcp", "print", "--binary", "/opt/bin/odoo-work-cli")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var cfg struct {
		MCPServers map[string]struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	e := cfg.MCPServers["odoo-work-cli"]
	if e.Type != "stdio" || e.Command != "/opt/bin/odoo-work-cli" || strings.Join(e.Args, " ") != "mcp" {
		t.Errorf("entry = %+v", e)
	}

	_, out, _ = execute(t, noSetup(t), "mcp", "print", "--command", "--read-only")
	if got := strings.TrimSpace(string(out)); got != "claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp --read-only" {
		t.Errorf("command = %q", got)
	}
}
