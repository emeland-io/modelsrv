package oapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// HandleGetEventsHistory handles GET /api/events/history with query params for filtering/pagination.
func (a *ApiServer) HandleGetEventsHistory(w http.ResponseWriter, r *http.Request) {
	q := events.EventQuery{}

	if v := r.URL.Query().Get("operation"); v != "" {
		op := events.ParseWireOperation(v)
		if op == events.UnknownOperation {
			http.Error(w, "invalid operation filter", http.StatusBadRequest)
			return
		}
		q.Operation = &op
	}

	if v := r.URL.Query().Get("resourceType"); v != "" {
		rt := events.ParseWireKind(v)
		if rt == events.UnknownResourceType {
			http.Error(w, "invalid resourceType filter", http.StatusBadRequest)
			return
		}
		q.ResourceType = &rt
	}

	if v := r.URL.Query().Get("resourceId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			http.Error(w, "invalid resourceId", http.StatusBadRequest)
			return
		}
		q.ResourceId = &id
	}

	if v := r.URL.Query().Get("sinceSeq"); v != "" {
		seq, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid sinceSeq", http.StatusBadRequest)
			return
		}
		q.SinceSeq = seq
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		lim, err := strconv.Atoi(v)
		if err != nil || lim < 1 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		q.Limit = lim
	}

	if r.URL.Query().Get("includePayload") == "true" {
		q.IncludePayload = true
	}

	results, err := a.Events.QueryEvents(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []events.StoredEvent{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
