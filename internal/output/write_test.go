package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
)

type item struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func TestWriteList(t *testing.T) {
	two := []item{{ID: 1, Name: "a & b"}, {ID: 2, Name: "c"}}

	tests := []struct {
		name   string
		format Format
		items  []item
		want   string
	}{
		{name: "json nil is empty array", format: FormatJSON, items: nil, want: "[]\n"},
		{name: "json empty is empty array", format: FormatJSON, items: []item{}, want: "[]\n"},
		{name: "jsonl nil is empty array", format: FormatJSONL, items: nil, want: "[]\n"},
		{name: "json list, html not escaped", format: FormatJSON, items: two,
			want: "[\n  {\n    \"id\": 1,\n    \"name\": \"a & b\"\n  },\n  {\n    \"id\": 2,\n    \"name\": \"c\"\n  }\n]\n"},
		{name: "jsonl one object per line", format: FormatJSONL, items: two,
			want: "{\"id\":1,\"name\":\"a & b\"}\n{\"id\":2,\"name\":\"c\"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteList(&buf, tt.format, tt.items); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.want {
				t.Errorf("got %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestWriteObject(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   string
	}{
		{name: "json indented", format: FormatJSON, want: "{\n  \"id\": 7,\n  \"name\": \"x\"\n}\n"},
		{name: "jsonl single line", format: FormatJSONL, want: "{\"id\":7,\"name\":\"x\"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteObject(&buf, tt.format, item{ID: 7, Name: "x"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.want {
				t.Errorf("got %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name     string
		format   Format
		err      error
		wantKind string
		wantMsg  string
		oneLine  bool
	}{
		{name: "typed error", format: FormatJSON, err: apperr.New(apperr.KindAuth, "bad key"), wantKind: "auth", wantMsg: "bad key"},
		{name: "wrapped typed error", format: FormatJSON, err: fmt.Errorf("ctx: %w", apperr.New(apperr.KindNotFound, "gone")), wantKind: "not_found", wantMsg: "ctx: gone"},
		{name: "plain error is internal", format: FormatJSON, err: errors.New("boom"), wantKind: "internal", wantMsg: "boom"},
		{name: "jsonl is one line", format: FormatJSONL, err: apperr.New(apperr.KindValidation, "line1\nline2"), wantKind: "validation", wantMsg: "line1\nline2", oneLine: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			WriteError(&buf, tt.format, tt.err)
			var got struct {
				Error struct {
					Kind    string `json:"kind"`
					Message string `json:"message"`
				} `json:"error"`
			}
			dec := json.NewDecoder(&buf)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("decoding %q: %v", buf.String(), err)
			}
			if dec.More() {
				t.Error("expected exactly one JSON value")
			}
			if got.Error.Kind != tt.wantKind || got.Error.Message != tt.wantMsg {
				t.Errorf("got %+v, want kind %q message %q", got.Error, tt.wantKind, tt.wantMsg)
			}
			if tt.oneLine && strings.Count(strings.TrimSpace(buf.String()), "\n") != 0 {
				t.Errorf("jsonl error spans several lines: %q", buf.String())
			}
		})
	}
}
