// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package tcpmeter

import (
	"time"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

const CheckName = "tcp_metrics"

type Config struct {
	Targets    []string      `json:"targets" yaml:"targets"`
	Interval   time.Duration `json:"interval" yaml:"interval"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	BannerGrab bool          `json:"banner_grab" yaml:"banner_grab"`
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
