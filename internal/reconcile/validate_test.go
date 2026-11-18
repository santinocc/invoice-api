package reconcile

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	ok := Transaction{ID: "tx_1", Date: "2026-11-28", Amount: 2266, Payer: "ACME"}

	if err := Validate([]Transaction{ok}); err != nil {
		t.Fatalf("valid transaction: got %v, want nil", err)
	}

	tooMany := make([]Transaction, MaxTransactions+1)
	for i := range tooMany {
		tooMany[i] = ok
	}

	tests := []struct {
		name string
		txns []Transaction
	}{
		{"no transactions", nil},
		{"more than the maximum", tooMany},
		{"empty ID", []Transaction{{ID: "", Date: "2026-11-28", Amount: 2266}}},
		{"the same ID twice", []Transaction{ok, ok}},
		{"zero amount", []Transaction{{ID: "tx_1", Date: "2026-11-28", Amount: 0}}},
		{"negative amount", []Transaction{{ID: "tx_1", Date: "2026-11-28", Amount: -500}}},
		{"bad date", []Transaction{{ID: "tx_1", Date: "28/11/2026", Amount: 2266}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.txns); !errors.Is(err, ErrInvalidTransactions) {
				t.Errorf("got %v, want an error wrapping ErrInvalidTransactions", err)
			}
		})
	}
}
