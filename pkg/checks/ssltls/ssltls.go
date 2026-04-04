// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package ssltls

import (
	"context"
	"crypto/tls"
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
	_ checks.Check   = (*SslTls)(nil)
	_ checks.Runtime = (*Config)(nil)
)

type SslTls struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

type metrics struct {
	expiryDays *prometheus.GaugeVec
	status     *prometheus.GaugeVec
}

func newMetrics() metrics {
	return metrics{
		expiryDays: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_ssl_expiry_days",
				Help: "Days until SSL certificate expires",
			},
			[]string{"target"},
		),
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_ssl_status",
				Help: "Status of SSL check (1 for success, 0 for failure)",
			},
			[]string{"target"},
		),
	}
}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	m.expiryDays.Describe(ch)
	m.status.Describe(ch)
}

func (m metrics) Collect(ch chan<- prometheus.Metric) {
	m.expiryDays.Collect(ch)
	m.status.Collect(ch)
}

func NewCheck() checks.Check {
	return &SslTls{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		metrics: newMetrics(),
	}
}

func (s *SslTls) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.DoneChan:
			return nil
		case <-time.After(s.config.Interval):
			log.Debug("Running ssl_tls check")
			res := s.check(ctx)
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

func (s *SslTls) check(ctx context.Context) map[string]float64 {
	results := make(map[string]float64)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, t := range s.config.Targets {
		wg.Add(1)
		target := t
		go func() {
			defer wg.Done()
			expiry, err := getCertExpiry(target, s.config.Timeout)

			success := 1.0
			if err != nil {
				success = 0.0
				expiry = -1.0
			}

			mu.Lock()
			results[target] = expiry
			mu.Unlock()

			s.metrics.expiryDays.WithLabelValues(target).Set(expiry)
			s.metrics.status.WithLabelValues(target).Set(success)
		}()
	}
	wg.Wait()
	return results
}

func getCertExpiry(target string, timeout time.Duration) (float64, error) {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}

	config := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	tlsConn := tls.Client(conn, config)
	if err := tlsConn.Handshake(); err != nil {
		return 0, err
	}

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return 0, fmt.Errorf("no certificates found")
	}

	expiry := certs[0].NotAfter
	days := time.Until(expiry).Hours() / 24
	return days, nil
}

func (s *SslTls) Shutdown() {
	s.DoneChan <- struct{}{}
}

func (s *SslTls) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		s.config = *c
		return nil
	}
	return fmt.Errorf("config mismatch")
}

func (s *SslTls) GetConfig() checks.Runtime {
	return &s.config
}

func (s *SslTls) Name() string {
	return CheckName
}

func (s *SslTls) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]float64](map[string]float64{})
}

func (s *SslTls) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{s.metrics}
}

func (s *SslTls) RemoveLabelledMetrics(target string) error {
	s.metrics.expiryDays.DeleteLabelValues(target)
	s.metrics.status.DeleteLabelValues(target)
	return nil
}
