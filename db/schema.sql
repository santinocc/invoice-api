CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor TEXT NOT NULL CHECK (vendor <> ''),
    issue_date DATE NOT NULL,
    due_date DATE NOT NULL,
    subtotal BIGINT NOT NULL,
    tax BIGINT NOT NULL,
    total BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT due_not_before_issue CHECK (due_date >= issue_date)
);

CREATE TABLE invoice_items (
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    position INT NOT NULL,
    description TEXT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price BIGINT NOT NULL CHECK (unit_price >= 0),
    tax_rate SMALLINT NOT NULL CHECK (tax_rate IN (8, 10)),
    PRIMARY KEY (invoice_id, position)
);

-- List invoices sorted by due date, paged with a (due_date, id) cursor.
CREATE INDEX invoices_due_date_id ON invoices (due_date, id);

-- The same list filtered by one vendor. Vendor comes first: the leftmost-prefix rule.
CREATE INDEX invoices_vendor_due_date_id ON invoices (vendor, due_date, id);

-- Reconciliation looks up open invoices by exact total: WHERE total = ANY($1).
CREATE INDEX invoices_total ON invoices (total);
