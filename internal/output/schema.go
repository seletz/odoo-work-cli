package output

import "embed"

// Schemas holds the JSON Schemas (draft 2020-12) of every machine-readable
// output shape, as schemas/<name>.schema.json. They are the contract
// verified by the per-command schema tests.
//
//go:embed schemas/*.schema.json
var Schemas embed.FS

// Schema names, one per output shape.
const (
	SchemaProject        = "project"
	SchemaTask           = "task"
	SchemaEntry          = "entry"
	SchemaUser           = "user"
	SchemaAttendance     = "attendance"
	SchemaClockStatus    = "clock_status"
	SchemaAttendanceEdit = "attendance_edit"
	SchemaError          = "error"
)
