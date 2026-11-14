package invoice

// LineItem is one row on an invoice
type LineItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"` // yen, never float
	TaxRate     int    `json:"tax_rate"`   // 10 or 8 (percent)
}

// Amount is the line total before tax: quantity x unit price.
func (li LineItem) Amount() int64 {
	return int64(li.Quantity) * li.UnitPrice
}

// CalculateTotals fills in Subtotal, Tax and Total.
// Tax is rounded down once per tax rate, as required for qualified invoices in Japan.
func (inv *Invoice) CalculateTotals() {
	byRate := map[int]int64{}
	for _, item := range inv.Items {
		byRate[item.TaxRate] += item.Amount()
	}
	var subtotal int64
	var tax int64
	for rate, sum := range byRate {
		subtotal += sum
		tax += sum * int64(rate) / 100
	}
	inv.Subtotal = subtotal
	inv.Tax = tax
	inv.Total = subtotal + tax
}

// Invoice is a bill from a vendor
type Invoice struct {
	ID        string     `json:"id"`
	Vendor    string     `json:"vendor"`
	IssueDate string     `json:"issue_date"`
	DueDate   string     `json:"due_date"`
	Items     []LineItem `json:"items"`
	Subtotal  int64      `json:"subtotal"`
	Tax       int64      `json:"tax"`
	Total     int64      `json:"total"`
}
