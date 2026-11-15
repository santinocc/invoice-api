package invoice

import (
	"errors"
	"testing"
)

func TestMemoryStoreCreateAndGet(t *testing.T) {
	s := NewMemoryStore()

	created, err := s.Create(validInvoice())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() did not set and ID")
	}

	got, err := s.Get(created.ID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", created.ID, err)
	}
	if got.Vendor != created.Vendor {
		t.Errorf("Get() vendor = %q, want %q", got.Vendor, created.Vendor)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Get("inv_999")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}

}
