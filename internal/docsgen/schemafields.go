package docsgen

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/output"
)

// schemaNode is the part of a JSON Schema the reference reads.
type schemaNode struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Type        json.RawMessage       `json:"type"`
	Ref         string                `json:"$ref"`
	OneOf       []schemaNode          `json:"oneOf"`
	Items       *schemaNode           `json:"items"`
	Const       json.RawMessage       `json:"const"`
	Format      string                `json:"format"`
	Pattern     string                `json:"pattern"`
	Required    []string              `json:"required"`
	Properties  map[string]schemaNode `json:"properties"`
	Defs        map[string]schemaNode `json:"$defs"`
	order       []string
}

func loadSchema(name string) (*schemaNode, error) {
	raw, err := output.Schemas.ReadFile("schemas/" + name)
	if err != nil {
		return nil, err
	}
	var s schemaNode
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	// Keep the property order of the file: it is the order of the output.
	var keys struct {
		Properties json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, err
	}
	if s.order, err = objectKeys(keys.Properties); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &s, nil
}

// objectKeys returns the keys of a JSON object in file order.
func objectKeys(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// schemaFields lists the fields of schema name (e.g. "entry") with their
// types, for OUTPUT.
//
// :param name: the schema name, without .schema.json
// :returns: one definition per field
func schemaFields(name string) ([]def, error) {
	s, err := loadSchema(name + ".schema.json")
	if err != nil {
		return nil, err
	}
	defs, err := loadSchema("defs.schema.json")
	if err != nil {
		return nil, err
	}
	req := map[string]bool{}
	for _, r := range s.Required {
		req[r] = true
	}
	var out []def
	for _, field := range s.order {
		p := s.Properties[field]
		meta := typeOf(p, defs)
		if !req[field] {
			meta += ", may be absent"
		}
		d := def{term: []span{{text: field, style: styleCode}}, meta: meta}
		if p.Description != "" {
			d.body = []string{strings.TrimSuffix(p.Description, ".") + "."}
		}
		out = append(out, d)
	}
	return out, nil
}

// typeOf describes the type of a schema node in words.
func typeOf(n schemaNode, defs *schemaNode) string {
	switch {
	case n.Ref != "":
		if strings.HasPrefix(n.Ref, "defs.schema.json#/$defs/") {
			return typeOf(defs.Defs[strings.TrimPrefix(n.Ref, "defs.schema.json#/$defs/")], defs)
		}
		return "object (" + n.Ref + ")"
	case len(n.OneOf) > 0:
		var alts []string
		for _, o := range n.OneOf {
			alts = append(alts, typeOf(o, defs))
		}
		return strings.Join(alts, " or ")
	case len(n.Const) > 0:
		return "always " + string(n.Const)
	}
	var types []string
	if len(n.Type) > 0 && n.Type[0] == '[' {
		_ = json.Unmarshal(n.Type, &types)
	} else if len(n.Type) > 0 {
		var t string
		_ = json.Unmarshal(n.Type, &t)
		types = []string{t}
	}
	for i, t := range types {
		switch {
		case t == "array" && n.Items != nil:
			types[i] = "array of " + typeOf(*n.Items, defs)
		case t == "string" && n.Format == "date-time":
			types[i] = "string (RFC 3339 timestamp)"
		case t == "string" && n.Pattern == `^\d{4}-\d{2}-\d{2}$`:
			types[i] = "string (YYYY-MM-DD)"
		}
	}
	return strings.Join(types, " or ")
}
