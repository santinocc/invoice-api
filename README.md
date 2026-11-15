# invoice-api

A small invoice API written in Go, built as a learning project. It stores invoices and calculates Japanese consumption tax (10% standard, 8% reduced). Later stages add PostgreSQL storage and reconciliation of bank transactions against open invoices.

## Status

Stage 1 is finished. This list shows only what is actually done.

- [x] `POST /invoices`: validates the invoice, calculates totals, saves it, returns `201`
- [x] `GET /invoices/{id}`: returns the invoice, or `404`
- [x] `GET /healthz`
- [x] Tax calculated per tax rate and rounded down once per rate, not per line
- [x] In-memory storage behind a `Store` interface, safe for concurrent requests
- [x] Tests for the domain, the store (including 100 concurrent creates under the race detector) and the HTTP handlers
- [x] CI on GitHub Actions: `gofmt`, `go vet`, `go test -race`

Not done yet:

- [ ] Stage 2: PostgreSQL storage and `GET /invoices` with filters
- [ ] Stage 3: `POST /reconcile` to match bank transactions to open invoices
- [ ] Stage 4: linting with `golangci-lint`, Dockerfile, graceful shutdown

Invoices are lost when the server stops, because storage is in memory only.

## Requirements

- Go 1.27 or newer (the exact version is in `go.mod`)

## Run

```sh
go run ./cmd/server
```

The server listens on port `8080`. Set `PORT` to change it (see `.env.example`).

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
{"id":"inv_1","vendor":"Example Supplies","issue_date":"2026-11-01","due_date":"2026-11-30","items":[...],"subtotal":2080,"tax":186,"total":2266}
```

Get it back:

```sh
curl -i localhost:8080/invoices/inv_1
```

Errors use the matching status code and a JSON body such as `{"error":"invalid invoice: vendor is required"}`:

| Status | When |
|---|---|
| `400` | The body is not valid JSON, has unknown fields, or fails validation |
| `404` | No invoice has that ID |
| `500` | The store failed |

## Rules

- Amounts are whole yen stored as `int64`. Floating point is never used for money.
- Dates use the format `YYYY-MM-DD`. The due date cannot be before the issue date.
- Each item needs a quantity above 0, a unit price of 0 or more, and a tax rate of 8 or 10.
- Tax is calculated on the total of each tax rate and rounded down once per rate, as required for qualified invoices (適格請求書) in Japan. Rounding each line separately can give a different result.
- The client cannot set `id`, `subtotal`, `tax` or `total`; the server overwrites them.

## Test

```sh
go test -race ./...
```

## Project layout

```
cmd/server/        entry point: reads PORT, wires the store and routes, starts the server
internal/invoice/  domain: types, validation, tax calculation, Store interface, in-memory store
internal/httpapi/  HTTP: routes, JSON handlers, error responses
```

## License

MIT. See [LICENSE](LICENSE).
