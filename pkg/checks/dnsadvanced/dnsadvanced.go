// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dnsadvanced

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

var (
	_ checks.Check   = (*DnsAdvanced)(nil)
	_ checks.Runtime = (*Config)(nil)
)

type DnsAdvanced struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

type metrics struct {
	status   *prometheus.GaugeVec
	duration *prometheus.GaugeVec
}

func newMetrics() metrics {
	return metrics{
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_dns_advanced_status",
				Help: "Status of the DNS check (1 for success, 0 for failure)",
			},
			[]string{"target", "type", "resolver"},
		),
		duration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_dns_advanced_duration_seconds",
				Help: "Duration of the DNS check in seconds",
			},
			[]string{"target", "type", "resolver"},
		),
	}
}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	m.status.Describe(ch)
	m.duration.Describe(ch)
}

func (m metrics) Collect(ch chan<- prometheus.Metric) {
	m.status.Collect(ch)
	m.duration.Collect(ch)
}

func NewCheck() checks.Check {
	return &DnsAdvanced{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		metrics: newMetrics(),
	}
}

func (d *DnsAdvanced) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.DoneChan:
			return nil
		case <-time.After(d.config.Interval):
			log.Debug("Running dns_advanced check")
			res := d.check(ctx)
			cResult <- checks.ResultDTO{
				Name: d.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
		}
	}
}

type DnsResult struct {
	Status     string  `json:"status"`
	DurationMs float64 `json:"duration_ms"`
}

func (d *DnsAdvanced) check(ctx context.Context) map[string]any {
	results := make(map[string]any)
	var wg sync.WaitGroup
	var mu sync.Mutex

	resolvers := d.config.Resolvers
	if len(resolvers) == 0 {
		resolvers = []string{"8.8.8.8:53"}
	}

	for _, q := range d.config.Queries {
		for _, r := range resolvers {
			wg.Add(1)
			query := q
			resolver := r
			if _, _, err := net.SplitHostPort(resolver); err != nil {
				resolver = net.JoinHostPort(resolver, "53")
			}

			go func() {
				defer wg.Done()
				start := time.Now()
				err := doDnsQuery(query, resolver, d.config.Timeout)
				duration := time.Since(start)

				success := 1.0
				status := "OK"
				if err != nil {
					success = 0.0
					status = err.Error()
				}

				key := fmt.Sprintf("%s-%s-%s", query.Name, query.Type, resolver)
				mu.Lock()
				results[key] = map[string]any{
					"status":      status,
					"duration_ms": float64(duration.Nanoseconds()) / 1e6,
				}
				mu.Unlock()

				d.metrics.status.WithLabelValues(query.Name, query.Type, resolver).Set(success)
				d.metrics.duration.WithLabelValues(query.Name, query.Type, resolver).Set(duration.Seconds())
			}()
		}
	}
	wg.Wait()
	return results
}

func doDnsQuery(q Query, resolver string, timeout time.Duration) error {
	c := new(dns.Client)
	c.Timeout = timeout

	m := new(dns.Msg)
	qType := dns.StringToType[q.Type]
	if qType == 0 {
		qType = dns.TypeA
	}
	m.SetQuestion(dns.Fqdn(q.Name), qType)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, resolver)
	if err != nil {
		return err
	}

	if r.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("DNS error: %s", dns.RcodeToString[r.Rcode])
	}

	return nil
}

func (d *DnsAdvanced) Shutdown() {
	d.DoneChan <- struct{}{}
}

func (d *DnsAdvanced) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		d.Mu.Lock()
		defer d.Mu.Unlock()
		d.config = *c
		return nil
	}
	return fmt.Errorf("config mismatch")
}

func (d *DnsAdvanced) GetConfig() checks.Runtime {
	return &d.config
}

func (d *DnsAdvanced) Name() string {
	return CheckName
}

func (d *DnsAdvanced) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]any](map[string]any{})
}

func (d *DnsAdvanced) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{d.metrics}
}

func (d *DnsAdvanced) RemoveLabelledMetrics(target string) error {
	d.metrics.status.DeletePartialMatch(prometheus.Labels{"target": target})
	d.metrics.duration.DeletePartialMatch(prometheus.Labels{"target": target})
	return nil
}
