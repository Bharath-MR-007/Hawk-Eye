// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dnsadvanced

import (
	"time"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

const CheckName = "dns_advanced"

type Query struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}

type Config struct {
	Targets        []string      `json:"targets" yaml:"targets"` // Deprecated, use Queries
	Queries        []Query       `json:"queries" yaml:"queries"`
	Resolvers      []string      `json:"resolvers" yaml:"resolvers"`
	Interval       time.Duration `json:"interval" yaml:"interval"`
	Timeout        time.Duration `json:"timeout" yaml:"timeout"`
	ValidateDnssec bool          `json:"validate_dnssec" yaml:"validate_dnssec"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Interval < time.Second {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "interval", Reason: "interval must be at least 1 second"}
	}
	return nil
}
