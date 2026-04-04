// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package nnmi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
)

type Config struct {
	Enabled  bool          `yaml:"enabled" mapstructure:"enabled"`
	Host     string        `yaml:"host" mapstructure:"host"`
	Port     int           `yaml:"port" mapstructure:"port"`
	UseSSL   bool          `yaml:"use_ssl" mapstructure:"use_ssl"`
	Username string        `yaml:"username" mapstructure:"username"`
	Password string        `yaml:"password" mapstructure:"password"`
	Timeout  time.Duration `yaml:"timeout_seconds" mapstructure:"timeout_seconds"`
	CacheTTL time.Duration `yaml:"cache_ttl_minutes" mapstructure:"cache_ttl_minutes"`
}

type NNMIClient struct {
	baseURL    string
	username   string
	password   string
	token      string
	tokenExp   time.Time
	httpClient *http.Client
	mu         sync.RWMutex
	ipCache    map[string]*CachedDevice
	cacheTTL   time.Duration
}

type CachedDevice struct {
	Device    *NNMIDevice
	ExpiresAt time.Time
}

type NNMIDevice struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname"`
	Status     string `json:"status"`
	DeviceType string `json:"device_type"`
	Vendor     string `json:"vendor"`
	InNNMi     bool   `json:"in_nnmi"`
}

func NewClient(cfg Config) *NNMIClient {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port)

	return &NNMIClient{
		baseURL:  baseURL,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		ipCache:  make(map[string]*CachedDevice),
		cacheTTL: cfg.CacheTTL,
	}
}

func (c *NNMIClient) FindDeviceByIP(ctx context.Context, ip string) (*NNMIDevice, error) {
	c.mu.RLock()
	cached, exists := c.ipCache[ip]
	c.mu.RUnlock()

	if exists && time.Now().Before(cached.ExpiresAt) {
		return cached.Device, nil
	}

	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/nnmi/api/topo/v1/node?managementAddress=%s", c.baseURL, url.QueryEscape(ip))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.cacheIP(ip, nil)
		return &NNMIDevice{InNNMi: false}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NNMi IP lookup failed with status: %d", resp.StatusCode)
	}

	var ipResult struct {
		Links struct {
			Items []struct {
				Href  string `json:"href"`
				Title string `json:"title"`
			} `json:"items"`
		} `json:"_links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ipResult); err != nil {
		return nil, err
	}

	if len(ipResult.Links.Items) == 0 {
		c.cacheIP(ip, nil)
		return &NNMIDevice{InNNMi: false}, nil
	}

	// href contains something like "/nnmi/api/topo/v1/node/6e4fdf30-7ddc-45a9-939c-ff4223f2d31c"
	href := ipResult.Links.Items[0].Href
	parts := strings.Split(href, "/")
	nodeUUID := parts[len(parts)-1]
	
	device, err := c.getNodeByUUID(ctx, nodeUUID)
	if err != nil {
		return nil, err
	}

	device.InNNMi = true
	c.cacheIP(ip, device)
	return device, nil
}

func (c *NNMIClient) getNodeByUUID(ctx context.Context, uuid string) (*NNMIDevice, error) {
	nodeURL := fmt.Sprintf("%s/nnmi/api/topo/v1/node/%s", c.baseURL, uuid)
	req, err := http.NewRequestWithContext(ctx, "GET", nodeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NNMi Node lookup failed with status: %d", resp.StatusCode)
	}

	var nodeData struct {
		UUID           string `json:"uuid"`
		Name           string `json:"name"`
		Hostname       string `json:"hostname"`
		Status         string `json:"status"`
		DeviceCategory string `json:"deviceCategory"`
		DeviceVendor   string `json:"deviceVendor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&nodeData); err != nil {
		return nil, err
	}

	return &NNMIDevice{
		UUID:       nodeData.UUID,
		Name:       nodeData.Name,
		Hostname:   nodeData.Hostname,
		Status:     nodeData.Status,
		DeviceType: nodeData.DeviceCategory,
		Vendor:     nodeData.DeviceVendor,
	}, nil
}

func (c *NNMIClient) cacheIP(ip string, device *NNMIDevice) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ipCache[ip] = &CachedDevice{
		Device:    device,
		ExpiresAt: time.Now().Add(c.cacheTTL),
	}
}

func (c *NNMIClient) authenticate(ctx context.Context) error {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/idp/oauth2/token", c.baseURL),
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to authenticate: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.token = tokenResp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	logger.FromContext(ctx).Debug("NNMi OAuth2 token refreshed")
	return nil
}

func (c *NNMIClient) ensureValidToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExp) {
		return nil
	}
	return c.authenticate(ctx)
}

func (c *NNMIClient) GetBaseURL() string {
	return c.baseURL
}
