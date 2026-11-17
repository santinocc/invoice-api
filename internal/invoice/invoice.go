package invoice

import (
	"errors"
	"fmt"
	"time"
)

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

// ErrInvalidInvoice is returned when an invoice fails validation.
var ErrInvalidInvoice = errors.New("invalid invoice")

const dateLayout = "2006-01-02"

// Validate checks that the invoice has the fields and values it needs.
func (inv Invoice) Validate() error {
	if inv.Vendor == "" {
		return fmt.Errorf("%w: vendor is required", ErrInvalidInvoice)
	}

	issue, err := time.Parse(dateLayout, inv.IssueDate)
	if err != nil {
		return fmt.Errorf("%w: issue_date must be YYYY-MM-DD", ErrInvalidInvoice)
	}

	var due time.Time
	due, err = time.Parse(dateLayout, inv.DueDate)
	if err != nil {
		return fmt.Errorf("%w: due_date must be YYYY-MM-DD", ErrInvalidInvoice)
	}

	if due.Before(issue) {
		return fmt.Errorf("%w: due_date is before issue_date", ErrInvalidInvoice)
	}

	if len(inv.Items) == 0 {
		return fmt.Errorf("%w: at least one item is required", ErrInvalidInvoice)
	}

	for _, item := range inv.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: item quantity must be positive", ErrInvalidInvoice)
		}
		if item.UnitPrice < 0 {
			return fmt.Errorf("%w: item unit_price cannot be negative", ErrInvalidInvoice)
		}
		if item.TaxRate != 8 && item.TaxRate != 10 {
			return fmt.Errorf("%w: item tax_rate must be 8 or 10", ErrInvalidInvoice)
		}
	}

	return nil
}

// Invoice is a bill from a vendor
type Invoice struct {
	ID        string     `json:"id"`
	Vendor    string     `json:"vendor"`
	IssueDate string     `json:"issue_date"`
	DueDate   string     `json:"due_date"`
	Items     []LineItem `json:"items,omitempty"` // empty in list results (summaries)
	Subtotal  int64      `json:"subtotal"`
	Tax       int64      `json:"tax"`
	Total     int64      `json:"total"`
}
