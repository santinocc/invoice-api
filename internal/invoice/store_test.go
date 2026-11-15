package invoice

import (
	"errors"
	"sync"
	"testing"
)

func TestMemoryStoreCreateAndGet(t *testing.T) {
	s := NewMemoryStore()

	created, err := s.Create(t.Context(), validInvoice())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() did not set an ID")
	}

	got, err := s.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", created.ID, err)
	}
	if got.Vendor != created.Vendor {
		t.Errorf("Get() vendor = %q, want %q", got.Vendor, created.Vendor)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Get(t.Context(), "inv_999")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}

}

func TestMemoryStoreConcurrentCreate(t *testing.T) {
	s := NewMemoryStore()
	const n = 100

	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Create(t.Context(), validInvoice()); err != nil {
				t.Errorf("Create() error = %v", err)
			}
		}()
	}
	wg.Wait()

	if len(s.invoices) != n {
		t.Errorf("stored %d invoices, want %d", len(s.invoices), n)
	}
}
