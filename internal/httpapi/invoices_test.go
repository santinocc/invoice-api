package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santinocc/invoice-api/internal/invoice"
)

const validBody = `{"vendor":"Example Supplies","issue_date":"2026-11-01","due_date":"2026-11-30","items":[{"description":"Rice","quantity":2,"unit_price":540,"tax_rate":8},{"description":"Paper","quantity":1,"unit_price":1000,"tax_rate":10}]}`

func TestCreateInvoice(t *testing.T) {
	h := NewServer(invoice.NewMemoryStore()).Routes()

	req := httptest.NewRequest(http.MethodPost, "/invoices", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body)
	}

	var got invoice.Invoice
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Error("response has no id")
	}
	if got.Subtotal != 2080 || got.Tax != 186 || got.Total != 2266 {
		t.Errorf("totals = %d/%d/%d, want 2080/186/2266", got.Subtotal, got.Tax, got.Total)
	}
}

func TestGetInvoiceNotFound(t *testing.T) {
	h := NewServer(invoice.NewMemoryStore()).Routes()

	req := httptest.NewRequest(http.MethodGet, "/invoices/inv_999", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreateInvoiceBadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "not json", body: `not json`},
		{name: "empty vendor", body: `{"vendor":""}`},
		{name: "unknown field", body: `{"vendr":"Example"}`},
		{name: "empty body", body: ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewServer(invoice.NewMemoryStore()).Routes()
			req := httptest.NewRequest(http.MethodPost, "/invoices", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body)
			}
		})
	}
}
