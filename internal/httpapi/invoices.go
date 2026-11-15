package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// handleCreateInvoice handles POST /invoices.
func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var inv invoice.Invoice
	if err := dec.Decode(&inv); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be a valid invoice in JSON")
		return
	}

	if err := inv.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	inv.CalculateTotals()

	created, err := s.store.Create(inv)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the invoice")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleGetInvoice handles GET /invoices/{id}.
func (s *Server) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := s.store.Get(id)
	if errors.Is(err, invoice.ErrNotFound) {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load the invoice")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}
