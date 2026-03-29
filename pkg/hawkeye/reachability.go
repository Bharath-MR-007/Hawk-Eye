// pkg/hawkeye/reachability.go
package hawkeye

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/Bharath-MR-007/hawk-eye/pkg/config"
	"gopkg.in/yaml.v3"
)

type ReachabilityRequest struct {
	Target   string `json:"target"`
	Protocol string `json:"protocol"` // icmp, snmp, tcp
	Timeout  int    `json:"timeout"`  // ms
}

type ReachabilityResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Latency float64 `json:"latency"` // ms
}

func (s *Hawkeye) handleReachabilityTest(w http.ResponseWriter, r *http.Request) {
	var req ReachabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 1000
	}

	var res ReachabilityResponse

	switch strings.ToLower(req.Protocol) {
	case "icmp":
		res = s.testICMP(req.Target, req.Timeout)
	case "snmp":
		res = s.testSNMP(req.Target, req.Timeout)
	case "tcp":
		res = s.testTCP(req.Target, req.Timeout)
	case "netconf":
		res = s.testNetConf(req.Target, req.Timeout)
	default:
		res = ReachabilityResponse{Success: false, Message: "Unsupported protocol: " + req.Protocol}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Hawkeye) testICMP(target string, timeout int) ReachabilityResponse {
	start := time.Now()
	
	// Use OS ping command for simplicity and to avoid raw socket privileges in container/mac
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", strconv.Itoa(timeout), target)
	} else if runtime.GOOS == "darwin" {
		// macOS
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeout), target)
	} else {
		// Linux/RHEL (iputils-ping -W takes seconds, not milliseconds)
		timeoutSec := timeout / 1000
		if timeoutSec < 1 {
			timeoutSec = 1
		}
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), target)
	}

	err := cmd.Run()
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: "Host unreachable or timed out",
			Latency: latency,
		}
	}

	return ReachabilityResponse{
		Success: true,
		Message: "ICMP Echo Reply received",
		Latency: latency,
	}
}

func (s *Hawkeye) testTCP(target string, timeout int) ReachabilityResponse {
	host := target
	port := "80"

	// If target has port, split it
	if strings.Contains(target, ":") {
		h, p, err := net.SplitHostPort(target)
		if err == nil {
			host = h
			port = p
		}
	} else {
		// Default to 80, try 443 if no response on 80? No, let's just stay simple.
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Duration(timeout)*time.Millisecond)
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: fmt.Sprintf("TCP Connection failed to %s: %v", port, err),
			Latency: latency,
		}
	}
	defer conn.Close()

	return ReachabilityResponse{
		Success: true,
		Message: fmt.Sprintf("TCP Connected to port %s", port),
		Latency: latency,
	}
}

func (s *Hawkeye) testNetConf(target string, timeout int) ReachabilityResponse {
	host := target
	port := "830"

	// If target has port, split it
	if strings.Contains(target, ":") {
		h, p, err := net.SplitHostPort(target)
		if err == nil {
			host = h
			port = p
		}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Duration(timeout)*time.Millisecond)
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: fmt.Sprintf("NetConf Connection failed to %s: %v", port, err),
			Latency: latency,
		}
	}
	defer conn.Close()

	return ReachabilityResponse{
		Success: true,
		Message: fmt.Sprintf("NetConf Connected to port %s (SSH subsystem)", port),
		Latency: latency,
	}
}

func (s *Hawkeye) testSNMP(target string, timeout int) ReachabilityResponse {
	// Try to use community from config if available
	community := "public"
	b, err := os.ReadFile("snmp_config.yaml")
	if err == nil {
		var snmpCfg config.SnmpConfig
		if err := yaml.Unmarshal(b, &snmpCfg); err == nil && snmpCfg.Community != "" {
			community = snmpCfg.Community
		}
	}

	g := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161, // Default SNMP port
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(timeout) * time.Millisecond,
		Retries:   1,
	}

	start := time.Now()
	err = g.Connect()
	if err != nil {
		return ReachabilityResponse{Success: false, Message: "SNMP Connection Error: " + err.Error(), Latency: float64(time.Since(start).Milliseconds())}
	}
	defer g.Conn.Close()

	// Perform a simple Get for sysUpTime.0
	oid := ".1.3.6.1.2.1.1.3.0"
	result, err := g.Get([]string{oid})
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: "SNMP Get failed: " + err.Error(),
			Latency: latency,
		}
	}

	if len(result.Variables) > 0 && result.Variables[0].Value != nil {
		return ReachabilityResponse{
			Success: true,
			Message: "SNMP Response received (sysUpTime found)",
			Latency: latency,
		}
	}

	return ReachabilityResponse{
		Success: false,
		Message: "SNMP Target reachable but returned empty data",
		Latency: latency,
	}
}
