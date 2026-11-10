package docsgen

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleNamed(t *testing.T, name string) SampleOutput {
	t.Helper()
	outs, err := RunSamples()
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range outs {
		if o.Sample.Name == name {
			return o
		}
	}
	t.Fatalf("no sample %q", name)
	return SampleOutput{}
}

func TestRunSamples_TableFromDemoState(t *testing.T) {
	o := sampleNamed(t, "projects")
	if o.Code != 0 || !strings.Contains(o.Output, "Lighthouse Portal") {
		t.Fatalf("projects sample: code %d\n%s", o.Code, o.Output)
	}
}

func TestRunSamples_JSONIsValid(t *testing.T) {
	outs, err := RunSamples()
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range outs {
		if !o.Sample.JSON {
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(o.Output), &v); err != nil {
			t.Errorf("%s: output is not JSON: %v\n%s", o.Sample.Name, err, o.Output)
		}
	}
}

func TestRunSamples_ErrorEnvelope(t *testing.T) {
	o := sampleNamed(t, "error-not-found")
	if o.Code != 1 || !strings.Contains(o.Output, `"kind": "not_found"`) && !strings.Contains(o.Output, `"kind":"not_found"`) {
		t.Fatalf("want exit 1 with a not_found envelope, got %d:\n%s", o.Code, o.Output)
	}
}

func TestRunSamples_Deterministic(t *testing.T) {
	a, err := RunSamples()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := RunSamples()
	for i := range a {
		if a[i].Output != b[i].Output {
			t.Errorf("%s differs between runs", a[i].Sample.Name)
		}
	}
}

func TestSample_CommandLineQuotesArgs(t *testing.T) {
	s := Sample{Args: []string{"entries", "add", "--description", "Code review"}}
	if got, want := s.CommandLine(), `odoo-work-cli entries add --description "Code review"`; got != want {
		t.Errorf("CommandLine = %q, want %q", got, want)
	}
}
