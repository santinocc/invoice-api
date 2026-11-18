package invoice

import (
	"slices"
	"testing"
)

func TestMemoryStoreFindByTotals(t *testing.T) {
	s := NewMemoryStore()
	for _, inv := range []Invoice{
		{Vendor: "ACME", DueDate: "2026-12-15", Total: 5400},  // inv_1
		{Vendor: "Kanto", DueDate: "2026-11-30", Total: 5400}, // inv_2
		{Vendor: "ACME", DueDate: "2026-11-30", Total: 2266},  // inv_3
		{Vendor: "ACME", DueDate: "2026-11-30", Total: 1100},  // inv_4
	} {
		inv.Items = []LineItem{{Description: "Paper", Quantity: 1, UnitPrice: 1000, TaxRate: 10}}
		if _, err := s.Create(t.Context(), inv); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name   string
		totals []int64
		want   []string
	}{
		{"two totals, sorted by due date then ID", []int64{5400, 2266}, []string{"inv_2", "inv_3", "inv_1"}},
		{"a total nobody has", []int64{999}, nil},
		{"no totals", nil, nil},
		{"the same total twice", []int64{1100, 1100}, []string{"inv_4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.FindByTotals(t.Context(), tt.totals)
			if err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, inv := range got {
				ids = append(ids, inv.ID)
				if inv.Items != nil {
					t.Errorf("%s has items; FindByTotals must return summaries only", inv.ID)
				}
			}
			if !slices.Equal(ids, tt.want) {
				t.Errorf("got %v, want %v", ids, tt.want)
			}
		})
	}
}
