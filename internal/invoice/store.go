package invoice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// ErrNotFound is returned when no invoice has the given ID.
var ErrNotFound = errors.New("invoice not found")

// ErrInvalidFilter is returned when a ListFilter has a bad value,
// such as a malformed date, a limit out of range, or a tampered cursor.
var ErrInvalidFilter = errors.New("invalid filter")

// MaxListLimit is the largest page List may be asked for.
const MaxListLimit = 100

// Store saves and loads invoices.
type Store interface {
	Create(ctx context.Context, inv Invoice) (Invoice, error)
	Get(ctx context.Context, id string) (Invoice, error)
	List(ctx context.Context, f ListFilter) ([]Invoice, error)
}

// ListFilter selects which invoices List returns. Empty fields match everything.
//
// Results are sorted by due date, then ID. AfterDueDate and AfterID form the
// cursor: only invoices that sort after that pair are returned.
type ListFilter struct {
	Vendor       string // exact match
	DueBefore    string // "2006-01-02"; only invoices due strictly before this date
	AfterDueDate string // cursor: due date of the last invoice on the previous page
	AfterID      string // cursor: ID of the last invoice on the previous page
	Limit        int    // maximum number of invoices to return
}

// MemoryStore keeps invoices in a map. It is safe for concurrent use.
type MemoryStore struct {
	mu       sync.Mutex
	invoices map[string]Invoice
	nextID   int
}

// NewMemoryStore returns an empty MemoryStore ready to use.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{invoices: map[string]Invoice{}}
}

// Create gives the invoice a new ID, saves it, and returns it.
func (s *MemoryStore) Create(ctx context.Context, inv Invoice) (Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	inv.ID = fmt.Sprintf("inv_%d", s.nextID)
	s.invoices[inv.ID] = inv
	return inv, nil
}

// Get returns the invoice with the given ID, or ErrNotFound.
func (s *MemoryStore) Get(ctx context.Context, id string) (Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]

	if !ok {
		return Invoice{}, ErrNotFound
	}

	return inv, nil
}

// List returns summaries (no items) of the invoices that match f,
// sorted by due date, then ID.
func (s *MemoryStore) List(ctx context.Context, f ListFilter) ([]Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []Invoice
	for _, inv := range s.invoices {
		// skip this invoice if a vendor filter is set and it does not match
		if f.Vendor != "" && inv.Vendor != f.Vendor {
			continue
		}
		// skip this invoice if a due-before filter is set and inv is not due before it
		if f.DueBefore != "" && inv.DueDate >= f.DueBefore {
			continue
		}
		// cursor: only invoices that sort after the last one on the previous page.
		if f.AfterDueDate != "" && cmp.Or(cmp.Compare(inv.DueDate, f.AfterDueDate), cmp.Compare(inv.ID, f.AfterID)) <= 0 {
			continue
		}
		inv.Items = nil // summaries only
		result = append(result, inv)
	}

	slices.SortFunc(result, func(a, b Invoice) int {
		return cmp.Or(cmp.Compare(a.DueDate, b.DueDate), cmp.Compare(a.ID, b.ID))
	})

	// if f.Limit is above 0 and result is longer than f.Limit, keep only the first f.Limit
	if f.Limit > 0 && len(result) > f.Limit {
		result = result[:f.Limit]
	}
	return result, nil
}

// Validate checks the filter's dates and limit.
// Every error wraps ErrInvalidFilter.
func (f ListFilter) Validate() error {
	if f.Limit < 1 || f.Limit > MaxListLimit {
		return fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidFilter, MaxListLimit)
	}

	if f.DueBefore != "" {
		if _, err := time.Parse(dateLayout, f.DueBefore); err != nil {
			return fmt.Errorf("%w: due_before must be a date like 2026-12-31", ErrInvalidFilter)
		}
	}

	if f.AfterDueDate != "" {
		if _, err := time.Parse(dateLayout, f.AfterDueDate); err != nil {
			return fmt.Errorf("%w: bad cursor", ErrInvalidFilter)
		}
	}

	return nil
}
