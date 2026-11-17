package invoice

import (
	"slices"
	"testing"
)

func TestMemoryStoreList(t *testing.T) {
	// Four invoices. Create gives them IDs inv_1 to inv_4 in this order.
	//
	//   inv_1  ACME     due 2026-11-30
	//   inv_2  Example  due 2026-11-30   (same date as inv_1: the ID breaks the tie)
	//   inv_3  ACME     due 2026-12-15
	//   inv_4  Example  due 2026-11-20
	s := NewMemoryStore()
	for _, inv := range []Invoice{
		{Vendor: "ACME", DueDate: "2026-11-30"},
		{Vendor: "Example", DueDate: "2026-11-30"},
		{Vendor: "ACME", DueDate: "2026-12-15"},
		{Vendor: "Example", DueDate: "2026-11-20"},
	} {
		inv.Items = []LineItem{{Description: "Paper", Quantity: 1, UnitPrice: 1000, TaxRate: 10}}
		if _, err := s.Create(t.Context(), inv); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name   string
		filter ListFilter
		want   []string // invoice IDs, in the order List must return them
	}{
		{
			name:   "no filter returns all, sorted by due date then ID",
			filter: ListFilter{},
			want:   []string{"inv_4", "inv_1", "inv_2", "inv_3"},
		},
		{
			name:   "vendor",
			filter: ListFilter{Vendor: "ACME"},
			want:   []string{"inv_1", "inv_3"},
		},
		{
			name:   "due before keeps only earlier invoices",
			filter: ListFilter{DueBefore: "2026-12-01"},
			want:   []string{"inv_4", "inv_1", "inv_2"},
		},
		{
			name:   "limit",
			filter: ListFilter{Limit: 2},
			want:   []string{"inv_4", "inv_1"},
		},
		{
			name:   "cursor in the middle of a tie",
			filter: ListFilter{AfterDueDate: "2026-11-30", AfterID: "inv_1"},
			want:   []string{"inv_2", "inv_3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.List(t.Context(), tt.filter)
			if err != nil {
				t.Fatal(err)
			}

			var ids []string
			for _, inv := range got {
				ids = append(ids, inv.ID)
				if inv.Items != nil {
					t.Errorf("%s has items; List must return summaries only", inv.ID)
				}
			}
			if !slices.Equal(ids, tt.want) {
				t.Errorf("got %v, want %v", ids, tt.want)
			}
		})
	}
}
