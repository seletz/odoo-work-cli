// Package mcp is a small Model Context Protocol server for stdio: newline
// delimited JSON-RPC 2.0 with initialize, ping, tools/list and tools/call.
// The package knows nothing about Odoo: a Tool maps its arguments to
// command-line arguments and the Runner executes them. Read and write
// tools differ only in their annotations.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// ProtocolVersions are the protocol revisions this server speaks, newest
// first. initialize answers with the client's version when it is in the
// list, else with the newest one.
var ProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// kindValidation is the error kind of bad tool arguments, the same kind
// the CLI reports for bad flags (apperr.KindValidation).
const kindValidation = "validation"

// JSON-RPC error codes.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Runner executes a command line (without the program name) and returns its
// stdout and whether it failed.
type Runner func(ctx context.Context, argv []string) (stdout string, failed bool)

// Server is one MCP server. Zero MaxBytes uses DefaultMaxBytes.
type Server struct {
	// Name and Version identify the server in initialize.
	Name, Version string
	// Instructions is shown to the model by clients that support it.
	Instructions string
	Tools        []Tool
	Run          Runner
	// MaxBytes bounds the size of the first content item of a tool result.
	MaxBytes int
}

// DefaultMaxBytes is the default bound of the first content item of one tool
// result: 32 KiB. Claude Code accepts about 25,000 tokens per MCP result;
// JSON with ids and paths costs up to about 2 bytes per token, so 32 KiB
// stays near 16,000 tokens even in that worst case, with room for the hint.
const DefaultMaxBytes = 32 * 1024

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve reads messages from r and writes responses to w until r ends. It
// returns nil at end of input; only a write error or a read error ends it
// with an error.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if out := s.Handle(ctx, line); out != nil {
				if _, werr := w.Write(append(out, '\n')); werr != nil {
					return werr
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

// Handle processes one message and returns the response, or nil for a
// notification.
func (s *Server) Handle(ctx context.Context, msg []byte) []byte {
	msg = bytes.TrimSpace(msg)
	if len(msg) > 0 && msg[0] == '[' {
		return marshal(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeInvalidRequest, "batches are not supported"}})
	}
	var req request
	if err := json.Unmarshal(msg, &req); err != nil {
		return marshal(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "parse error: " + err.Error()}})
	}
	isNotification := req.ID == nil
	if req.JSONRPC != "2.0" || req.Method == "" {
		if isNotification {
			return nil
		}
		return marshal(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{codeInvalidRequest, "not a JSON-RPC 2.0 request"}})
	}
	if isNotification {
		return nil // notifications/initialized, notifications/cancelled, ...
	}
	result, rerr := s.dispatch(ctx, req)
	if rerr != nil {
		return marshal(response{JSONRPC: "2.0", ID: req.ID, Error: rerr})
	}
	return marshal(response{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func marshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32603, err.Error()}})
	}
	return b
}

func (s *Server) dispatch(ctx context.Context, req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &p); err != nil {
				return nil, &rpcError{codeInvalidParams, "invalid params: " + err.Error()}
			}
		}
		v := ProtocolVersions[0]
		if slices.Contains(ProtocolVersions, p.ProtocolVersion) {
			v = p.ProtocolVersion
		}
		res := map[string]any{
			"protocolVersion": v,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
		}
		if s.Instructions != "" {
			res["instructions"] = s.Instructions
		}
		return res, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.Tools))
		for _, t := range s.Tools {
			tools = append(tools, t.descriptor())
		}
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil || p.Name == "" {
			return nil, &rpcError{codeInvalidParams, "tools/call needs a tool name"}
		}
		i := slices.IndexFunc(s.Tools, func(t Tool) bool { return t.Name == p.Name })
		if i < 0 {
			return nil, &rpcError{codeInvalidParams, fmt.Sprintf("unknown tool %q", p.Name)}
		}
		return s.call(ctx, s.Tools[i], p.Arguments), nil
	}
	return nil, &rpcError{codeMethodNotFound, "method not found: " + req.Method}
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

func text(s string) content { return content{Type: "text", Text: s} }

// errorResult is a tool failure in the shape of the CLI's JSON error.
func errorResult(kind, msg string) toolResult {
	b, _ := json.Marshal(map[string]any{"error": map[string]string{"kind": kind, "message": msg}})
	return toolResult{Content: []content{text(string(b))}, IsError: true}
}

func (s *Server) call(ctx context.Context, t Tool, rawArgs json.RawMessage) toolResult {
	argv, err := t.argv(rawArgs)
	if err != nil {
		return errorResult(kindValidation, err.Error())
	}
	out, failed := s.Run(ctx, argv)
	if failed {
		return toolResult{Content: []content{text(argumentNames(t, strings.TrimSpace(out)))}, IsError: true}
	}
	max := s.MaxBytes
	if max <= 0 {
		max = DefaultMaxBytes
	}
	first, hint, err := s.bound(t, strings.TrimSpace(out), t.cursor(rawArgs), queryHash(argv), max)
	if err != nil {
		return errorResult(kindValidation, err.Error())
	}
	res := toolResult{Content: []content{text(first)}}
	if hint != nil {
		b, _ := json.Marshal(hint)
		res.Content = append(res.Content, text(string(b)))
	}
	return res
}

// argumentNames rewrites the flags a validation error of the CLI names (--project-id)
// to the arguments of the tool (argument "by_session"): the caller of a
// tool never saw a flag. Only flags that are an argument of t are renamed
// and only in errors of kind validation; any other output is returned as
// it is.
func argumentNames(t Tool, out string) string {
	var e struct {
		Error struct {
			Kind    string `json:"kind"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(out), &e) != nil || e.Error.Kind != kindValidation {
		return out
	}
	msg := e.Error.Message
	for _, p := range t.Params {
		if p.Server || p.Positional {
			continue
		}
		flag := p.Flag
		if flag == "" {
			flag = strings.ReplaceAll(p.Name, "_", "-")
		}
		// A flag ends at a character a flag name cannot contain, so --limit
		// does not match inside --limits.
		msg = flagRef(flag).ReplaceAllString(msg, "${1}argument "+strconv.Quote(p.Name)+"${2}")
	}
	if msg == e.Error.Message {
		return out
	}
	b, err := json.Marshal(map[string]any{"error": map[string]string{"kind": kindValidation, "message": msg}})
	if err != nil {
		return out
	}
	return string(b)
}

// flagRef matches --flag as a whole word; group 1 is what precedes it.
func flagRef(flag string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^A-Za-z0-9-])--` + regexp.QuoteMeta(flag) + `($|[^A-Za-z0-9_-])`)
}

// bound makes out fit into max bytes: verbatim when it fits, compacted when
// that is enough, else one page of the paged list of the tool, else cut as
// a last resort (see truncate). A cursor selects the page.
func (s *Server) bound(t Tool, out, cursor, query string, max int) (string, *Hint, error) {
	if cursor != "" {
		// A cursor of other arguments is wrong whatever the result is.
		if _, err := checkCursor(cursor, query, nil); err != nil {
			return "", nil, err
		}
	}
	if cursor == "" && len(out) <= max {
		return out, nil, nil
	}
	var cb bytes.Buffer
	if json.Compact(&cb, []byte(out)) != nil {
		if cursor != "" {
			return "", nil, errNoPages
		}
		return out, nil, nil
	}
	if cursor == "" && cb.Len() <= max {
		return cb.String(), nil, nil
	}
	if len(t.Units) > 0 {
		page, hint, ok, err := paginate(cb.Bytes(), t, cursor, query, max)
		if err != nil {
			return "", nil, err
		}
		if ok {
			return string(page), hint, nil
		}
	}
	if cursor != "" {
		return "", nil, errNoPages
	}
	return truncate(cb.String(), max, t.Detail, t.Narrow)
}

// queryHash identifies the arguments of a call without the cursor: the
// command line built from them. A cursor is only good for the query that
// issued it.
func queryHash(argv []string) string {
	b, _ := json.Marshal(argv)
	return digest(b)
}
