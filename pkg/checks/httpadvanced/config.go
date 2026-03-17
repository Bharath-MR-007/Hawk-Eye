// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package httpadvanced

import (
	"fmt"
	"net/url"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/helper"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

const (
	minInterval = 1 * time.Second
	minTimeout  = 1 * time.Second
)

type Target struct {
	Url               string            `json:"url" yaml:"url"`
	Method            string            `json:"method" yaml:"method"`
	Body              string            `json:"body,omitempty" yaml:"body,omitempty"`
	Headers           map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	ExpectedStatus    int               `json:"expected_status" yaml:"expected_status"`
	ExpectedPattern   string            `json:"expected_pattern,omitempty" yaml:"expected_pattern,omitempty"`
	FollowRedirects   bool              `json:"follow_redirects" yaml:"follow_redirects"`
	SslVerify         bool              `json:"ssl_verify" yaml:"ssl_verify"`
}

type Config struct {
	Targets  []Target           `json:"targets" yaml:"targets"`
	Interval time.Duration      `json:"interval" yaml:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	for _, t := range c.Targets {
		_, err := url.Parse(t.Url)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "invalid target URL"}
		}
	}

	if c.Interval < minInterval {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "interval", Reason: fmt.Sprintf("interval must be at least %v", minInterval)}
	}

	return nil
}
