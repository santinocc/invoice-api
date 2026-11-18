package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santinocc/invoice-api/internal/invoice"
)

func TestReconcile(t *testing.T) {
	store := invoice.NewMemoryStore()
	h := NewServer(store).Routes()

	// Create inv_1 through the API, so its totals are calculated: total 2266.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/invoices", strings.NewReader(validBody)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invoice: status %d; body: %s", rec.Code, rec.Body)
	}

	body := `{"transactions":[
		{"id":"tx_1","date":"2026-11-28","amount":2266,"payer":"ｴｸｻﾝﾌﾟﾙ"},
		{"id":"tx_2","date":"2026-11-28","amount":999,"payer":"Nobody"}
	]}`
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/reconcile", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	want := `{"matched":[{"transaction_id":"tx_1","invoice_id":"inv_1"}],"ambiguous":[],"unmatched":["tx_2"]}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("\n got  %s want %s", got, want)
	}
}

func TestReconcileBadRequest(t *testing.T) {
	h := NewServer(invoice.NewMemoryStore()).Routes()

	tests := []struct {
		name string
		body string
	}{
		{"not JSON", `nope`},
		{"unknown field", `{"transactions":[],"extra":1}`},
		{"no transactions", `{"transactions":[]}`},
		{"amount zero", `{"transactions":[{"id":"tx_1","date":"2026-11-28","amount":0}]}`},
		{"duplicate id", `{"transactions":[{"id":"tx_1","date":"2026-11-28","amount":1},{"id":"tx_1","date":"2026-11-28","amount":2}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/reconcile", strings.NewReader(tt.body)))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
		})
	}
}
