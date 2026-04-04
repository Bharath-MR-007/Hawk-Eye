// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package hawkeye

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/internal/nnmi"
	"gopkg.in/yaml.v3"
	"net/url"
	"strings"
)

const nnmiConfigFile = "nnmi_config.yaml"

// handleGetNnmi returns the current NNMi configuration (password redacted).
func (s *Hawkeye) handleGetNnmi(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	b, err := os.ReadFile(nnmiConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"enabled":false,"host":"","port":9443,"user":""}`))
			return
		}
		log.Error("Failed to read NNMi config", "error", err)
		http.Error(w, "Failed to read config", http.StatusInternalServerError)
		return
	}

	var cfg nnmi.Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse NNMi config", "error", err)
		http.Error(w, "Failed to parse config", http.StatusInternalServerError)
		return
	}
	cfg.Password = "" // never expose to frontend

	w.Header().Set("Content-Type", "application/json")
	// Map to what frontend expects
	type frontendConfig struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	_ = json.NewEncoder(w).Encode(frontendConfig{
		Enabled: cfg.Enabled,
		Host:    cfg.Host,
		Port:    cfg.Port,
		User:    cfg.Username,
	})
}

// handleUpdateNnmi saves the NNMi configuration to disk and updates runtime config.
func (s *Hawkeye) handleUpdateNnmi(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	// Map from frontend format
	type frontendUpdate struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	var fe frontendUpdate
	if err := json.NewDecoder(r.Body).Decode(&fe); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	cfg := nnmi.Config{
		Enabled:  fe.Enabled,
		Host:     fe.Host,
		Port:     fe.Port,
		Username: fe.User,
		Password: fe.Password,
		UseSSL:   fe.Port == 9443 || fe.Port == 443,
		Timeout:  s.config.Nnmi.Timeout,
		CacheTTL: s.config.Nnmi.CacheTTL,
	}

	if cfg.Port == 0 {
		cfg.Port = 9443
	}

	out, err := yaml.Marshal(&cfg)
	if err != nil {
		http.Error(w, "Failed to serialize config", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(nnmiConfigFile, out, 0o600); err != nil {
		log.Error("Failed to write NNMi config", "error", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// Update the running config
	s.config.Nnmi = cfg
	log.InfoContext(r.Context(), "NNMi configuration updated and applied", "host", cfg.Host, "port", cfg.Port)

	// Trigger immediate reconcile to update traceroute checks
	s.TriggerReconcile(r.Context())

	// Trigger a reconcile of the controller to update the NNMi client in active checks
	// We use the same config but since s.config.Nnmi is updated, Reconcile will pick it up
	// Actually we need to pass a runtime.Config. We can construct one or just wait for next loader tick.
	// For immediate effect:
	// s.cRuntime <- s.config.ToRuntimeConfig() // If such method exists

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"saved"}`))
}

// handleTestNnmi attempts a simple HTTP connectivity test to the NNMi REST API endpoint.
func (s *Hawkeye) handleTestNnmi(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	type frontendUpdate struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	var fe frontendUpdate
	if err := json.NewDecoder(r.Body).Decode(&fe); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if fe.Port == 0 {
		fe.Port = 9443
	}

	scheme := "http"
	if fe.Port == 9443 || fe.Port == 443 {
		scheme = "https"
	}

	urlStr := fmt.Sprintf("%s://%s:%d/idp/oauth2/token", scheme, fe.Host, fe.Port)
	log.InfoContext(r.Context(), "Testing NNMi OAuth connection", "url", urlStr)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", fe.User)
	data.Set("password", fe.Password)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"message":"Failed to build request: ` + err.Error() + `"}`))
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 8 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		log.Warn("NNMi test connection failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"success":false,"message":"Connection failed: %s"}`, err.Error())))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"Connection successful! OAuth token acquired."}`))
		return
	}

	msg := fmt.Sprintf("HTTP %d from NNMi server. Invalid credentials or OAuth token scheme.", resp.StatusCode)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"success":false,"message":"%s"}`, msg)))
}
