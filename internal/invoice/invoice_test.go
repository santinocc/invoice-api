package invoice

import "testing"

func TestLineItemAmount(t *testing.T) {
	tests := []struct {
		name string
		item LineItem
		want int64
	}{
		{name: "one item", item: LineItem{Quantity: 1, UnitPrice: 500}, want: 500},
		{name: "several items", item: LineItem{Quantity: 3, UnitPrice: 120}, want: 360},
		{name: "free item", item: LineItem{Quantity: 2, UnitPrice: 0}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.Amount()
			if got != tt.want {
				t.Errorf("Amount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCalculateTotals(t *testing.T) {
	// Part 1: the SHAPE of one row (what each test case contains)
	tests := []struct {
		name         string
		items        []LineItem
		wantSubtotal int64
		wantTax      int64
		wantTotal    int64
	}{
		// Part 2: the ROWS (the actual test data), one {...} per case
		{name: "single 10% item", items: []LineItem{
			{Quantity: 1, UnitPrice: 1000, TaxRate: 10},
		},
			wantSubtotal: 1000,
			wantTax:      100,
			wantTotal:    1100,
		},
		{name: "mixed rates", items: []LineItem{
			{Quantity: 1, UnitPrice: 1000, TaxRate: 8},
			{Quantity: 1, UnitPrice: 1000, TaxRate: 10},
		},
			wantSubtotal: 2000,
			wantTax:      180,
			wantTotal:    2180,
		},
		{name: "rounds once per rate", items: []LineItem{
			{Quantity: 1, UnitPrice: 105, TaxRate: 10},
			{Quantity: 1, UnitPrice: 105, TaxRate: 10},
		},
			wantSubtotal: 210,
			wantTax:      21,
			wantTotal:    231,
		},
	}
	// Part 3: the LOOP that runs every row the same way
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invoice{Items: tt.items}
			inv.CalculateTotals()

			if inv.Subtotal != tt.wantSubtotal {
				t.Errorf("Subtotal = %d, want %d", inv.Subtotal, tt.wantSubtotal)
			}
			if inv.Tax != tt.wantTax {
				t.Errorf("Tax = %d, want %d", inv.Tax, tt.wantTax)
			}
			if inv.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", inv.Total, tt.wantTotal)
			}
		})
	}
}
