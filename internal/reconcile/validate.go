package reconcile

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTransactions is returned when a reconcile request has bad input.
var ErrInvalidTransactions = errors.New("invalid transactions")

// MaxTransactions is the most transactions one request may send.
const MaxTransactions = 1000

// Validate checks the transactions before any matching runs.
// Every error wraps ErrInvalidTransactions.
func Validate(txns []Transaction) error {
	if len(txns) < 1 || len(txns) > MaxTransactions {
		return fmt.Errorf("%w: send between 1 and %d transactions", ErrInvalidTransactions, MaxTransactions)
	}

	seen := map[string]bool{}
	for i, tx := range txns {
		if tx.ID == "" {
			return fmt.Errorf("%w: transaction %d: id is required", ErrInvalidTransactions, i)
		}
		if seen[tx.ID] {
			return fmt.Errorf("%w: transaction %d: duplicate id %q", ErrInvalidTransactions, i, tx.ID)
		}
		seen[tx.ID] = true
		if tx.Amount <= 0 {
			return fmt.Errorf("%w: transaction %d: amount must be above 0", ErrInvalidTransactions, i)
		}
		if _, err := time.Parse("2006-01-02", tx.Date); err != nil {
			return fmt.Errorf("%w: transaction %d: date must look like 2026-11-28", ErrInvalidTransactions, i)
		}
	}

	return nil
}
