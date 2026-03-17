// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/helper"
	"github.com/Bharath-MR-007/hawk-eye/pkg/api"
)

func TestConfig_Validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "config ok",
			config: Config{
				HawkeyeName: "hawkeye.com",
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "https://test.de/config",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},

			wantErr: false,
		},
		{
			name: "loader - url missing",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				HawkeyeName: "hawkeye.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},

			wantErr: true,
		},
		{
			name: "loader - url malformed",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				HawkeyeName: "hawkeye.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "this is not a valid url",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "loader - retry count to high",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				HawkeyeName: "hawkeye.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "test.de",
						Timeout: time.Minute,
						RetryCfg: helper.RetryConfig{
							Count: 100000,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "loader - file path malformed",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				HawkeyeName: "hawkeye.com",
				Loader: LoaderConfig{
					Type: "file",
					File: FileLoaderConfig{
						Path: "",
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "targetManager - Wrong Scheme",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				HawkeyeName: "hawkeye.com",
				Loader: LoaderConfig{
					Type: "file",
					File: FileLoaderConfig{
						Path: "",
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isDNSName(t *testing.T) {
	tests := []struct {
		name    string
		dnsName string
		want    bool
	}{
		{name: "dns name", dnsName: "hawkeye.de", want: true},
		{name: "dns name with subdomain", dnsName: "hawkeye.test.de", want: true},
		{name: "dns name with subdomain and tld and -", dnsName: "sub-hawkeye.test.de", want: true},
		{name: "empty name", dnsName: "", want: false},
		{name: "dns name without tld", dnsName: "hawkeye", want: false},
		{name: "name with underscore", dnsName: "test_de", want: false},
		{name: "name with space", dnsName: "test de", want: false},
		{name: "name with special chars", dnsName: "test!de", want: false},
		{name: "name with capitals", dnsName: "tEst.de", want: false},
		{name: "name with empty tld", dnsName: "tEst.de.", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDNSName(tt.dnsName); got != tt.want {
				t.Errorf("isDNSName() = %v, want %v", got, tt.want)
			}
		})
	}
}
