# invoice-api

A small invoice API written in Go, built as a learning project. It stores invoices, calculates Japanese consumption tax (10% standard, 8% reduced), and will later reconcile bank transactions against open invoices.

## Status

Work in progress. This section lists only what is actually finished.

- [x] Project setup: Go module, CI (gofmt, `go vet`, `go test -race`), MIT license
- [x] `GET /healthz`
- [ ] Stage 1: `POST /invoices` and `GET /invoices/{id}`, in-memory storage, tax calculation, tests
- [ ] Stage 2: PostgreSQL storage and `GET /invoices` with filters
- [ ] Stage 3: `POST /reconcile` to match bank transactions to open invoices
- [ ] Stage 4: layered structure, linting, Dockerfile

## Requirements

- Go 1.27 or newer (the exact version is in `go.mod`)

## Run

```sh
cp .env.example .env   # optional; PORT defaults to 8080
go run ./cmd/server
curl http://localhost:8080/healthz
```

## Test

```sh
go test ./...
```

## Project layout

```
cmd/server/        entry point: reads config, wires routes, starts the HTTP server
internal/invoice/  domain: invoice types, validation, tax calculation, storage
internal/httpapi/  HTTP handlers: decode JSON, call the domain, encode responses
```

## License

MIT. See [LICENSE](LICENSE).
