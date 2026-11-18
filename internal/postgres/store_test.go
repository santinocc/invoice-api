package postgres

import (
	"cmp"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// newTestStore connects to the database in TEST_DATABASE_URL and empties it,
// so every test starts from a known, empty state (isolation by truncation).
//
// It skips the test when TEST_DATABASE_URL is not set. These tests delete
// data, so they never read DATABASE_URL: running them must be a deliberate choice.
//
// Tests in this package must not call t.Parallel: one test's TRUNCATE
// would wipe another test's rows while it runs.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres tests")
	}

	db, err := Open(t.Context(), url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.ExecContext(t.Context(), "TRUNCATE invoices CASCADE"); err != nil {
		t.Fatalf("empty test database: %v", err)
	}
	return NewStore(db)
}

// testInvoice returns a valid invoice with two items at different tax rates.
func testInvoice(vendor, dueDate string) invoice.Invoice {
	inv := invoice.Invoice{
		Vendor:    vendor,
		IssueDate: "2026-11-01",
		DueDate:   dueDate,
		Items: []invoice.LineItem{
			{Description: "Rice", Quantity: 2, UnitPrice: 540, TaxRate: 8},
			{Description: "Paper", Quantity: 1, UnitPrice: 1000, TaxRate: 10},
		},
	}
	inv.CalculateTotals()
	return inv
}

func TestStoreCreateAndGet(t *testing.T) {
	s := newTestStore(t)

	created, err := s.Create(t.Context(), testInvoice("Example Supplies", "2026-11-30"))
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
	if got.Total != 2266 {
		t.Errorf("Total = %d, want 2266", got.Total)
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Description != "Rice" {
		t.Errorf("Description = %q, want Rice", got.Items[0].Description)
	}
}

func TestStoreGetNotFound(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name string
		id   string
	}{
		{name: "valid UUID that does not exist", id: "00000000-0000-0000-0000-000000000000"},
		{name: "not a UUID", id: "inv_999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Get(t.Context(), tt.id)
			if !errors.Is(err, invoice.ErrNotFound) {
				t.Errorf("Get() error = %v, want ErrNotFound", err)
			}
		})
	}

}

// TestStoreCreateRollsBack checks that Create is all-or-nothing: when an item
// fails a database CHECK, the invoice row inserted just before it is undone too.
func TestStoreCreateRollsBack(t *testing.T) {
	s := newTestStore(t)

	inv := testInvoice("Example Supplies", "2026-11-30")
	inv.Items[1].TaxRate = 5 // Validate would reject this; here it reaches the database's CHECK

	if _, err := s.Create(t.Context(), inv); err == nil {
		t.Fatal("Create() succeeded with tax rate 5, want an error")
	}

	var count int
	if err := s.db.QueryRowContext(t.Context(), "SELECT count(*) FROM invoices").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("invoices after a failed Create = %d, want 0 (the transaction did not roll back)", count)
	}
}

// TestStoreList checks the same filters and cursor as the memory store's test.
// Postgres IDs are random UUIDs, so the test checks order and counts, then
// pages through with the cursor and compares the pages with the full list.
func TestStoreList(t *testing.T) {
	s := newTestStore(t)
	for _, inv := range []invoice.Invoice{
		testInvoice("ACME", "2026-11-30"),
		testInvoice("Example", "2026-11-30"), // same due date: the ID breaks the tie
		testInvoice("ACME", "2026-12-15"),
		testInvoice("Example", "2026-11-20"),
	} {
		if _, err := s.Create(t.Context(), inv); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.List(t.Context(), invoice.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("len(all) = %d, want 4", len(all))
	}
	byDueDateThenID := func(a, b invoice.Invoice) int {
		return cmp.Or(cmp.Compare(a.DueDate, b.DueDate), cmp.Compare(a.ID, b.ID))
	}
	if !slices.IsSortedFunc(all, byDueDateThenID) {
		t.Errorf("List is not sorted by due date, then ID: %v", all)
	}
	for _, inv := range all {
		if inv.Items != nil {
			t.Errorf("%s has items; List must return summaries only", inv.ID)
		}
	}

	counts := []struct {
		name   string
		filter invoice.ListFilter
		want   int
	}{
		{"vendor", invoice.ListFilter{Vendor: "ACME"}, 2},
		{"due before", invoice.ListFilter{DueBefore: "2026-12-01"}, 3},
		{"vendor and due before", invoice.ListFilter{Vendor: "ACME", DueBefore: "2026-12-01"}, 1},
		{"limit", invoice.ListFilter{Limit: 2}, 2},
	}
	for _, tt := range counts {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.List(t.Context(), tt.filter)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tt.want {
				t.Errorf("got %d invoices, want %d", len(got), tt.want)
			}
		})
	}

	t.Run("cursor pages add up to the full list", func(t *testing.T) {
		first, err := s.List(t.Context(), invoice.ListFilter{Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		last := first[len(first)-1]
		rest, err := s.List(t.Context(), invoice.ListFilter{AfterDueDate: last.DueDate, AfterID: last.ID})
		if err != nil {
			t.Fatal(err)
		}
		if paged := append(first, rest...); !slices.Equal(ids(paged), ids(all)) {
			t.Errorf("pages = %v, want %v", ids(paged), ids(all))
		}
	})

	t.Run("cursor ID that is not a UUID", func(t *testing.T) {
		_, err := s.List(t.Context(), invoice.ListFilter{AfterDueDate: "2026-11-30", AfterID: "inv_1"})
		if !errors.Is(err, invoice.ErrInvalidFilter) {
			t.Errorf("err = %v, want ErrInvalidFilter", err)
		}
	})
}

func ids(invs []invoice.Invoice) []string {
	var out []string
	for _, inv := range invs {
		out = append(out, inv.ID)
	}
	return out
}

func TestStoreFindByTotals(t *testing.T) {
	s := newTestStore(t)
	var created []invoice.Invoice
	for _, due := range []string{"2026-12-15", "2026-11-30"} { // both total 2266
		inv, err := s.Create(t.Context(), testInvoice("ACME", due))
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, inv)
	}

	got, err := s.FindByTotals(t.Context(), []int64{2266, 999})
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by due date: the second invoice created is due first.
	want := []string{created[1].ID, created[0].ID}
	if !slices.Equal(ids(got), want) {
		t.Errorf("got %v, want %v", ids(got), want)
	}
	for _, inv := range got {
		if inv.Items != nil {
			t.Errorf("%s has items; FindByTotals must return summaries only", inv.ID)
		}
	}

	none, err := s.FindByTotals(t.Context(), []int64{999})
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Errorf("total 999: got %d invoices, want 0", len(none))
	}
}
