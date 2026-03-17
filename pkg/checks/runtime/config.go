// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"errors"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/dns"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/dnsadvanced"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/health"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/httpadvanced"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/latency"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/ssltls"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/tcpmeter"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/traceroute"
)

// Config holds the runtime configuration
// for the various checks
// the hawkeye supports
type Config struct {
	Health       *health.Config       `yaml:"health" json:"health"`
	Latency      *latency.Config      `yaml:"latency" json:"latency"`
	Dns          *dns.Config          `yaml:"dns" json:"dns"`
	Traceroute   *traceroute.Config   `yaml:"traceroute" json:"traceroute"`
	HttpAdvanced *httpadvanced.Config `yaml:"http_advanced" json:"http_advanced"`
	SslTls       *ssltls.Config       `yaml:"ssl_tls" json:"ssl_tls"`
	DnsAdvanced  *dnsadvanced.Config  `yaml:"dns_advanced" json:"dns_advanced"`
	TcpMetrics   *tcpmeter.Config     `yaml:"tcp_metrics" json:"tcp_metrics"`
}

// Empty returns true if no checks are configured
func (c Config) Empty() bool {
	return c.size() == 0
}

func (c Config) Validate() (err error) {
	for _, cfg := range c.Iter() {
		if vErr := cfg.Validate(); vErr != nil {
			err = errors.Join(err, vErr)
		}
	}

	return err
}

// Iter returns configured checks in an iterable format
func (c Config) Iter() []checks.Runtime {
	var configs []checks.Runtime
	if c.Health != nil {
		configs = append(configs, c.Health)
	}
	if c.Latency != nil {
		configs = append(configs, c.Latency)
	}
	if c.Dns != nil {
		configs = append(configs, c.Dns)
	}
	if c.Traceroute != nil {
		configs = append(configs, c.Traceroute)
	}
	if c.HttpAdvanced != nil {
		configs = append(configs, c.HttpAdvanced)
	}
	if c.SslTls != nil {
		configs = append(configs, c.SslTls)
	}
	if c.DnsAdvanced != nil {
		configs = append(configs, c.DnsAdvanced)
	}
	if c.TcpMetrics != nil {
		configs = append(configs, c.TcpMetrics)
	}
	return configs
}

// size returns the number of checks configured
func (c Config) size() int {
	size := 0
	if c.HasHealthCheck() {
		size++
	}
	if c.HasLatencyCheck() {
		size++
	}
	if c.HasDNSCheck() {
		size++
	}
	if c.HasTracerouteCheck() {
		size++
	}
	if c.HttpAdvanced != nil {
		size++
	}
	if c.SslTls != nil {
		size++
	}
	if c.DnsAdvanced != nil {
		size++
	}
	if c.TcpMetrics != nil {
		size++
	}
	return size
}

// HasHealthCheck returns true if the check has a health check configured
func (c Config) HasHealthCheck() bool {
	return c.Health != nil
}

// HasLatencyCheck returns true if the check has a latency check configured
func (c Config) HasLatencyCheck() bool {
	return c.Latency != nil
}

// HasDNSCheck returns true if the check has a dns check configured
func (c Config) HasDNSCheck() bool {
	return c.Dns != nil
}

// HasTracerouteCheck returns true if the check has a traceroute check configured
func (c Config) HasTracerouteCheck() bool {
	return c.Traceroute != nil
}

// HasCheck returns true if the check has a check with the given name configured
func (c Config) HasCheck(name string) bool {
	switch name {
	case health.CheckName:
		return c.HasHealthCheck()
	case latency.CheckName:
		return c.HasLatencyCheck()
	case dns.CheckName:
		return c.HasDNSCheck()
	case traceroute.CheckName:
		return c.HasTracerouteCheck()
	case httpadvanced.CheckName:
		return c.HttpAdvanced != nil
	case ssltls.CheckName:
		return c.SslTls != nil
	case dnsadvanced.CheckName:
		return c.DnsAdvanced != nil
	case tcpmeter.CheckName:
		return c.TcpMetrics != nil
	default:
		return false
	}
}

// For returns the runtime configuration for the check with the given name
func (c Config) For(name string) checks.Runtime {
	switch name {
	case health.CheckName:
		if c.HasHealthCheck() {
			return c.Health
		}
	case latency.CheckName:
		if c.HasLatencyCheck() {
			return c.Latency
		}
	case dns.CheckName:
		if c.HasDNSCheck() {
			return c.Dns
		}
	case traceroute.CheckName:
		if c.HasTracerouteCheck() {
			return c.Traceroute
		}
	case httpadvanced.CheckName:
		if c.HttpAdvanced != nil {
			return c.HttpAdvanced
		}
	case ssltls.CheckName:
		if c.SslTls != nil {
			return c.SslTls
		}
	case dnsadvanced.CheckName:
		if c.DnsAdvanced != nil {
			return c.DnsAdvanced
		}
	case tcpmeter.CheckName:
		if c.TcpMetrics != nil {
			return c.TcpMetrics
		}
	}
	return nil
}
