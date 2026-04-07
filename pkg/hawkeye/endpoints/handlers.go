// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package endpoints

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/Bharath-MR-007/hawk-eye/pkg/api"
)

// APIHandler wraps the manager routines
type APIHandler struct {
	Manager *Manager
}

func NewAPIHandler(m *Manager) *APIHandler {
	return &APIHandler{Manager: m}
}

// GetRoutes returns the REST routes
func (h *APIHandler) GetRoutes() []api.Route {
	return []api.Route{
		{Path: "/api/v1/endpoints", Method: http.MethodGet, Handler: h.getEndpoints},
		{Path: "/api/v1/endpoints", Method: http.MethodPost, Handler: h.addEndpoint},
		{Path: "/api/v1/endpoints/{id}", Method: http.MethodDelete, Handler: h.deleteEndpoint},
		{Path: "/api/v1/endpoints/test", Method: http.MethodPost, Handler: h.testEndpoint},
	}
}

func (h *APIHandler) getEndpoints(w http.ResponseWriter, r *http.Request) {
	eps := h.Manager.GetEndpoints()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eps)
}

func (h *APIHandler) addEndpoint(w http.ResponseWriter, r *http.Request) {
	var ep Endpoint
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Manager.AddEndpoint(&ep); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ep)
}

func (h *APIHandler) deleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing ID", http.StatusBadRequest)
		return
	}

	h.Manager.DeleteEndpoint(id)
	w.WriteHeader(http.StatusNoContent)
}

// testEndpoint is used prior to saving to preview connectivity
func (h *APIHandler) testEndpoint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL     string        `json:"url"`
		Timeout time.Duration `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Timeout == 0 {
		body.Timeout = 5 * time.Second
	}

	up, latency, err := h.Manager.TestConnectivity(r.Context(), body.URL, body.Timeout)
	res := struct {
		Success bool   `json:"success"`
		Latency string `json:"latency"`
		Error   string `json:"error,omitempty"`
	}{
		Success: up,
		Latency: latency.String(),
	}
	if err != nil {
		res.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
