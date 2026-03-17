// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package tcpmeter

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

var (
	_ checks.Check   = (*TcpMeter)(nil)
	_ checks.Runtime = (*Config)(nil)
)

type TcpMeter struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

type metrics struct {
	status *prometheus.GaugeVec
	timing *prometheus.GaugeVec
}

func newMetrics() metrics {
	return metrics{
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_tcp_status",
				Help: "Status of TCP connection (1 for success, 0 for failure)",
			},
			[]string{"target"},
		),
		timing: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_tcp_connect_seconds",
				Help: "Time to establish TCP connection in seconds",
			},
			[]string{"target"},
		),
	}
}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	m.status.Describe(ch)
	m.timing.Describe(ch)
}

func (m metrics) Collect(ch chan<- prometheus.Metric) {
	m.status.Collect(ch)
	m.timing.Collect(ch)
}

func NewCheck() checks.Check {
	return &TcpMeter{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		metrics: newMetrics(),
	}
}

func (t *TcpMeter) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.DoneChan:
			return nil
		case <-time.After(t.config.Interval):
			log.Debug("Running tcp_metrics check")
			res := t.check(ctx)
			cResult <- checks.ResultDTO{
				Name: t.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
		}
	}
}

func (t *TcpMeter) check(ctx context.Context) map[string]float64 {
	log := logger.FromContext(ctx)
	results := make(map[string]float64)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, target := range t.config.Targets {
		wg.Add(1)
		tgt := target
		go func() {
			defer wg.Done()
			start := time.Now()
			conn, err := net.DialTimeout("tcp", tgt, t.config.Timeout)
			duration := time.Since(start)

			success := 1.0
			var localAddr, remoteAddr string
			if err != nil {
				success = 0.0
				duration = 0
			} else {
				localAddr = conn.LocalAddr().String()
				remoteAddr = conn.RemoteAddr().String()
				conn.Close()
			}

			mu.Lock()
			results[tgt] = duration.Seconds()
			mu.Unlock()

			t.metrics.status.WithLabelValues(tgt).Set(success)
			t.metrics.timing.WithLabelValues(tgt).Set(duration.Seconds())

			// Log the details as requested in the probe logic
			if success == 1.0 {
				log.Debug("TCP connection established", "target", tgt, "local", localAddr, "remote", remoteAddr)
			}
		}()
	}
	wg.Wait()
	return results
}

func (t *TcpMeter) Shutdown() {
	t.DoneChan <- struct{}{}
}

func (t *TcpMeter) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		t.Mu.Lock()
		defer t.Mu.Unlock()
		t.config = *c
		return nil
	}
	return fmt.Errorf("config mismatch")
}

func (t *TcpMeter) GetConfig() checks.Runtime {
	return &t.config
}

func (t *TcpMeter) Name() string {
	return CheckName
}

func (t *TcpMeter) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]float64](map[string]float64{})
}

func (t *TcpMeter) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{t.metrics}
}

func (t *TcpMeter) RemoveLabelledMetrics(target string) error {
	t.metrics.status.DeleteLabelValues(target)
	t.metrics.timing.DeleteLabelValues(target)
	return nil
}
