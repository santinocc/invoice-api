package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/santinocc/invoice-api/internal/reconcile"
)

type reconcileRequest struct {
	Transactions []reconcile.Transaction `json:"transactions"`
}

// handleReconcile serves POST /reconcile. It only suggests matches and saves
// nothing, so sending the same request twice is harmless.
func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req reconcileRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be JSON with a transactions list")
		return
	}
	if err := reconcile.Validate(req.Transactions); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Each distinct amount once: a statement may hold hundreds of equal payments.
	var totals []int64
	seen := map[int64]bool{}
	for _, tx := range req.Transactions {
		if !seen[tx.Amount] {
			seen[tx.Amount] = true
			totals = append(totals, tx.Amount)
		}
	}

	candidates, err := s.store.FindByTotals(r.Context(), totals)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load invoices")
		return
	}

	res := reconcile.Suggest(req.Transactions, candidates)
	// Send [] rather than null for empty lists, so clients can always loop over them.
	if res.Matched == nil {
		res.Matched = []reconcile.Match{}
	}
	if res.Ambiguous == nil {
		res.Ambiguous = []reconcile.Ambiguity{}
	}
	if res.Unmatched == nil {
		res.Unmatched = []string{}
	}
	writeJSON(w, http.StatusOK, res)
}
