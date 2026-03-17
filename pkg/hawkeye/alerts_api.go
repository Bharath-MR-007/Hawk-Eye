package hawkeye

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"gopkg.in/yaml.v3"
)

type AlertRule struct {
	Alert       string            `yaml:"alert" json:"alert"`
	Expr        string            `yaml:"expr" json:"expr"`
	For         string            `yaml:"for" json:"for"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

type AlertGroup struct {
	Name  string      `yaml:"name" json:"name"`
	Rules []AlertRule `yaml:"rules" json:"rules"`
}

type PrometheusRulesConfig struct {
	Groups []AlertGroup `yaml:"groups" json:"groups"`
}

func (s *Hawkeye) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	b, err := os.ReadFile("prometheus_rules.yaml")
	if err != nil {
		log.Error("Failed to read prometheus rules", "error", err)
		http.Error(w, "Failed to read alerts configuration", http.StatusInternalServerError)
		return
	}

	var config PrometheusRulesConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		log.Error("Failed to parse prometheus rules", "error", err)
		http.Error(w, "Internal error parsing alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (s *Hawkeye) handleUpdateAlerts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var config PrometheusRulesConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		log.Error("Invalid request payload for alerts", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	b, err := yaml.Marshal(&config)
	if err != nil {
		log.Error("Failed to encode prometheus rules to YAML", "error", err)
		http.Error(w, "Error generating configuration format", http.StatusInternalServerError)
		return
	}

	// Make sure we use proper indentation (but yaml.Marshal is usually fine)
	if err := os.WriteFile("prometheus_rules.yaml", b, 0644); err != nil {
		log.Error("Failed to write prometheus rules", "error", err)
		http.Error(w, "Failed to save alerts configuration", http.StatusInternalServerError)
		return
	}

	// Trigger Prometheus reload
	// Ignore errors if Prometheus is temporarily unavailable, since it will pick it up eventually
	// or prometheus web lifecycle needs to process it.
	go func() {
		http.Post("http://prometheus:9090/-/reload", "", nil)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
