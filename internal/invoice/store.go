package invoice

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrNotFound is returned when no invoice has the given ID.
var ErrNotFound = errors.New("invoice not found")

// Store saves and loads invoices.
type Store interface {
	Create(ctx context.Context, inv Invoice) (Invoice, error)
	Get(ctx context.Context, id string) (Invoice, error)
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
