// Package postgres stores invoices in PostgreSQL.
//
// It depends on the invoice package, never the other way around:
// the domain does not know which database it is saved in.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

// This line does not run anything: it makes the build fail
// if Store ever stops satisfying the invoice.Store interface.
var _ invoice.Store = (*Store)(nil)

// invalidTextRepresentation is the Postgres error code for a value that
// cannot be read as its column's type, such as "inv_999" for a UUID column.
const invalidTextRepresentation = "22P02"

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

// Get returns the invoice with the given ID and its items, or invoice.ErrNotFound
func (s *Store) Get(ctx context.Context, id string) (invoice.Invoice, error) {
	var inv invoice.Invoice
	err := s.db.QueryRowContext(ctx,
		`SELECT id, vendor, issue_date::text, due_date::text, subtotal, tax, total
	FROM invoices
	WHERE id = $1`,
		id,
	).Scan(&inv.ID, &inv.Vendor, &inv.IssueDate, &inv.DueDate, &inv.Subtotal, &inv.Tax, &inv.Total)
	if errors.Is(err, sql.ErrNoRows) {
		return invoice.Invoice{}, invoice.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return invoice.Invoice{}, invoice.ErrNotFound
	}
	if err != nil {
		return invoice.Invoice{}, fmt.Errorf("select invoice: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT description, quantity, unit_price, tax_rate
FROM invoice_items
WHERE invoice_id = $1
ORDER BY position`,
		id,
	)
	if err != nil {
		return invoice.Invoice{}, fmt.Errorf("select items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item invoice.LineItem
		if err := rows.Scan(&item.Description, &item.Quantity, &item.UnitPrice, &item.TaxRate); err != nil {
			return invoice.Invoice{},
				fmt.Errorf("scan item: %w", err)
		}
		inv.Items = append(inv.Items, item)
	}
	if err := rows.Err(); err != nil {
		return invoice.Invoice{}, fmt.Errorf("read items: %w", err)
	}

	return inv, nil
}

// List returns summaries (no items) of the invoices that match f,
// sorted by due date, then ID.
func (s *Store) List(ctx context.Context, f invoice.ListFilter) ([]invoice.Invoice, error) {
	var conds []string // SQL conditions, joined with AND
	var args []any     // their values, in the order of $1, $2, ...

	if f.Vendor != "" {
		args = append(args, f.Vendor)
		conds = append(conds, fmt.Sprintf("vendor = $%d", len(args)))
	}
	// due-before filter
	if f.DueBefore != "" {
		args = append(args, f.DueBefore)
		conds = append(conds, fmt.Sprintf("due_date < $%d", len(args)))
	}

	// cursor: a row comparison, so ID breaks ties between equal due dates
	if f.AfterDueDate != "" {
		args = append(args, f.AfterDueDate, f.AfterID)
		conds = append(conds, fmt.Sprintf("(due_date, id) > ($%d, $%d)", len(args)-1, len(args)))
	}

	query := `SELECT id, vendor, issue_date::text, due_date::text, subtotal, tax, total FROM invoices`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY due_date, id"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		// A cursor ID that is not a UUID: the client changed the cursor.
		return nil, fmt.Errorf("%w: bad cursor", invoice.ErrInvalidFilter)
	}
	if err != nil {
		return nil, fmt.Errorf("select invoices: %w", err)
	}
	defer rows.Close()

	var result []invoice.Invoice
	for rows.Next() {
		var inv invoice.Invoice
		if err := rows.Scan(&inv.ID, &inv.Vendor, &inv.IssueDate, &inv.DueDate, &inv.Subtotal, &inv.Tax, &inv.Total); err != nil {
			return nil,
				fmt.Errorf("scan invoice: %w", err)
		}
		result = append(result, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read invoices: %w", err)
	}

	return result, nil
}
