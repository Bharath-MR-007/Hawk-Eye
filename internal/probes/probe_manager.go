package probes

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ProbeResult struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Target    string                 `json:"target"`
	Duration  time.Duration          `json:"duration_ms"`
	Success   bool                   `json:"success"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ProbeManager manages all probes and broadcasts results to subscribed WebSocket clients.
type ProbeManager struct {
	probes  map[string]Probe
	mu      sync.RWMutex
	Results chan *ProbeResult // exported so WebSocket handler can subscribe

	// subscriber management
	subsMu      sync.RWMutex
	subscribers map[string]chan *ProbeResult

	// Prometheus metrics
	probeDuration *prometheus.HistogramVec
	probeErrors   *prometheus.CounterVec
	targetHealth  *prometheus.GaugeVec
}

// NewProbeManager creates an initialized ProbeManager with a running broadcast loop.
func NewProbeManager() *ProbeManager {
	pm := &ProbeManager{
		probes:      make(map[string]Probe),
		Results:     make(chan *ProbeResult, 256),
		subscribers: make(map[string]chan *ProbeResult),
	}
	go pm.broadcastLoop()
	return pm
}

// Subscribe returns a channel for a specific subscriber ID that receives all probe results.
// The caller must call Unsubscribe when done to avoid leaking goroutines.
func (pm *ProbeManager) Subscribe(id string) chan *ProbeResult {
	ch := make(chan *ProbeResult, 64)
	pm.subsMu.Lock()
	pm.subscribers[id] = ch
	pm.subsMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (pm *ProbeManager) Unsubscribe(id string) {
	pm.subsMu.Lock()
	defer pm.subsMu.Unlock()
	if ch, ok := pm.subscribers[id]; ok {
		close(ch)
		delete(pm.subscribers, id)
	}
}

// Publish sends a probe result to all subscribers.
func (pm *ProbeManager) Publish(result *ProbeResult) {
	// Non-blocking send to the main Results channel
	select {
	case pm.Results <- result:
	default:
	}
}

// broadcastLoop fans out results from the Results channel to all subscribers.
func (pm *ProbeManager) broadcastLoop() {
	for result := range pm.Results {
		pm.subsMu.RLock()
		for _, ch := range pm.subscribers {
			// non-blocking send per subscriber — slow clients just miss frames
			select {
			case ch <- result:
			default:
			}
		}
		pm.subsMu.RUnlock()
	}
}

type Probe interface {
	Name() string
	Type() string
	Run(ctx context.Context, target interface{}) (*ProbeResult, error)
	Interval() time.Duration
}

// Enhanced DNS probe with multiple resolvers
type DNSProbe struct {
	resolvers  []string
	queryTypes []string
	timeout    time.Duration
}

func (p *DNSProbe) Name() string            { return "dns_advanced_internal" }
func (p *DNSProbe) Type() string            { return "dns" }
func (p *DNSProbe) Interval() time.Duration { return 30 * time.Second }

func (p *DNSProbe) Run(ctx context.Context, target interface{}) (*ProbeResult, error) {
	result := &ProbeResult{
		Timestamp: time.Now(),
		Type:      "dns",
		Target:    target.(string),
	}

	details := make(map[string]interface{})

	resolverResults := make(map[string]interface{})
	for _, resolver := range p.resolvers {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: p.timeout}
				return d.DialContext(ctx, "udp", resolver+":53")
			},
		}

		start := time.Now()
		ips, err := r.LookupIPAddr(ctx, target.(string))
		duration := time.Since(start)

		errStr := ""
		if err != nil {
			errStr = err.Error()
		}

		resolverResults[resolver] = map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"ips":         ips,
			"error":       errStr,
		}
	}

	details["resolvers"] = resolverResults
	result.Details = details
	result.Success = true

	return result, nil
}

// Advanced TCP metrics probe
type TCPProbe struct {
	timeout time.Duration
}

func (p *TCPProbe) Name() string            { return "tcp_metrics_internal" }
func (p *TCPProbe) Type() string            { return "tcp" }
func (p *TCPProbe) Interval() time.Duration { return 30 * time.Second }

func (p *TCPProbe) Run(ctx context.Context, target interface{}) (*ProbeResult, error) {
	targetStr := target.(string)
	_, _, err := net.SplitHostPort(targetStr)
	if err != nil {
		return nil, err
	}

	result := &ProbeResult{
		Timestamp: time.Now(),
		Type:      "tcp",
		Target:    targetStr,
	}

	startConnect := time.Now()
	conn, err := net.DialTimeout("tcp", targetStr, p.timeout)
	connectTime := time.Since(startConnect)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}
	defer conn.Close()

	result.Success = true
	result.Details = map[string]interface{}{
		"connection_time_ms": connectTime.Milliseconds(),
		"local_addr":         conn.LocalAddr().String(),
		"remote_addr":        conn.RemoteAddr().String(),
	}
	return result, nil
}
