// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package httpadvanced

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

var (
	_ checks.Check   = (*HttpAdvanced)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "http_advanced"

type HttpAdvanced struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

type metrics struct {
	status  *prometheus.GaugeVec
	latency *prometheus.GaugeVec
}

func newMetrics() metrics {
	return metrics{
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_http_advanced_status",
				Help: "Status of the advanced HTTP check (1 for success, 0 for failure)",
			},
			[]string{"target"},
		),
		latency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hawkeye_http_advanced_latency_seconds",
				Help: "Latency of the advanced HTTP check in seconds",
			},
			[]string{"target"},
		),
	}
}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	m.status.Describe(ch)
	m.latency.Describe(ch)
}

func (m metrics) Collect(ch chan<- prometheus.Metric) {
	m.status.Collect(ch)
	m.latency.Collect(ch)
}

func NewCheck() checks.Check {
	return &HttpAdvanced{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config:  Config{Retry: checks.DefaultRetry},
		metrics: newMetrics(),
	}
}

func (h *HttpAdvanced) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.DoneChan:
			return nil
		case <-time.After(h.config.Interval):
			log.Debug("Running http_advanced check")
			res := h.check(ctx)
			cResult <- checks.ResultDTO{
				Name: h.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
		}
	}
}

type HttpPerfTrace struct {
	Status string  `json:"status"`
	DNS    float64 `json:"dns_ms"`
	TCP    float64 `json:"tcp_ms"`
	TLS    float64 `json:"tls_ms"`
	TTFB   float64 `json:"ttfb_ms"`
	Total  float64 `json:"total_ms"`
}

func (h *HttpAdvanced) check(ctx context.Context) map[string]any {
	log := logger.FromContext(ctx)
	results := make(map[string]any)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, t := range h.config.Targets {
		wg.Add(1)
		target := t
		go func() {
			defer wg.Done()
			success := 1
			status := "healthy"

			perfTrace := HttpPerfTrace{}

			start := time.Now()
			err := h.doRequest(ctx, target, &perfTrace)
			duration := time.Since(start)
			perfTrace.Total = float64(duration.Milliseconds())

			if err != nil {
				success = 0
				status = err.Error()
				log.Warn("Advanced HTTP check failed", "target", target.Url, "error", err)
			}
			perfTrace.Status = status

			mu.Lock()
			results[target.Url] = perfTrace
			mu.Unlock()

			h.metrics.status.WithLabelValues(target.Url).Set(float64(success))
			h.metrics.latency.WithLabelValues(target.Url).Set(duration.Seconds())
		}()
	}
	wg.Wait()
	return results
}

func (h *HttpAdvanced) doRequest(ctx context.Context, t Target, perfTrace *HttpPerfTrace) error {
	client := &http.Client{
		Timeout: h.config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !t.FollowRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !t.SslVerify},
		},
	}

	req, err := http.NewRequestWithContext(ctx, t.Method, t.Url, strings.NewReader(t.Body))
	if err != nil {
		return err
	}

	var dnsStart, tcpStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { perfTrace.DNS = float64(time.Since(dnsStart).Milliseconds()) },

		ConnectStart: func(_, _ string) { tcpStart = time.Now() },
		ConnectDone:  func(_, _ string, _ error) { perfTrace.TCP = float64(time.Since(tcpStart).Milliseconds()) },

		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone:  func(_ tls.ConnectionState, _ error) { perfTrace.TLS = float64(time.Since(tlsStart).Milliseconds()) },

		GotFirstResponseByte: func() { perfTrace.TTFB = float64(time.Since(req.Context().Value("start").(time.Time)).Milliseconds()) },
	}

	req = req.WithContext(httptrace.WithClientTrace(context.WithValue(req.Context(), "start", time.Now()), trace))

	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if t.ExpectedStatus != 0 && resp.StatusCode != t.ExpectedStatus {
		return fmt.Errorf("expected status %d, got %d", t.ExpectedStatus, resp.StatusCode)
	}

	if t.ExpectedPattern != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		matched, err := regexp.Match(t.ExpectedPattern, body)
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("body did not match expected pattern")
		}
	}

	return nil
}

func (h *HttpAdvanced) Shutdown() {
	h.DoneChan <- struct{}{}
}

func (h *HttpAdvanced) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		h.Mu.Lock()
		defer h.Mu.Unlock()
		h.config = *c
		return nil
	}
	return fmt.Errorf("config mismatch")
}

func (h *HttpAdvanced) GetConfig() checks.Runtime {
	return &h.config
}

func (h *HttpAdvanced) Name() string {
	return CheckName
}

func (h *HttpAdvanced) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]any](map[string]any{})
}

func (h *HttpAdvanced) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{h.metrics}
}

func (h *HttpAdvanced) RemoveLabelledMetrics(target string) error {
	h.metrics.status.DeleteLabelValues(target)
	h.metrics.latency.DeleteLabelValues(target)
	return nil
}
