package reconcile

import (
	"reflect"
	"testing"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// Open invoices used by every case.
var invoices = []invoice.Invoice{
	{ID: "inv_a", Vendor: "Example Supplies", Total: 2266},
	{ID: "inv_b", Vendor: "ACME", Total: 5400},
	{ID: "inv_c", Vendor: "Kanto Foods", Total: 5400}, // same total as inv_b
	{ID: "inv_d", Vendor: "ACME", Total: 1100},
	{ID: "inv_e", Vendor: "ACME", Total: 1100}, // same total AND vendor as inv_d
}

func TestSuggest(t *testing.T) {
	tests := []struct {
		name string
		txns []Transaction
		want Result
	}{
		{
			name: "one invoice has the amount: matched, even if the payer name differs",
			txns: []Transaction{{ID: "tx_1", Amount: 2266, Payer: "ｴｸｻﾝﾌﾟﾙ"}},
			want: Result{Matched: []Match{{TransactionID: "tx_1", InvoiceID: "inv_a"}}},
		},
		{
			name: "no invoice has the amount: unmatched",
			txns: []Transaction{{ID: "tx_1", Amount: 999, Payer: "Nobody"}},
			want: Result{Unmatched: []string{"tx_1"}},
		},
		{
			name: "two invoices have the amount, the payer picks one: matched",
			txns: []Transaction{{ID: "tx_1", Amount: 5400, Payer: "Kanto Foods"}},
			want: Result{Matched: []Match{{TransactionID: "tx_1", InvoiceID: "inv_c"}}},
		},
		{
			name: "two invoices have the amount, the payer matches neither: ambiguous",
			txns: []Transaction{{ID: "tx_1", Amount: 5400, Payer: "Someone Else"}},
			want: Result{Ambiguous: []Ambiguity{{TransactionID: "tx_1", InvoiceIDs: []string{"inv_b", "inv_c"}}}},
		},
		{
			name: "same amount and same vendor: ambiguous",
			txns: []Transaction{{ID: "tx_1", Amount: 1100, Payer: "ACME"}},
			want: Result{Ambiguous: []Ambiguity{{TransactionID: "tx_1", InvoiceIDs: []string{"inv_d", "inv_e"}}}},
		},
		{
			name: "several transactions keep their request order in each list",
			txns: []Transaction{
				{ID: "tx_1", Amount: 999},
				{ID: "tx_2", Amount: 2266},
				{ID: "tx_3", Amount: 5400, Payer: "ACME"},
				{ID: "tx_4", Amount: 1},
			},
			want: Result{
				Matched: []Match{
					{TransactionID: "tx_2", InvoiceID: "inv_a"},
					{TransactionID: "tx_3", InvoiceID: "inv_b"},
				},
				Unmatched: []string{"tx_1", "tx_4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Suggest(tt.txns, invoices)
			// reflect.DeepEqual compares nested structs and slices field by field.
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}
