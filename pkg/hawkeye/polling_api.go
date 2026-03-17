// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package hawkeye

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"gopkg.in/yaml.v3"
)

type CheckConfig struct {
	Interval string   `yaml:"interval,omitempty" json:"interval"`
	Timeout  string   `yaml:"timeout,omitempty" json:"timeout"`
	Targets  []string `json:"targets"`
}

type PollingPayload map[string]CheckConfig

func (s *Hawkeye) handleGetPolling(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	configPath := "checks.yaml"
	if s.config.Loader.File.Path != "" {
		configPath = s.config.Loader.File.Path
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("Failed to read config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		log.Error("Failed to parse config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	parseTargets := func(val interface{}) []string {
		var tgs []string
		if targetsArr, ok := val.([]interface{}); ok {
			for _, t := range targetsArr {
				if strVal, ok := t.(string); ok {
					tgs = append(tgs, strVal)
				} else if mapVal, ok := t.(map[string]interface{}); ok {
					if url, ok := mapVal["url"].(string); ok {
						tgs = append(tgs, url)
					} else if addr, ok := mapVal["addr"].(string); ok {
						tgs = append(tgs, addr)
					} else if name, ok := mapVal["name"].(string); ok {
						tgs = append(tgs, name)
					}
				}
			}
		}
		return tgs
	}

	res := make(PollingPayload)
	for key, val := range m {
		if checkMap, ok := val.(map[string]interface{}); ok {
			var config CheckConfig
			if interval, ok := checkMap["interval"].(string); ok {
				config.Interval = interval
			}
			if timeout, ok := checkMap["timeout"].(string); ok {
				config.Timeout = timeout
			}

			// Also extract targets to let the frontend correlate them
			if tgs, ok := checkMap["targets"]; ok {
				config.Targets = parseTargets(tgs)
			}

			// Add regardless if they have interval/timeout or not so they can be configured!
			if config.Interval != "" || config.Timeout != "" {
				res[key] = config
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Hawkeye) handleUpdatePolling(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req PollingPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	configPath := "checks.yaml"
	if s.config.Loader.File.Path != "" {
		configPath = s.config.Loader.File.Path
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("Failed to read config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		log.Error("Failed to parse config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	findKey := func(node *yaml.Node, key string) *yaml.Node {
		if node.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return node.Content[i+1]
			}
		}
		return nil
	}

	setOrAddKey := func(mapping *yaml.Node, key, value string) {
		if value == "" {
			return
		}
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == key {
				mapping.Content[i+1].Value = value
				return
			}
		}
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Value: value},
		)
	}

	if len(root.Content) > 0 {
		for checkName, config := range req {
			if checkNode := findKey(root.Content[0], checkName); checkNode != nil {
				setOrAddKey(checkNode, "interval", config.Interval)
				setOrAddKey(checkNode, "timeout", config.Timeout)
			}
		}
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		log.Error("Failed to marshal config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		log.Error("Failed to write config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
