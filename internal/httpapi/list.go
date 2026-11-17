package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/santinocc/invoice-api/internal/invoice"
)

// defaultListLimit is the page size when the client does not send ?limit=.
const defaultListLimit = 20

// listResponse is the body of GET /invoices.
// NextCursor is left out when there are no more pages.
type listResponse struct {
	Invoices   []invoice.Invoice `json:"invoices"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// handleListInvoices serves GET /invoices?vendor=&due_before=&limit=&cursor=
func (s *Server) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := invoice.ListFilter{
		Vendor:    q.Get("vendor"),
		DueBefore: q.Get("due_before"),
		Limit:     defaultListLimit,
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "limit must be a whole number")
			return
		}
		f.Limit = n
	}
	if c := q.Get("cursor"); c != "" {
		dueDate, id, err := decodeCursor(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		f.AfterDueDate, f.AfterID = dueDate, id
	}
	if err := f.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Ask for one extra row: if it comes back, there is a next page.
	limit := f.Limit
	f.Limit = limit + 1
	invoices, err := s.store.List(r.Context(), f)
	if errors.Is(err, invoice.ErrInvalidFilter) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list invoices")
		return
	}

	var next string
	if len(invoices) > limit {
		invoices = invoices[:limit]
		last := invoices[len(invoices)-1]
		next = encodeCursor(last.DueDate, last.ID)
	}
	resp := listResponse{Invoices: invoices, NextCursor: next}
	if resp.Invoices == nil {
		resp.Invoices = []invoice.Invoice{} // send [] rather than null when nothing matches
	}
	writeJSON(w, http.StatusOK, resp)
}
