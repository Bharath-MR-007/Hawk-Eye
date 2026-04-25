// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package snmp

import (
	"time"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

// CheckName is the name of the SNMP check
const CheckName = "snmp"

// OidConfig represents a single OID to poll
type OidConfig struct {
	Name string `json:"name" yaml:"name"`
	Oid  string `json:"oid" yaml:"oid"`
}

// Config defines the configuration for the SNMP check
type Config struct {
	Targets   []string      `json:"targets" yaml:"targets"`
	Interval  time.Duration `json:"interval" yaml:"interval"`
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
	Community string        `json:"community" yaml:"community"`
	Version   string        `json:"version" yaml:"version"` // "v2c" or "v3"
	Port      int           `json:"port" yaml:"port"`
	Oids      []OidConfig   `json:"oids" yaml:"oids"`
}

// For returns the name of the check
func (c *Config) For() string {
	return CheckName
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Interval < time.Second {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "interval", Reason: "interval must be at least 1 second"}
	}
	if len(c.Oids) == 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "oids", Reason: "at least one OID must be configured"}
	}
	if c.Port <= 0 || c.Port > 65535 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "port", Reason: "invalid port number"}
	}
	if c.Community == "" {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "community", Reason: "community string cannot be empty"}
	}
	return nil
}
