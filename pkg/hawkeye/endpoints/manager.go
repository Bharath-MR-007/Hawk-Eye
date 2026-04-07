// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"gopkg.in/yaml.v3"
)

// Endpoint represents a remote Sparrow instance
type Endpoint struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	URL             string        `json:"url"`
	Region          string        `json:"region"`
	Environment     string        `json:"environment"`
	PollingInterval time.Duration `json:"polling_interval"`
	Timeout         time.Duration `json:"timeout"`
	Enabled         bool          `json:"enabled"`
	Status          string        `json:"status"` // UP or DOWN
	LastSeen        time.Time     `json:"last_seen"`
	Latency         time.Duration `json:"latency"`
}

// Prometheus internal metrics for endpoints
var (
	endpointUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hawkeye_endpoint_up",
		Help: "Whether the registered endpoint is up (1) or down (0)",
	}, []string{"endpoint_name", "region", "environment", "id"})
	
	endpointLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hawkeye_endpoint_latency_seconds",
		Help: "Latency of the remote endpoint scraping in seconds",
	}, []string{"endpoint_name", "region", "environment", "id"})

	endpointLastSeen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hawkeye_endpoint_last_seen_timestamp_seconds",
		Help: "Timestamp of the last successful reachability check",
	}, []string{"endpoint_name", "region", "environment", "id"})
)

// Manager handles the lifecycle of distributed Sparrow endpoints
type Manager struct {
	mu           sync.RWMutex
	endpoints    map[string]*Endpoint
	registryFile string
	sdFile       string
	client       *http.Client
	ctx          context.Context
	cancel       context.CancelFunc
}

// PrometheusFileSD represents a single Prometheus target group
type PrometheusFileSD struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels"`
}

// NewManager creates a new Endpoint Manager
func NewManager(registryFile, sdFile string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		endpoints:    make(map[string]*Endpoint),
		registryFile: registryFile,
		sdFile:       sdFile,
		client:       &http.Client{Timeout: 5 * time.Second},
		ctx:          ctx,
		cancel:       cancel,
	}
	m.loadRegistry()
	go m.workerPool()
	return m
}

// AddEndpoint adds or updates an endpoint
func (m *Manager) AddEndpoint(ep *Endpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ep.ID == "" {
		ep.ID = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if ep.PollingInterval == 0 {
		ep.PollingInterval = 15 * time.Second
	}
	if ep.Timeout == 0 {
		ep.Timeout = 5 * time.Second
	}
	ep.Status = "UNKNOWN"

	m.endpoints[ep.ID] = ep
	m.saveRegistry()
	m.generateSDConfig()

	return nil
}

// GetEndpoints returns all registered endpoints
func (m *Manager) GetEndpoints() []Endpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Endpoint
	for _, ep := range m.endpoints {
		// return a copy
		result = append(result, *ep)
	}
	return result
}

// DeleteEndpoint removes an endpoint by ID
func (m *Manager) DeleteEndpoint(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.endpoints, id)
	m.saveRegistry()
	m.generateSDConfig()
}

// TestConnectivity tests a given URL for /metrics accessibility
func (m *Manager) TestConnectivity(ctx context.Context, targetURL string, timeout time.Duration) (bool, time.Duration, error) {
	tracer := otel.Tracer("endpoint_manager")
	_, span := tracer.Start(ctx, "TestConnectivity")
	defer span.End()

	client := &http.Client{Timeout: timeout}
	start := time.Now()
	resp, err := client.Get(targetURL + "/metrics")
	latency := time.Since(start)

	if err != nil {
		return false, latency, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Empty body

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, latency, nil
	}
	return false, latency, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func (m *Manager) Shutdown() {
	m.cancel()
}

// loadRegistry loads the state from disk
func (m *Manager) loadRegistry() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.registryFile)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.FromContext(m.ctx).Error("Failed to read endpoint registry", "error", err)
		}
		return
	}

	var endpoints []Endpoint
	if err := json.Unmarshal(data, &endpoints); err != nil {
		logger.FromContext(m.ctx).Error("Failed to unmarshal endpoint registry", "error", err)
		return
	}

	for i := range endpoints {
		m.endpoints[endpoints[i].ID] = &endpoints[i]
		
		// Setup initial prometheus states from loaded registry
		labels := prometheus.Labels{"endpoint_name": endpoints[i].Name, "region": endpoints[i].Region, "environment": endpoints[i].Environment, "id": endpoints[i].ID}
		if endpoints[i].Status == "UP" {
			endpointUp.With(labels).Set(1)
		} else {
			endpointUp.With(labels).Set(0)
		}
		endpointLatency.With(labels).Set(endpoints[i].Latency.Seconds())
		endpointLastSeen.With(labels).Set(float64(endpoints[i].LastSeen.Unix()))
	}
}

// saveRegistry saves the state to disk
func (m *Manager) saveRegistry() {
	var list []Endpoint
	for _, ep := range m.endpoints {
		list = append(list, *ep)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		logger.FromContext(m.ctx).Error("Failed to marshal endpoint registry", "error", err)
		return
	}
	os.WriteFile(m.registryFile, data, 0644)
}

// generateSDConfig generates the file_sd_configs for Prometheus
func (m *Manager) generateSDConfig() {
	var sdConfigs []PrometheusFileSD

	for _, ep := range m.endpoints {
		if !ep.Enabled {
			continue
		}
		// Assuming the URL is http://host:port, extract host:port
		// Provide basic URL parsing (simplified for requirements)
		sdConfigs = append(sdConfigs, PrometheusFileSD{
			Targets: []string{ep.URL},
			Labels: map[string]string{
				"endpoint_name": ep.Name,
				"region":        ep.Region,
				"environment":   ep.Environment,
				"sparrow_id":    ep.ID,
			},
		})
	}

	data, err := yaml.Marshal(sdConfigs)
	if err != nil {
		logger.FromContext(m.ctx).Error("Failed to marshal SD configs", "error", err)
		return
	}
	os.WriteFile(m.sdFile, data, 0644)
}

// workerPool checks all endpoints based on their interval
func (m *Manager) workerPool() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Rate limit concurrency
	sem := make(chan struct{}, 10)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			var toCheck []*Endpoint
			now := time.Now()
			for _, ep := range m.endpoints {
				if !ep.Enabled {
					continue
				}
				// Determine if it is time to poll
				if now.Sub(ep.LastSeen) > ep.PollingInterval || ep.Status == "UNKNOWN" {
					toCheck = append(toCheck, ep)
				}
			}
			m.mu.RUnlock()

			for _, ep := range toCheck {
				sem <- struct{}{}
				go func(endpoint *Endpoint) {
					defer func() { <-sem }()
					
					tracer := otel.Tracer("endpoint_manager")
					ctx, span := tracer.Start(m.ctx, "HealthCheckWorker")
					defer span.End()
					
					// We perform a test connectivity check against the endpoint
					up, lat, err := m.TestConnectivity(ctx, endpoint.URL, endpoint.Timeout)

					m.mu.Lock()
					labels := prometheus.Labels{"endpoint_name": endpoint.Name, "region": endpoint.Region, "environment": endpoint.Environment, "id": endpoint.ID}
					if up {
						endpoint.Status = "UP"
						endpoint.LastSeen = time.Now()
						endpoint.Latency = lat
						
						endpointUp.With(labels).Set(1)
						endpointLatency.With(labels).Set(lat.Seconds())
						endpointLastSeen.With(labels).Set(float64(endpoint.LastSeen.Unix()))
					} else {
						endpoint.Status = "DOWN"
						endpointUp.With(labels).Set(0)
						logger.FromContext(m.ctx).Debug("Endpoint check failed", "id", endpoint.ID, "error", err)
					}
					m.saveRegistry()
					m.mu.Unlock()
				}(ep)
			}
		}
	}
}
