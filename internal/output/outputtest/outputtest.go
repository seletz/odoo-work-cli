// Package outputtest validates command output against the JSON Schemas
// embedded in package output. It is meant for tests only.
package outputtest

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/seletz/odoo-work-cli/internal/output"
)

const baseURL = "https://odoo-work-cli.invalid/schemas/"

var (
	compileOnce sync.Once
	compiler    *jsonschema.Compiler
	compileErr  error
	cacheMu     sync.Mutex
	cache       = map[string]*jsonschema.Schema{}
)

func load() (*jsonschema.Compiler, error) {
	compileOnce.Do(func() {
		c := jsonschema.NewCompiler()
		c.AssertFormat()
		files, err := fs.Glob(output.Schemas, "schemas/*.schema.json")
		if err != nil {
			compileErr = err
			return
		}
		for _, f := range files {
			raw, err := output.Schemas.ReadFile(f)
			if err != nil {
				compileErr = err
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				compileErr = err
				return
			}
			if err := c.AddResource(baseURL+strings.TrimPrefix(f, "schemas/"), doc); err != nil {
				compileErr = err
				return
			}
		}
		compiler = c
	})
	return compiler, compileErr
}

func schema(name string) (*jsonschema.Schema, error) {
	c, err := load()
	if err != nil {
		return nil, err
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if s, ok := cache[name]; ok {
		return s, nil
	}
	s, err := c.Compile(baseURL + name + ".schema.json")
	if err != nil {
		return nil, err
	}
	cache[name] = s
	return s, nil
}

// Validate checks one JSON document against the named schema.
//
// :param name: schema name (output.Schema* constant)
// :param raw: the JSON document
// :returns: a parse, compile or validation error
func Validate(name string, raw []byte) error {
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	s, err := schema(name)
	if err != nil {
		return err
	}
	return s.Validate(inst)
}

func validate(t *testing.T, name string, raw []byte) {
	t.Helper()
	if err := Validate(name, raw); err != nil {
		t.Fatalf("output does not match schema %q: %v\n%s", name, err, raw)
	}
}

// AssertObject checks that out is exactly one JSON document matching the
// named schema.
//
// :param t: the test
// :param name: schema name (output.Schema* constant)
// :param out: the command's stdout
func AssertObject(t *testing.T, name string, out []byte) {
	t.Helper()
	assertSingleDocument(t, out)
	validate(t, name, out)
}

// AssertList checks that out is one JSON array whose items match the
// named schema, and returns the number of items.
//
// :param t: the test
// :param name: item schema name (output.Schema* constant)
// :param out: the command's stdout
// :returns: the number of items
func AssertList(t *testing.T, name string, out []byte) int {
	t.Helper()
	assertSingleDocument(t, out)
	var items []json.RawMessage
	if err := json.Unmarshal(out, &items); err != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", err, out)
	}
	for _, it := range items {
		validate(t, name, it)
	}
	return len(items)
}

// AssertLines checks jsonl output: every line is an object matching the
// named schema. An empty result must be the single line "[]".
//
// :param t: the test
// :param name: item schema name (output.Schema* constant)
// :param out: the command's stdout
// :returns: the number of records
func AssertLines(t *testing.T, name string, out []byte) int {
	t.Helper()
	text := strings.TrimSuffix(string(out), "\n")
	if text == "[]" {
		return 0
	}
	if text == "" {
		t.Fatal("empty output; an empty jsonl result must be []")
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		validate(t, name, []byte(line))
	}
	return len(lines)
}

func assertSingleDocument(t *testing.T, out []byte) {
	t.Helper()
	if len(bytes.TrimSpace(out)) == 0 {
		t.Fatal("empty output")
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var v json.RawMessage
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if dec.More() {
		t.Fatalf("output holds more than one JSON document:\n%s", out)
	}
}
