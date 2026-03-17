// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package ssltls

import (
	"time"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

const CheckName = "ssl_tls"

type Config struct {
	Targets             []string      `json:"targets" yaml:"targets"`
	Interval            time.Duration `json:"interval" yaml:"interval"`
	Timeout             time.Duration `json:"timeout" yaml:"timeout"`
	CheckExpiryDays     int           `json:"check_expiry_days" yaml:"check_expiry_days"`
	CheckProtocols      []string      `json:"check_protocols" yaml:"check_protocols"`
	CheckCipherStrength bool          `json:"check_cipher_strength" yaml:"check_cipher_strength"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Interval < time.Minute {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "interval", Reason: "interval must be at least 1 minute"}
	}
	for _, t := range c.Targets {
		if t == "" {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "target cannot be empty"}
		}
	}
	return nil
}
