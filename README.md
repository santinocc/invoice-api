# invoice-api

A small invoice API written in Go, built as a learning project. It stores invoices in PostgreSQL (or in memory), calculates Japanese consumption tax (10% standard, 8% reduced), lists invoices with filters and cursor pagination, and suggests which bank payment pays which invoice (reconciliation, 照合).

## Status

Stages 1 to 3 are finished. This list shows only what is actually done.

- [x] `POST /invoices`: validates the invoice, calculates totals, saves it, returns `201`
- [x] `GET /invoices/{id}`: returns the invoice with its items, or `404`
- [x] `GET /invoices`: lists invoice summaries, filtered by vendor and due date, with cursor pagination
- [x] `POST /reconcile`: suggests matches between bank transactions and invoices; saves nothing
- [x] `GET /healthz`
- [x] Tax calculated per tax rate and rounded down once per rate, not per line
- [x] Two stores behind one `Store` interface: PostgreSQL when `DATABASE_URL` is set, in memory otherwise
- [x] PostgreSQL: an invoice and its items are saved in one transaction; indexes for the list queries and for lookups by total
- [x] Tests for the domain, both stores and the HTTP handlers. The PostgreSQL tests run against a separate test database.
- [x] CI on GitHub Actions: `gofmt`, `go vet`, `go test -race`, with a PostgreSQL service so the database tests run on every push

Not done yet:

- [ ] Stage 4: linting with `golangci-lint`, Dockerfile, graceful shutdown
- [ ] Schema migrations. `db/schema.sql` only runs when the database volume is empty, so a schema change means resetting the local database (see below).
- [ ] Updating or deleting invoices
- [ ] Applying matches: invoices have no "paid" status yet, so a paid invoice can be suggested again
- [ ] Fuzzy matching: amounts must be equal, so a payment that arrives minus the bank transfer fee (振込手数料) is unmatched

## Requirements

- Go 1.27 or newer (the exact version is in `go.mod`)
- Docker, only to run PostgreSQL. Without it the server uses the in-memory store.

## Run

**In memory** (invoices are lost when the server stops):

```sh
go run ./cmd/server
```

**With PostgreSQL:**

```sh
docker compose up -d --wait
DATABASE_URL='postgres://invoice:invoice@localhost:5432/invoice?sslmode=disable' go run ./cmd/server
```

The first `docker compose up` creates two databases: `invoice` for the server and `invoice_test` for the tests. Both get the schema from `db/schema.sql`. The password in `docker-compose.yml` is for local development only.

If `DATABASE_URL` is set but PostgreSQL cannot be reached, the server exits with an error. It never falls back to memory silently, because that would accept invoices and then lose them.

The server listens on port `8080`. Set `PORT` to change it. All variables are listed in `.env.example`.

To reset the local database, for example after a change to `db/schema.sql` (this **deletes all local data**):

```sh
docker compose down -v
docker compose up -d --wait
```

## Use

Create an invoice:

```sh
curl -i -X POST localhost:8080/invoices -d '{
  "vendor": "Example Supplies",
  "issue_date": "2026-11-01",
  "due_date": "2026-11-30",
  "items": [
    {"description": "Rice",  "quantity": 2, "unit_price": 540,  "tax_rate": 8},
    {"description": "Paper", "quantity": 1, "unit_price": 1000, "tax_rate": 10}
  ]
}'
```

Response (`201 Created`):

```json
{"id":"342d3e97-d674-4f17-b2a7-eb3b41c5207f","vendor":"Example Supplies","issue_date":"2026-11-01","due_date":"2026-11-30","items":[...],"subtotal":2080,"tax":186,"total":2266}
```

PostgreSQL gives invoices UUIDs. The in-memory store uses `inv_1`, `inv_2` and so on.

Get it back:

```sh
curl -i localhost:8080/invoices/342d3e97-d674-4f17-b2a7-eb3b41c5207f
```

List invoices due before a date, two per page:

```sh
curl -i 'localhost:8080/invoices?due_before=2026-12-31&limit=2'
```

```json
{"invoices":[{"id":"...","vendor":"Example Supplies","due_date":"2026-11-20",...},{"id":"...","due_date":"2026-11-30",...}],"next_cursor":"MjAyNi0xMS0zMHwz..."}
```

| Query parameter | Meaning |
|---|---|
| `vendor` | Only this vendor (exact match) |
| `due_before` | Only invoices due before this date (`YYYY-MM-DD`) |
| `limit` | Page size, 1 to 100. Default 20. |
| `cursor` | The `next_cursor` from the previous page |

The list returns summaries without items, sorted by due date, then ID. When more invoices exist, the response includes `next_cursor`. Pass it back unchanged as `cursor` to get the next page. When it is missing, you have the last page. Pages stay correct when new invoices are added in between, because the cursor marks a position in the sort order, not a row count.

Suggest which payment pays which invoice:

```sh
curl -i -X POST localhost:8080/reconcile -d '{"transactions": [
  {"id": "tx_1", "date": "2026-11-28", "amount": 2266, "payer": "Example Supplies"},
  {"id": "tx_2", "date": "2026-11-29", "amount": 999,  "payer": "Nobody"}
]}'
```

```json
{"matched":[{"transaction_id":"tx_1","invoice_id":"342d3e97-..."}],"ambiguous":[],"unmatched":["tx_2"]}
```

Every transaction ends up in exactly one list:

1. The candidates are invoices whose `total` equals the `amount` exactly.
2. If there are several candidates and some have a `vendor` equal to the `payer`, only those are kept. The payer name only breaks ties, because bank payer names are often written differently (for example in half-width katakana).
3. One candidate: `matched`. None: `unmatched`. Several: `ambiguous`, with the candidate invoice IDs, for a person to decide.
4. If two or more transactions would match the same invoice, they are all `ambiguous`, whatever their order in the request. A transaction that is already ambiguous does not count as a claim, so it does not stop another transaction from matching one of its candidates.

The endpoint saves nothing, so sending the same request twice is harmless. A request takes 1 to 1000 transactions, each with a unique `id`, an `amount` above 0 and a `date` (`YYYY-MM-DD`). The server asks the database only for invoices whose total equals one of the amounts (`WHERE total = ANY($1)`), then matches them in Go with a map from amount to invoices, in O(n + m) time.

Errors use the matching status code and a JSON body such as `{"error":"invalid invoice: vendor is required"}`:

| Status | When |
|---|---|
| `400` | The body is not valid JSON, has unknown fields, or fails validation; a list parameter or cursor is invalid; or a reconcile transaction is invalid |
| `404` | No invoice has that ID |
| `500` | The store failed |

## Rules

- Amounts are whole yen stored as `int64`. Floating point is never used for money.
- Dates use the format `YYYY-MM-DD`. The due date cannot be before the issue date.
- Each item needs a quantity above 0, a unit price of 0 or more, and a tax rate of 8 or 10.
- Tax is calculated on the total of each tax rate and rounded down once per rate, as required for qualified invoices (適格請求書) in Japan. Rounding each line separately can give a different result.
- The client cannot set `id`, `subtotal`, `tax` or `total`; the server overwrites them.
- PostgreSQL also enforces the main rules with `CHECK` constraints, as a second line of defence.

## Test

```sh
go test -race ./...
```

Without `TEST_DATABASE_URL` the PostgreSQL tests are skipped. To run them, start PostgreSQL with `docker compose up -d --wait`, then:

```sh
TEST_DATABASE_URL='postgres://invoice:invoice@localhost:5432/invoice_test?sslmode=disable' go test -race ./...
```

These tests **empty the tables** before each test. That is why they read `TEST_DATABASE_URL` and never `DATABASE_URL`, and why they use the separate `invoice_test` database. In CI the build fails if any test is skipped, so the PostgreSQL tests cannot stop running unnoticed.

## Project layout

```
cmd/server/         entry point: reads the environment, picks the store, starts the server
internal/invoice/   domain: types, validation, tax calculation, Store interface, in-memory store
internal/postgres/  PostgreSQL store (database/sql with the pgx driver)
internal/reconcile/ matching rules for bank transactions and invoices (no storage, no HTTP)
internal/httpapi/   HTTP: routes, JSON handlers, list cursor, error responses
db/                 schema.sql and the script that creates the test database
```

`internal/postgres` and `internal/reconcile` import `internal/invoice`, never the other way round: the domain does not know where it is stored or how payments are matched.

## License

MIT. See [LICENSE](LICENSE).
