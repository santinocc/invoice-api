// Package reconcile suggests which bank transaction pays which invoice.
//
// It only suggests: nothing is saved, so calling it twice is harmless
// (idempotent). A person or another system confirms the matches.
// It imports invoice; invoice does not know about reconcile.
package reconcile

import "github.com/santinocc/invoice-api/internal/invoice"

// Transaction is one incoming payment from a bank statement.
type Transaction struct {
	ID     string `json:"id"`
	Date   string `json:"date"`   // "2006-01-02"
	Amount int64  `json:"amount"` // yen
	Payer  string `json:"payer"`  // the name the bank shows
}

// Match pairs a transaction with the one invoice it pays.
type Match struct {
	TransactionID string `json:"transaction_id"`
	InvoiceID     string `json:"invoice_id"`
}

// Ambiguity is a transaction that more than one invoice could explain.
// It is left for a person to decide, never guessed.
type Ambiguity struct {
	TransactionID string   `json:"transaction_id"`
	InvoiceIDs    []string `json:"invoice_ids"`
}

// Result sorts every transaction into exactly one of three lists.
type Result struct {
	Matched   []Match     `json:"matched"`
	Ambiguous []Ambiguity `json:"ambiguous"`
	Unmatched []string    `json:"unmatched"` // transaction IDs
}

// Suggest applies the matching rules to the transactions, using the given
// invoices as possible matches:
//
//  1. Candidates are invoices whose Total equals the transaction Amount.
//  2. If there is more than one candidate and some have a Vendor equal to
//     the Payer, keep only those. If none do, keep them all.
//  3. Exactly one candidate: matched. None: unmatched. More: ambiguous.
//
// It runs in O(n + m) for n transactions and m invoices, using a map from
// amount to invoices instead of comparing every pair.
func Suggest(txns []Transaction, invoices []invoice.Invoice) Result {
	var res Result

	byAmount := map[int64][]invoice.Invoice{}

	for _, inv := range invoices {
		byAmount[inv.Total] = append(byAmount[inv.Total], inv)
	}
	for _, tx := range txns {
		candidates := byAmount[tx.Amount]
		if len(candidates) > 1 {
			var same []invoice.Invoice
			for _, inv := range candidates {
				if inv.Vendor == tx.Payer {
					same = append(same, inv)
				}
			}
			if len(same) > 0 {
				candidates = same
			}
		}
		switch len(candidates) {
		case 0:
			res.Unmatched = append(res.Unmatched, tx.ID)
		case 1:
			res.Matched = append(res.Matched, Match{TransactionID: tx.ID, InvoiceID: candidates[0].ID})
		default:
			var ids []string
			for _, inv := range candidates {
				ids = append(ids, inv.ID)
			}
			res.Ambiguous = append(res.Ambiguous, Ambiguity{TransactionID: tx.ID, InvoiceIDs: ids})
		}
	}
	return res
}
