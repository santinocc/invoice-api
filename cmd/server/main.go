// Command server runs the invoice API over HTTP.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/santinocc/invoice-api/internal/httpapi"
	"github.com/santinocc/invoice-api/internal/invoice"
	"github.com/santinocc/invoice-api/internal/postgres"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Postgres when DATABASE_URL is set, memory otherwise. If the database
	// cannot be reached, exit at once (fail fast): never fall back to memory
	// silently, because that would accept invoices and then lose them.
	var store invoice.Store
	if url := os.Getenv("DATABASE_URL"); url != "" {
		db, err := postgres.Open(context.Background(), url)
		if err != nil {
			slog.Error("connect to database", "err", err)
			os.Exit(1)
		}

		defer db.Close()
		store = postgres.NewStore(db)
		slog.Info("storage", "type", "postgres")
	} else {
		store = invoice.NewMemoryStore()
		slog.Info("storage", "type", "memory", "note", "invoices are lost on restart")
	}

	api := httpapi.NewServer(store)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
