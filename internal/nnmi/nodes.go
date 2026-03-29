package nnmi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Node struct {
	UUID              string `json:"uuid"`
	Name              string `json:"name"`
	Hostname          string `json:"hostname"`
	SystemName        string `json:"systemName"`
	SystemDescription string `json:"systemDescription"`
	SystemLocation    string `json:"systemLocation"`
	SystemContact     string `json:"systemContact"`
	Status            string `json:"status"`
	ManagementMode    string `json:"managementMode"`
	DiscoveryState    string `json:"discoveryState"`
	IsSnmpSupported   bool   `json:"isSnmpSupported"`
	DeviceCategory    string `json:"deviceCategory"`
	DeviceVendor      string `json:"deviceVendor"`
	DeviceFamily      string `json:"deviceFamily"`
	Notes             string `json:"notes"`
}

type Incident struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Severity            string `json:"severity"`
	Priority            string `json:"priority"`
	FormattedMessage    string `json:"formattedMessage"`
	SourceNodeName      string `json:"sourceNodeName"`
	FirstOccurrenceTime string `json:"firstOccurrenceTime"`
	LastOccurrenceTime  string `json:"lastOccurrenceTime"`
	LifecycleState      string `json:"lifecycleState"`
}

// Get all nodes with filtering (PDF page 185)
func (c *NNMIClient) GetNodes(ctx context.Context, filter map[string]string) ([]Node, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/nnmi/api/topo/v1/node", c.baseURL)
	if len(filter) > 0 {
		params := url.Values{}
		for k, v := range filter {
			params.Set(k, v)
		}
		urlStr += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get nodes: status %d", resp.StatusCode)
	}

	var result struct {
		Embedded struct {
			Items []Node `json:"items"`
		} `json:"_embedded"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedded.Items, nil
}

// Get Open Incidents
func (c *NNMIClient) GetOpenKeyIncidents(ctx context.Context) ([]Incident, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	reqUrl := fmt.Sprintf("%s/nnmi/api/topo/v1/incident?lifecycleState=OPEN", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get incidents: status %d", resp.StatusCode)
	}

	var result struct {
		Embedded struct {
			Items []Incident `json:"items"`
		} `json:"_embedded"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedded.Items, nil
}

// Get network path between two nodes (PDF page 23)
type NetworkPath struct {
	Source      string `json:"pathSource"`
	Destination string `json:"pathDest"`
	Elements    []struct {
		Type       string `json:"elementType"`
		ID         string `json:"persistId"`
		Label      string `json:"label"`
		MACAddress string `json:"macAddress"`
		IsSource   bool   `json:"isSource"`
		IsDest     bool   `json:"isDestination"`
	} `json:"pathElements"`
}

func (c *NNMIClient) GetNetworkPath(ctx context.Context, source, dest string) (*NetworkPath, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/TopologyBeanService/TopologyBean?wsdl", c.baseURL), nil)
	if err != nil {
		return nil, err
	}
	_ = req

	// This is a SOAP call - would need XML marshaling
	// Simplified here - actual implementation would use SOAP client

	return nil, fmt.Errorf("SOAP implementation needed")
}
