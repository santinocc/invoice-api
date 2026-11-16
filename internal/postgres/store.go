// Package postgres stores invoices in PostgreSQL.
//
// It depends on the invoice package, never the other way around:
// the domain does not know which database it is saved in.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql

	"github.com/santinocc/invoice-api/internal/invoice"
)

// Open connects to PostgreSQL and checks that the connection works.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return db, nil
}

// Store saves and loads invoices in PostgreSQL.
// It implements invoice.Store.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store that uses db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create saves the invoice and its items in one transaction
// and returns the invoice with the ID that Postgres generated.
func (s *Store) Create(ctx context.Context, inv invoice.Invoice) (invoice.Invoice, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return invoice.Invoice{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // does nothing if Commit already succeeded

	err = tx.QueryRowContext(ctx,
		`INSERT INTO invoices (vendor, issue_date, due_date, subtotal, tax, total)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		inv.Vendor, inv.IssueDate, inv.DueDate, inv.Subtotal, inv.Tax, inv.Total,
	).Scan(&inv.ID)
	if err != nil {
		return invoice.Invoice{}, fmt.Errorf("insert invoice: %w", err)
	}

	for i, item := range inv.Items {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO invoice_items (invoice_id, position, description, quantity, unit_price, tax_rate)
		VALUES ($1, $2, $3, $4, $5, $6)`,
			inv.ID, i, item.Description, item.Quantity, item.UnitPrice, item.TaxRate,
		)
		if err != nil {
			return invoice.Invoice{},
				fmt.Errorf("insert item %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return invoice.Invoice{}, fmt.Errorf("commit: %w", err)
	}
	return inv, nil
}
