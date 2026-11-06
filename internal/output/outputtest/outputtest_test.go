package outputtest

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/output"
)

// TestValidate_RejectsContractViolations guards against schemas that
// accept anything, which would make every schema test vacuous.
func TestValidate_RejectsContractViolations(t *testing.T) {
	valid := `{"id":1,"date":"2026-07-16","project_id":1,"project":"P","task_id":null,"task":null,` +
		`"description":"d","hours":1.5,"minutes":90,"employee":null,"status":"draft","company":null}`

	tests := []struct {
		name    string
		schema  string
		doc     string
		wantErr bool
	}{
		{name: "valid entry", schema: output.SchemaEntry, doc: valid},
		{name: "missing field", schema: output.SchemaEntry, doc: strings.Replace(valid, `"minutes":90,`, "", 1), wantErr: true},
		{name: "unknown field", schema: output.SchemaEntry, doc: strings.Replace(valid, `"id":1,`, `"id":1,"extra":1,`, 1), wantErr: true},
		{name: "id as string", schema: output.SchemaEntry, doc: strings.Replace(valid, `"id":1,`, `"id":"1",`, 1), wantErr: true},
		{name: "bad date", schema: output.SchemaEntry, doc: strings.Replace(valid, `2026-07-16`, `16.07.2026`, 1), wantErr: true},
		{name: "bad datetime", schema: output.SchemaAttendance, doc: `{"id":1,"employee_id":null,"employee":null,"check_in":"08:00","check_out":null,"running":true,"worked_hours":0,"worked_minutes":0}`, wantErr: true},
		{name: "error without kind", schema: output.SchemaError, doc: `{"error":{"message":"x"}}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.schema, []byte(tt.doc))
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEverySchemaCompiles(t *testing.T) {
	files, err := fs.Glob(output.Schemas, "schemas/*.schema.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no schemas: %v", err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(strings.TrimPrefix(f, "schemas/"), ".schema.json")
		if _, err := schema(name); err != nil {
			t.Errorf("schema %s: %v", name, err)
		}
	}
}
