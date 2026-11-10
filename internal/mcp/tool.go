package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Param is one argument of a tool. It becomes a property of the input
// schema and a command-line flag (Flag, default the name with _ as -).
type Param struct {
	Name        string
	Type        string // "string", "integer", "boolean" or "list"
	Description string
	Required    bool
	// Positional passes the value as an argument after "--" instead of a flag.
	Positional bool
	Enum       []string
	// Flag overrides the flag name (without dashes).
	Flag string
	// Default is shown in the schema; the server does not apply it.
	Default any
	// Max bounds an integer; a larger value is a validation error.
	Max int
	// Server marks an argument the server handles itself (cursor): it is
	// in the schema but never passed to the command.
	Server bool
	// Number lets a string argument also be given as a JSON number
	// (hours: 2.5 or "2:30"); the number is passed as written.
	Number bool
}

// Tool is one tool: a command of the CLI with a fixed set of arguments.
type Tool struct {
	Name        string
	Title       string
	Description string
	// ReadOnly marks a tool that never changes anything. Destructive
	// marks a write tool that may overwrite or remove data; Idempotent one
	// whose repeated call with the same arguments has no further effect.
	// They become the MCP tool annotations.
	ReadOnly, Destructive, Idempotent bool
	// Command is the CLI command, e.g. {"search"}.
	Command []string
	Params  []Param
	// FixedFlags are always passed, e.g. "--json".
	FixedFlags []string
	// Units lists the candidates for the paged list of the result, the
	// natural unit to split along: the first of them that is a list in the
	// top-level object is paged, "" is a top-level array. A tool with Units
	// gets a cursor parameter; a result larger than the bound is then
	// returned page by page (see paginate).
	Units []string
	// Detail names the lists (by their key, e.g. "sessions") that may lose
	// entries as a last resort when a single page entry is over the bound.
	// Lists with sums and all numbers are never cut.
	Detail []string
	// Narrow tells the model how to ask for less when a result was cut,
	// e.g. "sessions:false or a smaller range".
	Narrow string
	// Defaults are flags added when the argument is absent: name -> value.
	Defaults map[string]int
	// TextDefaults are text flags added when the argument is absent or
	// empty: name -> value.
	TextDefaults map[string]string
}

func (t Tool) descriptor() map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, p := range t.Params {
		m := map[string]any{"type": p.Type, "description": p.Description}
		if p.Type == "list" {
			// An array of strings, or one comma-separated string.
			item := map[string]any{"type": "string"}
			if len(p.Enum) > 0 {
				item["enum"] = p.Enum
			}
			delete(m, "type")
			m["anyOf"] = []any{map[string]any{"type": "array", "items": item}, map[string]any{"type": "string"}}
		} else if len(p.Enum) > 0 {
			m["enum"] = p.Enum
		}
		if p.Type == "string" && p.Number {
			m["type"] = []string{"string", "number"}
		}
		if p.Default != nil {
			m["default"] = p.Default
		}
		if p.Max > 0 {
			m["maximum"] = p.Max
		}
		if p.Type == "integer" {
			m["minimum"] = 0
		}
		props[p.Name] = m
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	title := t.Title
	if title == "" {
		title = t.Name
	}
	return map[string]any{
		"name":        t.Name,
		"title":       title,
		"description": t.Description,
		"inputSchema": schema,
		"annotations": map[string]any{
			"title":           title,
			"readOnlyHint":    t.ReadOnly,
			"destructiveHint": !t.ReadOnly && t.Destructive,
			"idempotentHint":  t.ReadOnly || t.Idempotent,
			"openWorldHint":   false,
		},
	}
}

func decodeArgs(raw json.RawMessage) (map[string]json.RawMessage, error) {
	args := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return args, nil
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("arguments must be an object: %w", err)
	}
	return args, nil
}

// argv builds the command line for the arguments, validating names, types,
// enums and bounds.
func (t Tool) argv(raw json.RawMessage) ([]string, error) {
	args, err := decodeArgs(raw)
	if err != nil {
		return nil, err
	}
	for name := range args {
		if !slices.ContainsFunc(t.Params, func(p Param) bool { return p.Name == name }) {
			return nil, fmt.Errorf("unknown argument %q for tool %s", name, t.Name)
		}
	}
	argv := slices.Clone(t.Command)
	var positional []string
	set := map[string]bool{}
	for _, p := range t.Params {
		if p.Server {
			continue
		}
		v, ok := args[p.Name]
		if !ok || string(v) == "null" {
			if p.Required {
				return nil, fmt.Errorf("missing argument %q", p.Name)
			}
			continue
		}
		flag := p.Flag
		if flag == "" {
			flag = strings.ReplaceAll(p.Name, "_", "-")
		}
		switch p.Type {
		case "string":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				var n json.Number
				if !p.Number || json.Unmarshal(v, &n) != nil {
					if p.Number {
						return nil, fmt.Errorf("argument %q must be a number or a string", p.Name)
					}
					return nil, fmt.Errorf("argument %q must be a string", p.Name)
				}
				s = n.String()
			}
			if len(p.Enum) > 0 && !slices.Contains(p.Enum, s) {
				return nil, fmt.Errorf("argument %q must be one of %s, not %q", p.Name, strings.Join(p.Enum, ", "), s)
			}
			if p.Positional {
				positional = append(positional, s)
			} else if s != "" {
				argv = append(argv, "--"+flag+"="+s)
				set[p.Name] = true
			}
		case "list":
			var items []string
			if err := json.Unmarshal(v, &items); err != nil {
				var one string
				if err := json.Unmarshal(v, &one); err != nil {
					return nil, fmt.Errorf("argument %q must be an array of strings or a comma-separated string", p.Name)
				}
				items = strings.Split(one, ",")
			}
			for i, it := range items {
				items[i] = strings.TrimSpace(it)
				if len(p.Enum) > 0 && !slices.Contains(p.Enum, items[i]) {
					return nil, fmt.Errorf("argument %q: %q is not one of %s", p.Name, items[i], strings.Join(p.Enum, ", "))
				}
			}
			if len(items) > 0 {
				argv = append(argv, "--"+flag+"="+strings.Join(items, ","))
			}
		case "integer":
			var n int
			if err := json.Unmarshal(v, &n); err != nil || n < 0 {
				return nil, fmt.Errorf("argument %q must be a non-negative integer", p.Name)
			}
			if p.Max > 0 && n > p.Max {
				return nil, fmt.Errorf("argument %q is at most %d", p.Name, p.Max)
			}
			if p.Positional {
				positional = append(positional, strconv.Itoa(n))
				continue
			}
			argv = append(argv, fmt.Sprintf("--%s=%d", flag, n))
			set[p.Name] = true
		case "boolean":
			var b bool
			if err := json.Unmarshal(v, &b); err != nil {
				return nil, fmt.Errorf("argument %q must be a boolean", p.Name)
			}
			if b {
				argv = append(argv, "--"+flag)
			}
		}
	}
	for name, n := range t.Defaults {
		if !set[name] {
			argv = append(argv, fmt.Sprintf("--%s=%d", name, n))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(t.TextDefaults)) {
		if !set[name] {
			argv = append(argv, "--"+strings.ReplaceAll(name, "_", "-")+"="+t.TextDefaults[name])
		}
	}
	argv = append(argv, t.FixedFlags...)
	if len(positional) > 0 {
		argv = append(argv, "--")
		argv = append(argv, positional...)
	}
	return argv, nil
}

// cursor returns the value of the cursor argument, "" when absent.
func (t Tool) cursor(raw json.RawMessage) string {
	args, _ := decodeArgs(raw)
	var c string
	_ = json.Unmarshal(args["cursor"], &c)
	return c
}
