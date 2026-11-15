// Command server runs the invoice API over HTTP.
package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/santinocc/invoice-api/internal/httpapi"
	"github.com/santinocc/invoice-api/internal/invoice"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	api := httpapi.NewServer(invoice.NewMemoryStore())

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
