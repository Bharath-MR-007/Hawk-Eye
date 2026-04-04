// pkg/hawkeye/reachability.go
package hawkeye

import (
	"crypto/tls"
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
	Target    string `json:"target"`
	Protocol  string `json:"protocol"` // icmp, snmp, tcp
	Timeout   int    `json:"timeout"`  // ms
	Community string `json:"community,omitempty"`
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
		res = s.testSNMP(req.Target, req.Timeout, req.Community)
	case "tcp":
		res = s.testTCP(req.Target, req.Timeout)
	case "netconf":
		res = s.testNetConf(req.Target, req.Timeout)
	case "ssl":
		res = s.testSSL(req.Target, req.Timeout)
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

	out, err := cmd.CombinedOutput()
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: fmt.Sprintf("Host unreachable: %v, %s", err, string(out)),
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

func (s *Hawkeye) testSNMP(target string, timeout int, communityStr string) ReachabilityResponse {
	community := "public"

	// If GUI provided a community, use it directly
	if communityStr != "" {
		community = communityStr
	} else {
		// Try to use community from config if available
		b, err := os.ReadFile("snmp_config.yaml")
		if err == nil {
			var snmpCfg config.SnmpConfig
			if err := yaml.Unmarshal(b, &snmpCfg); err == nil && snmpCfg.Community != "" {
				community = snmpCfg.Community
			}
		}
	}
	fmt.Printf("DEBUG: Executing SNMP test against %s with community %q\n", target, community)

	g := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161, // Default SNMP port
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(timeout) * time.Millisecond,
		Retries:   1,
	}

	start := time.Now()
	err := g.Connect()
	if err != nil {
		return ReachabilityResponse{Success: false, Message: "SNMP Connection Error: " + err.Error(), Latency: float64(time.Since(start).Milliseconds())}
	}
	defer g.Conn.Close()

	// Perform a simple Get for sysUpTime.0 and sysName.0
	oidUpTime := ".1.3.6.1.2.1.1.3.0"
	oidSysName := ".1.3.6.1.2.1.1.5.0"
	result, err := g.Get([]string{oidUpTime, oidSysName})
	latency := float64(time.Since(start).Milliseconds())

	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: "SNMP Get failed: " + err.Error(),
			Latency: latency,
		}
	}

	if len(result.Variables) > 0 {
		sysName := "Unknown"
		for _, v := range result.Variables {
			// Name might come back with a leading dot or without depending on gosnmp internals
			if strings.HasSuffix(v.Name, "1.3.6.1.2.1.1.5.0") && v.Value != nil {
				if b, ok := v.Value.([]byte); ok {
					sysName = string(b)
				} else if s, ok := v.Value.(string); ok {
					sysName = s
				}
			}
		}

		return ReachabilityResponse{
			Success: true,
			Message: fmt.Sprintf("SNMP Success (sysName: %s)", sysName),
			Latency: latency,
		}
	}

	return ReachabilityResponse{
		Success: false,
		Message: "SNMP Target reachable but returned empty data",
		Latency: latency,
	}
}

func (s *Hawkeye) testSSL(target string, timeout int) ReachabilityResponse {
	fmt.Printf("DEBUG: Executing SSL test against %s\n", target)
	host := target
	port := "443"

	if strings.Contains(target, ":") {
		h, p, err := net.SplitHostPort(target)
		if err == nil {
			host = h
			port = p
		}
	}

	start := time.Now()
	// Dial with timeout
	dialer := &net.Dialer{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: fmt.Sprintf("SSL Connection failed: %v", err),
			Latency: float64(time.Since(start).Milliseconds()),
		}
	}
	defer conn.Close()

	// TLS handshake
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	})
	
	err = tlsConn.Handshake()
	latency := float64(time.Since(start).Milliseconds())
	
	if err != nil {
		return ReachabilityResponse{
			Success: false,
			Message: fmt.Sprintf("TLS Handshake failed: %v", err),
			Latency: latency,
		}
	}

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return ReachabilityResponse{
			Success: false,
			Message: "No certificates found",
			Latency: latency,
		}
	}

	expiry := certs[0].NotAfter
	days := int(time.Until(expiry).Hours() / 24)

	return ReachabilityResponse{
		Success: true,
		Message: fmt.Sprintf("SSL Valid (Expires in %d days)", days),
		Latency: latency,
	}
}
