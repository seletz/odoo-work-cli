package odoo

import (
	"errors"
	"testing"
)

func TestGetFields(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		model   string
		wantLen int
		wantErr bool
		wantMsg string
		checkFn func(t *testing.T, fields []FieldInfo)
	}{
		{
			name: "success returns fields",
			client: &mockClient{
				fields: []FieldInfo{
					{Name: "id", Type: "integer", String: "ID", Required: true},
					{Name: "name", Type: "char", String: "Name", Required: true},
					{Name: "active", Type: "boolean", String: "Active", Required: false},
				},
			},
			model:   "project.project",
			wantLen: 3,
			checkFn: func(t *testing.T, fields []FieldInfo) {
				t.Helper()
				if fields[0].Name != "id" {
					t.Errorf("fields[0].Name = %q, want %q", fields[0].Name, "id")
				}
				if fields[0].Type != "integer" {
					t.Errorf("fields[0].Type = %q, want %q", fields[0].Type, "integer")
				}
				if !fields[0].Required {
					t.Error("fields[0].Required = false, want true")
				}
			},
		},
		{
			name:    "empty fields returns no error",
			client:  &mockClient{fields: []FieldInfo{}},
			model:   "res.users",
			wantLen: 0,
		},
		{
			name:    "error is propagated",
			client:  &mockClient{fieldsErr: errors.New("model not found")},
			model:   "nonexistent.model",
			wantErr: true,
			wantMsg: "model not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := tt.client.GetFields(tt.model)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.wantMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(fields) != tt.wantLen {
				t.Fatalf("len(fields) = %d, want %d", len(fields), tt.wantLen)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, fields)
			}
		})
	}
}
