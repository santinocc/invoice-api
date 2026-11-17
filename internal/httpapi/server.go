// Package httpapi exposes the invoice domain over HTTP with JSON.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// Server serves the invoice API. It depends on the invoice.Store
// interface, not on a concrete store, so storage can change later.
type Server struct {
	store invoice.Store
}

// NewServer returns a Server that keeps invoices in store.
func NewServer(store invoice.Store) *Server {
	return &Server{store: store}
}

// Routes returns an http.Handler with every API route registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /invoices", s.handleCreateInvoice)
	mux.HandleFunc("GET /invoices", s.handleListInvoices)
	mux.HandleFunc("GET /invoices/{id}", s.handleGetInvoice)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// errorResponse is the JSON body sent with every error: {"error": "..."}.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON sends v as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends {"error": msg} with the given HTTP status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
