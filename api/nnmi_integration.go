package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Bharath-MR-007/hawk-eye/internal/nnmi"
	"github.com/gorilla/mux"
)

// A mocked DB interface for the integration based on the user's snippet.
type DatabaseMock interface {
	GetCachedIncidents(ctx context.Context) ([]map[string]interface{}, error)
	CacheIncidents(ctx context.Context, incidents []map[string]interface{})
	UpdateIncident(ctx context.Context, incident *nnmi.Incident)
}

type NNMIIntegrationHandler struct {
	nnmiClient *nnmi.NNMIClient
	hawkeyeDB  DatabaseMock
}

func NewNNMIIntegrationHandler(nnmiClient *nnmi.NNMIClient, db DatabaseMock) *NNMIIntegrationHandler {
	return &NNMIIntegrationHandler{
		nnmiClient: nnmiClient,
		hawkeyeDB:  db,
	}
}

func (h *NNMIIntegrationHandler) RegisterRoutes(r *mux.Router) {
	// NNMi data endpoints
	r.HandleFunc("/api/v1/integrations/nnmi/incidents", h.getNNMiIncidents).Methods("GET")
	r.HandleFunc("/api/v1/integrations/nnmi/incidents/{uuid}", h.getNNMiIncident).Methods("GET")
	r.HandleFunc("/api/v1/integrations/nnmi/incidents/{uuid}/notes", h.updateIncidentNotes).Methods("PATCH")
	r.HandleFunc("/api/v1/integrations/nnmi/nodes", h.getNNMiNodes).Methods("GET")
	r.HandleFunc("/api/v1/integrations/nnmi/nodes/{uuid}", h.getNNMiNode).Methods("GET")
	r.HandleFunc("/api/v1/integrations/nnmi/topology/path", h.getNetworkPath).Methods("GET")

	// Sync endpoints
	r.HandleFunc("/api/v1/integrations/nnmi/sync/incidents", h.syncIncidents).Methods("POST")
	r.HandleFunc("/api/v1/integrations/nnmi/sync/nodes", h.syncNodes).Methods("POST")

	// Webhook receiver for NNMi events
	r.HandleFunc("/api/v1/integrations/nnmi/webhook", h.receiveWebhook).Methods("POST")
}

func (h *NNMIIntegrationHandler) getNNMiIncidents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get from cache first
	if h.hawkeyeDB != nil {
		incidents, err := h.hawkeyeDB.GetCachedIncidents(ctx)
		if err == nil && len(incidents) > 0 {
			json.NewEncoder(w).Encode(incidents)
			return
		}
	}

	// Fallback to NNMi API
	if h.nnmiClient == nil {
		http.Error(w, "NNMi Client not configured", http.StatusInternalServerError)
		return
	}
	
	nnmiIncidents, err := h.nnmiClient.GetOpenKeyIncidents(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to HawkEye format
	var result []map[string]interface{}
	for _, inc := range nnmiIncidents {
		result = append(result, map[string]interface{}{
			"uuid":                inc.UUID,
			"name":                inc.Name,
			"severity":            inc.Severity,
			"priority":            inc.Priority,
			"message":             inc.FormattedMessage,
			"sourceNodeName":      inc.SourceNodeName,
			"firstOccurrenceTime": inc.FirstOccurrenceTime,
			"lastOccurrenceTime":  inc.LastOccurrenceTime,
			"status":              inc.LifecycleState,
		})
	}

	// Cache in HawkEye DB
	if h.hawkeyeDB != nil {
		go h.hawkeyeDB.CacheIncidents(ctx, result)
	}

	json.NewEncoder(w).Encode(result)
}

func (h *NNMIIntegrationHandler) broadcastIncidentUpdate(incident *nnmi.Incident) {
	// Dummy method for broadcast
}

func (h *NNMIIntegrationHandler) receiveWebhook(w http.ResponseWriter, r *http.Request) {
	// Receive NNMi webhook events (PDF page 136)
	var event struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process event
	switch event.Type {
	case "incident.created", "incident.updated":
		var incident nnmi.Incident
		if err := json.Unmarshal(event.Payload, &incident); err == nil {
			// Update HawkEye cache
			if h.hawkeyeDB != nil {
				h.hawkeyeDB.UpdateIncident(r.Context(), &incident)
			}

			// Trigger real-time updates for connected clients
			h.broadcastIncidentUpdate(&incident)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// Stubs for non-implemented handlers
func (h *NNMIIntegrationHandler) getNNMiIncident(w http.ResponseWriter, r *http.Request)     {}
func (h *NNMIIntegrationHandler) updateIncidentNotes(w http.ResponseWriter, r *http.Request) {}
func (h *NNMIIntegrationHandler) getNNMiNodes(w http.ResponseWriter, r *http.Request)        {}
func (h *NNMIIntegrationHandler) getNNMiNode(w http.ResponseWriter, r *http.Request)         {}
func (h *NNMIIntegrationHandler) getNetworkPath(w http.ResponseWriter, r *http.Request)      {}
func (h *NNMIIntegrationHandler) syncIncidents(w http.ResponseWriter, r *http.Request)       {}
func (h *NNMIIntegrationHandler) syncNodes(w http.ResponseWriter, r *http.Request)           {}
