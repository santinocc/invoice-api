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
