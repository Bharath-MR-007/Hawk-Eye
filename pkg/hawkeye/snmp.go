// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package hawkeye

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/config"
	"gopkg.in/yaml.v3"
)

// AlertmanagerWebhook represents the data sent by Alertmanager webhooks
type AlertmanagerWebhook struct {
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

func (s *Hawkeye) handleGetSnmp(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	b, err := os.ReadFile("snmp_config.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"enabled":false,"target":"localhost","port":162,"community":"public"}`))
			return
		}
		log.Error("Failed to read snmp config", "error", err)
		http.Error(w, "Failed to read config", http.StatusInternalServerError)
		return
	}

	var cfg config.SnmpConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse snmp config", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Hawkeye) handleUpdateSnmp(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	var cfg config.SnmpConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		log.Error("Invalid snmp config payload", "error", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	b, err := yaml.Marshal(&cfg)
	if err != nil {
		log.Error("Failed to marshal snmp config", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile("snmp_config.yaml", b, 0644); err != nil {
		log.Error("Failed to write snmp config", "error", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func (s *Hawkeye) handleSnmpTrap(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	// Read SNMP Config
	b, err := os.ReadFile("snmp_config.yaml")
	if err != nil {
		log.Error("SNMP config not found, skipping trap", "error", err)
		http.Error(w, "SNMP not configured", http.StatusServiceUnavailable)
		return
	}
	var cfg config.SnmpConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse snmp config", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !cfg.Enabled {
		log.Debug("SNMP traps are disabled")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Decode Alertmanager Webhook
	var webhook AlertmanagerWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		log.Error("Failed to decode alertmanager webhook", "error", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Send Traps for each alert
	for _, alert := range webhook.Alerts {
		log.Info("Sending SNMP trap for alert", "alert", alert.Labels["alertname"], "status", alert.Status)
		err := s.sendTrap(cfg, alert)
		if err != nil {
			log.Error("Failed to send SNMP trap", "error", err, "alert", alert.Labels["alertname"])
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Hawkeye) sendTrap(cfg config.SnmpConfig, alert Alert) error {
	g := &gosnmp.GoSNMP{
		Target:    cfg.Target,
		Port:      uint16(cfg.Port),
		Community: cfg.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(2) * time.Second,
		Retries:   3,
	}

	err := g.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer g.Conn.Close()

	// OIDs
	// Generic trap OID for enterprise specific traps
	trapOID := ".1.3.6.1.4.1.2021.251.1"

	// Variables
	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "Unknown Alert"
	}
	severity := alert.Labels["severity"]
	if severity == "" {
		severity = "unknown"
	}
	instance := alert.Labels["instance"]
	if instance == "" {
		instance = alert.Labels["target"]
	}
	description := alert.Annotations["description"]
	if description == "" {
		description = alert.Annotations["summary"]
	}

	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.6.3.1.1.4.1.0",
		Type:  gosnmp.ObjectIdentifier,
		Value: trapOID,
	}

	// Add data VarBinds
	vbs := []gosnmp.SnmpPDU{
		pdu,
		{
			Name:  ".1.3.6.1.4.1.2021.251.1.1", // AlertName
			Type:  gosnmp.OctetString,
			Value: alertName,
		},
		{
			Name:  ".1.3.6.1.4.1.2021.251.1.2", // Status
			Type:  gosnmp.OctetString,
			Value: alert.Status,
		},
		{
			Name:  ".1.3.6.1.4.1.2021.251.1.3", // Severity
			Type:  gosnmp.OctetString,
			Value: severity,
		},
		{
			Name:  ".1.3.6.1.4.1.2021.251.1.4", // Instance
			Type:  gosnmp.OctetString,
			Value: instance,
		},
		{
			Name:  ".1.3.6.1.4.1.2021.251.1.5", // Description
			Type:  gosnmp.OctetString,
			Value: description,
		},
	}

	trap := gosnmp.SnmpTrap{
		Variables: vbs,
	}

	_, err = g.SendTrap(trap)
	return err
}
