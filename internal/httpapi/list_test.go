package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// newListServer returns a handler whose store holds four invoices,
// inv_1 to inv_4, the same data as the invoice package's List test.
// Sorted by due date then ID they are: inv_4, inv_1, inv_2, inv_3.
func newListServer(t *testing.T) http.Handler {
	t.Helper()
	s := invoice.NewMemoryStore()
	for _, inv := range []invoice.Invoice{
		{Vendor: "ACME", DueDate: "2026-11-30"},
		{Vendor: "Example", DueDate: "2026-11-30"},
		{Vendor: "ACME", DueDate: "2026-12-15"},
		{Vendor: "Example", DueDate: "2026-11-20"},
	} {
		if _, err := s.Create(t.Context(), inv); err != nil {
			t.Fatal(err)
		}
	}
	return NewServer(s).Routes()
}

// getList calls GET /invoices with the given query and decodes a 200 response.
func getList(t *testing.T, h http.Handler, query url.Values) listResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invoices?"+query.Encode(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var resp listResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func ids(invs []invoice.Invoice) []string {
	var out []string
	for _, inv := range invs {
		out = append(out, inv.ID)
	}
	return out
}

// TestListInvoicesPages follows next_cursor until it runs out, the way a client would.
func TestListInvoicesPages(t *testing.T) {
	h := newListServer(t)

	var pages [][]string
	query := url.Values{"limit": {"2"}}
	for range 10 { // a safety stop, in case a bug makes the cursor loop forever
		resp := getList(t, h, query)
		pages = append(pages, ids(resp.Invoices))
		if resp.NextCursor == "" {
			break
		}
		query.Set("cursor", resp.NextCursor)
	}

	want := [][]string{{"inv_4", "inv_1"}, {"inv_2", "inv_3"}}
	if !slices.EqualFunc(pages, want, slices.Equal) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
}

func TestListInvoicesFiltersAndEmpty(t *testing.T) {
	h := newListServer(t)

	resp := getList(t, h, url.Values{"vendor": {"ACME"}, "due_before": {"2026-12-01"}})
	if got := ids(resp.Invoices); !slices.Equal(got, []string{"inv_1"}) {
		t.Errorf("vendor + due_before = %v, want [inv_1]", got)
	}
	if resp.NextCursor != "" {
		t.Errorf("next_cursor = %q on the only page, want none", resp.NextCursor)
	}

	// No match must be an empty JSON list, not null.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invoices?vendor=Nobody", nil))
	if body := rec.Body.String(); body != "{\"invoices\":[]}\n" {
		t.Errorf("body = %q, want an empty invoices list", body)
	}
}

func TestListInvoicesBadRequest(t *testing.T) {
	h := newListServer(t)

	tests := []struct {
		name  string
		query string
	}{
		{"limit not a number", "limit=abc"},
		{"limit zero", "limit=0"},
		{"limit above the maximum", "limit=101"},
		{"due_before not a date", "due_before=next-week"},
		{"cursor not base64", "cursor=%21%21%21"},
		{"cursor without an ID", "cursor=" + encodeCursor("2026-11-30", "")},
		{"cursor with a bad date", "cursor=" + encodeCursor("someday", "inv_1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invoices?"+tt.query, nil))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
		})
	}
}
