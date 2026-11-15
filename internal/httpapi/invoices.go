package httpapi

import (
	"net/http"
)

// handleCreateInvoice handles POST /invoices.
func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	// Piece B: replace this line with your code.
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

// handleGetInvoice handles GET /invoices/{id}.
func (s *Server) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	// Piece C: replace this line with your code.
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}
