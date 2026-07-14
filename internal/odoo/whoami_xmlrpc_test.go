package odoo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// whoAmIAllowedFields is the exact minimal field set WhoAmI must request.
// Anything beyond this (or an unrestricted read) makes Odoo touch fields
// backed by res.users.log, which non-admin users may not read.
var whoAmIAllowedFields = []string{"company_id", "email", "login", "name"}

const authenticateResponse = `<?xml version="1.0"?>
<methodResponse><params><param><value><int>42</int></value></param></params></methodResponse>`

// aclFaultResponse mimics the fault Odoo raises for non-admin users when a
// res.users read touches fields backed by res.users.log (issue #40).
const aclFaultResponse = `<?xml version="1.0"?>
<methodResponse><fault><value><struct>
<member><name>faultCode</name><value><int>4</int></value></member>
<member><name>faultString</name><value><string>Sie haben keinen Zugriff auf Datensaetze "Users Log" (res.users.log).</string></value></member>
</struct></value></fault></methodResponse>`

const userRecordResponse = `<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><struct>
<member><name>id</name><value><int>42</int></value></member>
<member><name>name</name><value><string>Test User</string></value></member>
<member><name>login</name><value><string>test@example.com</string></value></member>
<member><name>email</name><value><string>test@example.com</string></value></member>
<member><name>company_id</name><value><array><data>
<value><int>1</int></value><value><string>ACME Corp</string></value>
</data></array></value></member>
</struct></value>
</data></array></value></param></params></methodResponse>`

var xmlStringRe = regexp.MustCompile(`<string>([^<]*)</string>`)

// extractRequestedFields parses the "fields" option out of an XML-RPC
// execute_kw request body. Returns nil when no field restriction is present.
func extractRequestedFields(body string) []string {
	idx := strings.Index(body, "<name>fields</name>")
	if idx < 0 {
		return nil
	}
	rest := body[idx:]
	end := strings.Index(rest, "</array>")
	if end < 0 {
		return nil
	}
	var fields []string
	for _, m := range xmlStringRe.FindAllStringSubmatch(rest[:end], -1) {
		fields = append(fields, m[1])
	}
	return fields
}

// newMockOdooServer simulates an Odoo XML-RPC endpoint as seen by a
// non-admin user: reading res.users without a minimal field restriction
// raises the res.users.log ACL fault.
func newMockOdooServer(t *testing.T, requestedFields *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/xml")

		switch r.URL.Path {
		case "/xmlrpc/2/common":
			_, _ = fmt.Fprint(w, authenticateResponse)
		case "/xmlrpc/2/object":
			fields := extractRequestedFields(string(body))
			*requestedFields = fields
			sorted := append([]string(nil), fields...)
			sort.Strings(sorted)
			if strings.Join(sorted, ",") != strings.Join(whoAmIAllowedFields, ",") {
				_, _ = fmt.Fprint(w, aclFaultResponse)
				return
			}
			_, _ = fmt.Fprint(w, userRecordResponse)
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestWhoAmI_RequestsMinimalFields(t *testing.T) {
	var requestedFields []string
	server := newMockOdooServer(t, &requestedFields)
	defer server.Close()

	client, err := NewXMLRPCClient(server.URL, "testdb", "test@example.com", "api-key", "", "", nil)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	defer client.Close()

	info, err := client.WhoAmI()
	if err != nil {
		t.Fatalf("WhoAmI() error: %v (requested fields: %v)", err, requestedFields)
	}

	sorted := append([]string(nil), requestedFields...)
	sort.Strings(sorted)
	if strings.Join(sorted, ",") != strings.Join(whoAmIAllowedFields, ",") {
		t.Errorf("requested fields = %v, want exactly %v", requestedFields, whoAmIAllowedFields)
	}

	if info.ID != 42 {
		t.Errorf("ID = %d, want 42", info.ID)
	}
	if info.Name != "Test User" {
		t.Errorf("Name = %q, want %q", info.Name, "Test User")
	}
	if info.Login != "test@example.com" {
		t.Errorf("Login = %q, want %q", info.Login, "test@example.com")
	}
	if info.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", info.Email, "test@example.com")
	}
	if info.Company != "ACME Corp" {
		t.Errorf("Company = %q, want %q", info.Company, "ACME Corp")
	}
}
