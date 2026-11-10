package changelog

import "testing"

func TestMissingEntry(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{"nothing changed", nil, false},
		{"tui change without entry", []string{"internal/tui/render.go"}, true},
		{"cmd change without entry", []string{"cmd/odoo-work-cli/main.go"}, true},
		{"cli change with entry", []string{"internal/cli/entries.go", "CHANGELOG.md"}, false},
		{"test only", []string{"internal/tui/render_test.go"}, false},
		{"testdata only", []string{"internal/docsgen/testdata/frames/grid.txt"}, false},
		{"docs only", []string{"docs/how-to/95-troubleshoot.md"}, false},
		{"workflow only", []string{".github/workflows/ci.yml"}, false},
		{"test and code", []string{"internal/odoo/client_test.go", "internal/odoo/client.go"}, true},
		{"similar prefix", []string{"internals/x.go", "command/x.go"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MissingEntry(tt.files); got != tt.want {
				t.Errorf("MissingEntry(%v) = %v, want %v", tt.files, got, tt.want)
			}
		})
	}
}
