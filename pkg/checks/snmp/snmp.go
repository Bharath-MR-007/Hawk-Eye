// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package snmp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gosnmp/gosnmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

var (
	_ checks.Check   = (*SnmpCheck)(nil)
	_ checks.Runtime = (*Config)(nil)
)

// TargetResult holds the results for a single SNMP target
type TargetResult struct {
	Success bool            `json:"success"`
	Latency float64         `json:"latency"` // in seconds
	Values  map[string]any  `json:"values"`  // OID Name -> Value
	Error   string          `json:"error,omitempty"`
}

// SnmpResult is the complete result of an SNMP check run
type SnmpResult map[string]TargetResult

type SnmpCheck struct {
	checks.CheckBase
	config  Config
	metrics snmpMetrics
}

type snmpMetrics struct {
	status  *prometheus.GaugeVec
	latency *prometheus.GaugeVec
	value   *prometheus.GaugeVec
}

func newMetrics() snmpMetrics {
	return snmpMetrics{
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_snmp_status",
				Help: "Status of SNMP poll (1 for success, 0 for failure)",
			},
			[]string{"target"},
		),
		latency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_snmp_latency_seconds",
				Help: "Latency of SNMP poll in seconds",
			},
			[]string{"target"},
		),
		value: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_snmp_value",
				Help: "Value of polled SNMP OID (if numeric)",
			},
			[]string{"target", "oid_name"},
		),
	}
}

func (m snmpMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.status.Describe(ch)
	m.latency.Describe(ch)
	m.value.Describe(ch)
}

func (m snmpMetrics) Collect(ch chan<- prometheus.Metric) {
	m.status.Collect(ch)
	m.latency.Collect(ch)
	m.value.Collect(ch)
}

// NewCheck creates a new SNMP check
func NewCheck() checks.Check {
	return &SnmpCheck{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		metrics: newMetrics(),
	}
}

func (s *SnmpCheck) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.DoneChan:
			return nil
		case <-time.After(s.config.Interval):
			log.Debug("Running SNMP poll check")
			res := s.poll(ctx)
			cResult <- checks.ResultDTO{
				Name: s.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
		}
	}
}

func (s *SnmpCheck) poll(ctx context.Context) SnmpResult {
	log := logger.FromContext(ctx)
	results := make(SnmpResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, target := range s.config.Targets {
		wg.Add(1)
		tgt := target
		go func() {
			defer wg.Done()
			
			startTime := time.Now()
			res := s.pollTarget(tgt, log)
			latency := time.Since(startTime).Seconds()
			res.Latency = latency

			mu.Lock()
			results[tgt] = res
			mu.Unlock()

			// Update metrics
			status := 0.0
			if res.Success {
				status = 1.0
			}
			s.metrics.status.WithLabelValues(tgt).Set(status)
			s.metrics.latency.WithLabelValues(tgt).Set(latency)

			for oidName, val := range res.Values {
				if fval, ok := convertToFloat(val); ok {
					s.metrics.value.WithLabelValues(tgt, oidName).Set(fval)
				}
			}
		}()
	}
	wg.Wait()
	return results
}

func (s *SnmpCheck) pollTarget(target string, log *slog.Logger) TargetResult {
	version := gosnmp.Version2c
	if s.config.Version == "v3" {
		version = gosnmp.Version3
	}

	g := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(s.config.Port),
		Community: s.config.Community,
		Version:   version,
		Timeout:   s.config.Timeout,
		Retries:   3,
		MaxOids:   gosnmp.MaxOids,
	}

	err := g.Connect()
	if err != nil {
		return TargetResult{Success: false, Error: fmt.Sprintf("connection failed: %v", err)}
	}
	defer g.Conn.Close()

	oids := make([]string, len(s.config.Oids))
	oidMap := make(map[string]string)
	for i, o := range s.config.Oids {
		oids[i] = o.Oid
		oidMap[o.Oid] = o.Name
	}

	result, err := g.Get(oids)
	if err != nil {
		return TargetResult{Success: false, Error: fmt.Sprintf("GET failed: %v", err)}
	}

	values := make(map[string]any)
	for _, pdu := range result.Variables {
		name := oidMap[pdu.Name]
		if name == "" {
			name = pdu.Name
		}
		values[name] = pdu.Value
	}

	return TargetResult{
		Success: true,
		Values:  values,
	}
}

func convertToFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	default:
		return 0, false
	}
}

func (s *SnmpCheck) Shutdown() {
	s.DoneChan <- struct{}{}
}

func (s *SnmpCheck) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		s.config = *c
		return nil
	}
	return fmt.Errorf("config mismatch")
}

func (s *SnmpCheck) GetConfig() checks.Runtime {
	return &s.config
}

func (s *SnmpCheck) Name() string {
	return CheckName
}

func (s *SnmpCheck) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[SnmpResult](SnmpResult{})
}

func (s *SnmpCheck) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{s.metrics}
}

func (s *SnmpCheck) RemoveLabelledMetrics(target string) error {
	s.metrics.status.DeleteLabelValues(target)
	s.metrics.latency.DeleteLabelValues(target)
	// For the value metric, we have an additional label (oid_name)
	// We might need to iterate or just accept that it stays until next poll
	return nil
}
