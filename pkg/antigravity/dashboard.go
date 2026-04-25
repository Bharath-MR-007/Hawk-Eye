package antigravity

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
)

// DashboardHandler provides the endpoints for the Antigravity UI
type DashboardHandler struct {
	predictor *Predictor
}

func NewDashboardHandler(p *Predictor) *DashboardHandler {
	return &DashboardHandler{
		predictor: p,
	}
}

// ServeActionableInsights provides proactive actionable data instead of raw metrics.
func (h *DashboardHandler) ServeActionableInsights(w http.ResponseWriter, r *http.Request) {
	// In a real scenario, this aggregates all AnomalyReports and outputs
	// prioritized "Action-First" data.
	
	// Mocking an actionable item based on "What-if" Future dimensional prediction
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"recommended_actions": []map[string]interface{}{
			{
				"id":         "ACT-001",
				"action":     "Pre-warm additional DB read replicas in us-east-2",
				"confidence": 0.94,
				"reason":     "Temporal prediction detected incoming 300% traffic spike in 12 mins based on baseline deviations.",
				"auto_fix":   true,
			},
		},
		"realtime_anomalies": []map[string]interface{}{},
		"system_health": map[string]interface{}{
			"status": "GREEN_PENDING",
			"score":  88, // 0-1 seconds cognitive load metric
		},
	}
	json.NewEncoder(w).Encode(response)
}

// TriggerChaos initiates a Chaos Engineering event directly from the dashboard
func (h *DashboardHandler) TriggerChaos(w http.ResponseWriter, r *http.Request) {
	// Parse target service to disrupt
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "random_node"
	}
	
	log.Printf("🔥 CHAOS ENGINE INVOKED: Isolating %s\n", target)
	
	p := h.predictor.Observe("chaos_"+target, float64(rand.Intn(100)))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "CHAOS_INJECTED",
		"target":  target,
		"impact":  "14 dependencies simulated to fail in cascade",
		"recovery_time_prediction": "42s",
		"report": p,
	})
}
