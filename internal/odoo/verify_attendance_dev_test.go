//go:build devverify

package odoo

import (
	"os"
	"testing"
	"time"
)

// TestDevVerify_ListAttendance manually verifies ListAttendance against the
// dev environment. Run with:
//
//	go test -tags devverify -run TestDevVerify_ListAttendance -v ./internal/odoo/
func TestDevVerify_ListAttendance(t *testing.T) {
	url := os.Getenv("ODOO_URL")
	if url == "" {
		t.Skip("ODOO_URL not set")
	}
	client, err := NewXMLRPCClient(url, os.Getenv("ODOO_DATABASE"),
		os.Getenv("ODOO_USERNAME"), os.Getenv("ODOO_PASSWORD"),
		os.Getenv("ODOO_WEB_PASSWORD"), os.Getenv("ODOO_TOTP_SECRET"), nil)
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	defer client.Close()

	now := time.Now()
	// Monday of the current week.
	offset := (int(now.Weekday()) + 6) % 7
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -offset)
	to := from.AddDate(0, 0, 7)

	empID, err := client.findEmployeeID()
	if err != nil {
		t.Fatalf("findEmployeeID: %v", err)
	}

	// Seed: one closed record inside the week, one open record that started
	// before the week (midnight wrap). The dev user is an admin, so direct
	// XML-RPC creates on hr.attendance are allowed.
	create := func(fields map[string]interface{}) int64 {
		ids, err := client.client.Create("hr.attendance", []interface{}{fields}, nil)
		if err != nil || len(ids) == 0 {
			t.Fatalf("creating hr.attendance: %v", err)
		}
		return ids[0]
	}
	closedID := create(map[string]interface{}{
		"employee_id": empID,
		"check_in":    from.Add(32 * time.Hour).Format(odooDatetimeFormat), // Tue 08:00
		"check_out":   from.Add(36 * time.Hour).Format(odooDatetimeFormat), // Tue 12:00
	})
	openID := create(map[string]interface{}{
		"employee_id": empID,
		"check_in":    from.Add(-time.Hour).Format(odooDatetimeFormat), // Sun 23:00
	})
	defer func() {
		if err := client.client.Delete("hr.attendance", []int64{closedID, openID}); err != nil {
			t.Errorf("cleanup unlink failed: %v", err)
		}
	}()

	records, err := client.ListAttendance(from, to)
	if err != nil {
		t.Fatalf("ListAttendance: %v", err)
	}
	t.Logf("week %s..%s: %d records, total %.2f h",
		from.Format("2006-01-02"), to.Format("2006-01-02"),
		len(records), SumAttendanceHours(records, now))
	for _, r := range records {
		out := "open"
		if r.CheckOut != nil {
			out = r.CheckOut.Format("2006-01-02 15:04")
		}
		t.Logf("  #%d %s -> %s (%.2f h)", r.ID, r.CheckIn.Format("2006-01-02 15:04"), out, r.WorkedHours)
	}

	found := map[int64]bool{}
	for _, r := range records {
		found[r.ID] = true
	}
	if !found[closedID] {
		t.Errorf("closed in-week record %d missing from ListAttendance result", closedID)
	}
	if !found[openID] {
		t.Errorf("open midnight-wrap record %d missing from ListAttendance result", openID)
	}
}
