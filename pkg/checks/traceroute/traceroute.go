// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"

	"github.com/Bharath-MR-007/hawk-eye/internal/helper"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/internal/nnmi"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// IPv4HeaderSize is the size of an IPv4 header in bytes
	IPv4HeaderSize = 20
	// IPv6HeaderSize is the size of an IPv6 header in bytes
	IPv6HeaderSize = 40
	// mtuSize is the maximum transmission unit size
	mtuSize = 1500
	// basePort is the starting port for the TCP connection
	basePort = 30000
	// portRange is the range of ports to generate a random port from
	portRange = 10000
)

// randomPort returns a random port in the interval [30_000, 40_000)
func randomPort() int {
	return rand.N(portRange) + basePort // #nosec G404 // math.rand is fine here, we're not doing encryption
}

// tcpHop attempts to connect to the target host using TCP with the specified TTL and timeout.
// It returns a [net.Conn], the port used for the connection, and an error if the connection failed.
func tcpHop(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (net.Conn, int, error) {
	span := trace.SpanFromContext(ctx)

	for {
		port := randomPort()

		// Dialer with control function to set IP_TTL
		dialer := net.Dialer{
			LocalAddr: &net.TCPAddr{
				Port: port,
			},
			Timeout: timeout,
			Control: func(_, _ string, c syscall.RawConn) error {
				var opErr error
				if err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl) // #nosec G115 // The net package is safe to use
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		span.AddEvent("Attempting TCP connection", trace.WithAttributes(
			attribute.String("remote_addr", addr.String()),
			attribute.Int("ttl", ttl),
			attribute.Int("port", port),
		))

		// Attempt to connect to the target host
		conn, err := dialer.DialContext(ctx, "tcp", addr.String())

		switch {
		case err == nil:
			span.AddEvent("TCP connection succeeded", trace.WithAttributes(
				attribute.Stringer("remote_addr", addr),
				attribute.Int("ttl", ttl),
				attribute.Int("port", port),
			))
			return conn, port, nil
		case errors.Is(err, unix.EADDRINUSE):
			// Address in use, retry by continuing the loop
			continue
		case errors.Is(err, unix.EHOSTUNREACH):
			// No route to host is a special error because of how tcp traceroute works
			// we are expecting the connection to fail because of TTL expiry
			span.SetStatus(codes.Error, "No route to host")
			span.AddEvent("No route to host", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			logger.FromContext(ctx).DebugContext(ctx, "No route to host", "error", err.Error())
			return conn, port, err
		default:
			span.AddEvent("TCP connection failed", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return conn, port, err
		}
	}
}

// udpHop attempts to send a UDP packet to the target host using the specified TTL and timeout.
// It returns a [net.Conn], the port used for the connection, and an error if it failed.
func udpHop(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (net.Conn, int, error) {
	span := trace.SpanFromContext(ctx)

	for {
		port := randomPort()

		// Dialer with control function to set IP_TTL
		dialer := net.Dialer{
			LocalAddr: &net.UDPAddr{
				Port: port,
			},
			Timeout: timeout,
			Control: func(_, _ string, c syscall.RawConn) error {
				var opErr error
				if err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl)
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		span.AddEvent("Attempting UDP connection", trace.WithAttributes(
			attribute.String("remote_addr", addr.String()),
			attribute.Int("ttl", ttl),
			attribute.Int("port", port),
		))

		// Attempt to "connect" (set destination) for the UDP packet
		conn, err := dialer.DialContext(ctx, "udp", addr.String())

		if err == nil {
			// Send a small dummy payload
			_, _ = conn.Write([]byte("hawk-eye"))
			// We don't need to keep the connection open as ICMP is handled separately
			_ = conn.Close()
			return nil, port, nil
		}

		if errors.Is(err, unix.EADDRINUSE) {
			continue
		}

		return nil, port, err
	}
}

// readIcmpMessage reads a packet from the provided [icmp.PacketConn]. If the packet is 'Time Exceeded',
// it reads the address of the router that dropped created the icmp packet. It also reads the source port
// from the payload and finds the source port used by the previous tcp connection. If any error is returned,
// an icmp packet was either not received, or the received packet was not a time exceeded.
// readIcmpMessage reads a packet from the provided [icmp.PacketConn].
// It returns the matched identifier (port or sequence), the router address, the message type, the message code, and an error.
func readIcmpMessage(ctx context.Context, icmpListener *icmp.PacketConn, timeout time.Duration, method string) (int, net.Addr, icmp.Type, int, error) {
	log := logger.FromContext(ctx)
	if err := icmpListener.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("failed to set icmp read deadline: %w", err)
	}
	buffer := make([]byte, mtuSize)
	n, routerAddr, err := icmpListener.ReadFrom(buffer)
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("failed to read from icmp connection: %w", err)
	}

	msg, err := icmp.ParseMessage(ipv4.ICMPTypeTimeExceeded.Protocol(), buffer[:n])
	if err != nil {
		return 0, nil, nil, 0, err
	}

	var data []byte
	switch msg.Type {
	case ipv4.ICMPTypeTimeExceeded:
		data = msg.Body.(*icmp.TimeExceeded).Data[IPv4HeaderSize:]
	case ipv6.ICMPTypeTimeExceeded:
		data = msg.Body.(*icmp.TimeExceeded).Data[IPv6HeaderSize:]
	case ipv4.ICMPTypeDestinationUnreachable:
		data = msg.Body.(*icmp.DstUnreach).Data[IPv4HeaderSize:]
	case ipv6.ICMPTypeDestinationUnreachable:
		data = msg.Body.(*icmp.DstUnreach).Data[IPv6HeaderSize:]
	case ipv4.ICMPTypeEchoReply:
		echo := msg.Body.(*icmp.Echo)
		return echo.Seq, routerAddr, msg.Type, msg.Code, nil
	case ipv6.ICMPTypeEchoReply:
		echo := msg.Body.(*icmp.Echo)
		return echo.Seq, routerAddr, msg.Type, msg.Code, nil
	default:
		log.DebugContext(ctx, "unhandled icmp message", "type", msg.Type.Protocol())
		return 0, nil, msg.Type, msg.Code, errors.New("unhandled icmp message")
	}

	// For Time Exceeded and Destination Unreachable, the data contains the original packet's header
	if method == "tcp" || method == "udp" {
		if len(data) >= 2 {
			// Both TCP and UDP have the source port as the first 2 bytes of the transport header
			matchedPort := int(data[0])<<8 + int(data[1])
			return matchedPort, routerAddr, msg.Type, msg.Code, nil
		}
	} else if method == "icmp" {
		// ICMP header in quoted data: Type(1), Code(1), Checksum(2), ID(2), Seq(2)
		// Seq is at offset 6
		if len(data) >= 8 {
			matchedSeq := int(data[6])<<8 + int(data[7])
			return matchedSeq, routerAddr, msg.Type, msg.Code, nil
		}
	}

	return 0, nil, msg.Type, msg.Code, errors.New("could not match icmp response")
}

// TraceRoute performs a traceroute to the specified host using TCP and listens for ICMP Time Exceeded messages using ICMP.
func TraceRoute(ctx context.Context, cfg TracerouteConfig) (map[int][]Hop, error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("tracer.traceroute")
	ctx, sp := tracer.Start(ctx, "TraceRoute", trace.WithAttributes(
		attribute.String("target", cfg.Dest),
		attribute.Int("port", cfg.Port),
		attribute.Int("max_hops", cfg.MaxHops),
		attribute.Stringer("timeout", cfg.Timeout),
	))
	defer sp.End()

	// maps ttl -> attempted hops for that ttl
	hops := make(map[int][]Hop)
	log := logger.FromContext(ctx).With("target", cfg.Dest)

	canIcmp, icmpListener, err := newIcmpListener()
	if err != nil {
		log.WarnContext(ctx, "Failed to open ICMP socket, traceroute will rely on TCP/UDP only", "err", err)
	} else {
		defer closeIcmpListener(canIcmp, icmpListener)
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", cfg.Dest, cfg.Port))
	if err != nil {
		sp.SetStatus(codes.Error, err.Error())
		sp.RecordError(err)
		log.ErrorContext(ctx, "failed to resolve target name", "err", err)
		return nil, err
	}

	queueSize := cfg.MaxHops * (1 + cfg.Rc.Count)
	results := make(chan Hop, queueSize)
	var wg sync.WaitGroup

	for ttl := 1; ttl <= cfg.MaxHops; ttl++ {
		wg.Add(1)
		go func(ttl int) {
			c, hopSpan := tracer.Start(ctx, addr.String(), trace.WithAttributes(
				attribute.Int("ttl", ttl),
			))
			defer wg.Done()
			defer hopSpan.End()

			l := log.With("ttl", ttl)
			logctx := logger.IntoContext(c, l)

			retry := 0
			retryErr := helper.Retry(func(ctx context.Context) error {
				defer func() {
					retry++
				}()
				hop, hErr := doHop(ctx, icmpListener, canIcmp, addr, ttl, cfg.Timeout, cfg.Method, cfg.NNMiClient)
				if hop != nil {
					results <- *hop
				}
				if hErr != nil {
					return hErr
				}
				// Traceroute doesn't necessarily need to "reach" the destination at every hop,
				// but it needs a response (either TTL Exceeded or reached).
				// We consider a hop "failed" only if we get no response at all.
				if hop.Addr.IP == "" {
					return errors.New("no response")
				}
				return nil
			}, cfg.Rc)(logctx)
			if retryErr != nil {
				l.DebugContext(ctx, "Traceroute could not reach target")
				if !errors.Is(retryErr, syscall.EHOSTUNREACH) {
					hopSpan.SetStatus(codes.Error, retryErr.Error())
					hopSpan.RecordError(retryErr)
				}
				return
			}
			hopSpan.SetStatus(codes.Ok, "Hop succeeded")
		}(ttl)
	}

	wg.Wait()
	close(results)

	// Collect and log hops
	for r := range results {
		hops[r.Ttl] = append(hops[r.Ttl], r)
	}
	logHops(ctx, hops)

	sp.AddEvent("TraceRoute completed", trace.WithAttributes(
		attribute.Int("hops_count", len(hops)),
	))
	return hops, nil
}

// doHop performs a hop to the given address with the specified TTL and timeout.
// It returns a Hop struct containing the latency, TTL, address, and other details of the hop.
func doHop(ctx context.Context, icmpListener *icmp.PacketConn, canIcmp bool, addr net.Addr, ttl int, timeout time.Duration, method string, nnmiClient *nnmi.NNMIClient) (*Hop, error) {
	span := trace.SpanFromContext(ctx)
	start := time.Now()
	var clientIdentifier int
	var conn net.Conn
	var err error

	switch method {
	case "tcp":
		conn, clientIdentifier, err = tcpHop(ctx, addr, ttl, timeout)
	case "udp":
		conn, clientIdentifier, err = udpHop(ctx, addr, ttl, timeout)
	case "icmp":
		clientIdentifier = ttl // Use TTL as sequence number
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: 1, Seq: clientIdentifier,
				Data: []byte("hawk-eye"),
			},
		}
		b, _ := msg.Marshal(nil)
		if canIcmp {
			_, err = icmpListener.WriteTo(b, &net.IPAddr{IP: net.ParseIP(ipFromAddr(addr).String())})
		}
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
	span.SetAttributes(attribute.Int("ttl", ttl), attribute.Stringer("addr", addr), attribute.String("method", method))
	if err == nil && (method == "tcp") {
		latency := time.Since(start)
		hop := handleTcpSuccess(conn, addr, ttl, latency)
		fillNNMiInfo(ctx, hop, nnmiClient)
		span.AddEvent("Hop succeeded (Reached)", trace.WithAttributes(
			attribute.String("hop_name", hop.Name),
			attribute.Stringer("hop_addr", hop.Addr),
		))
		return hop, nil
	}

	if !canIcmp {
		latency := time.Since(start)
		return &Hop{Latency: latency, Ttl: ttl, Reached: false}, nil
	}

	hop := handleIcmpResponse(ctx, icmpListener, clientIdentifier, ttl, timeout, method)
	hop.Latency = time.Since(start)
	fillNNMiInfo(ctx, &hop, nnmiClient)
	if !hop.Reached {
		span.AddEvent("ICMP hop not reached", trace.WithAttributes(
			attribute.String("hop_name", hop.Name),
			attribute.Stringer("hop_addr", hop.Addr),
			attribute.Stringer("latency", hop.Latency),
		))
		return &hop, nil
	}

	span.AddEvent("ICMP hop reached", trace.WithAttributes(
		attribute.String("hop_name", hop.Name),
		attribute.Stringer("hop_addr", hop.Addr),
		attribute.Stringer("latency", hop.Latency),
	))
	return &hop, nil
}

func fillNNMiInfo(ctx context.Context, hop *Hop, nnmiClient *nnmi.NNMIClient) {
	if nnmiClient == nil || hop.Addr.IP == "" {
		return
	}

	device, err := nnmiClient.FindDeviceByIP(ctx, hop.Addr.IP)
	if err == nil && device != nil {
		hop.NNMiDevice = &NNMIDeviceInfo{
			UUID:          device.UUID,
			Name:          device.Name,
			Hostname:      device.Hostname,
			Status:        device.Status,
			DeviceType:    device.DeviceType,
			Vendor:        device.Vendor,
			ManagementURL: fmt.Sprintf("%s/nnmi/faces/pages/nodeDetails.xhtml?nodeUuid=%s", nnmiClient.GetBaseURL(), device.UUID),
			InNNMi:        device.InNNMi,
		}
	}
}

// newIcmpListener creates a new ICMP listener and returns a boolean indicating if the necessary permissions were granted.
func newIcmpListener() (bool, *icmp.PacketConn, error) {
	icmpListener, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		if !errors.Is(err, unix.EPERM) {
			return false, nil, err
		}
		return false, nil, nil
	}
	return true, icmpListener, nil
}

// closeIcmpListener closes the ICMP listener if it is not nil and the permissions were granted.
func closeIcmpListener(canIcmp bool, icmpListener *icmp.PacketConn) {
	if canIcmp && icmpListener != nil {
		icmpListener.Close() // #nosec G104
	}
}

// newHopAddress creates a new HopAddress from a [net.Addr].
func newHopAddress(addr net.Addr) HopAddress {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return HopAddress{
			IP:   addr.IP.String(),
			Port: addr.Port,
		}
	case *net.TCPAddr:
		return HopAddress{
			IP:   addr.IP.String(),
			Port: addr.Port,
		}
	case *net.IPAddr:
		return HopAddress{
			IP: addr.IP.String(),
		}
	default:
		return HopAddress{}
	}
}

// handleTcpSuccess handles a successful TCP connection by closing the connection and returning a Hop struct.
func handleTcpSuccess(conn net.Conn, addr net.Addr, ttl int, latency time.Duration) *Hop {
	conn.Close() // #nosec G104

	ipaddr := ipFromAddr(addr)
	names, _ := net.LookupAddr(ipaddr.String()) // we don't care about this lookup failing

	name := ""
	if len(names) >= 1 {
		name = names[0]
	}

	return &Hop{
		Latency: latency,
		Ttl:     ttl,
		Addr:    newHopAddress(addr),
		Name:    name,
		Reached: true,
	}
}

// handleIcmpResponse attempts to read an ICMP response that matches clientID
func handleIcmpResponse(ctx context.Context, icmpListener *icmp.PacketConn, clientID, ttl int, timeout time.Duration, method string) Hop {
	deadline := time.Now().Add(timeout)

	for time.Now().Unix() < deadline.Unix() {
		gotID, addr, msgType, msgCode, err := readIcmpMessage(ctx, icmpListener, timeout, method)
		if err != nil {
			continue
		}

		if gotID == clientID {
			ipaddr := ipFromAddr(addr)
			names, _ := net.LookupAddr(ipaddr.String())

			name := ""
			if len(names) >= 1 {
				name = names[0]
			}

			// Better reachability detection
			reached := false
			if method == "icmp" {
				// Reached if we get an Echo Reply (Type 0 for IPv4, 129 for IPv6)
				reached = msgType == ipv4.ICMPTypeEchoReply || msgType == ipv6.ICMPTypeEchoReply
			} else if method == "udp" {
				// Reached if we get Port Unreachable (Type 3, Code 3 for IPv4)
				reached = (msgType == ipv4.ICMPTypeDestinationUnreachable && msgCode == 3) ||
					(msgType == ipv6.ICMPTypeDestinationUnreachable && msgCode == 4) // Port Unreachable for IPv6
			}

			return Hop{
				Ttl:     ttl,
				Addr:    newHopAddress(addr),
				Name:    name,
				Reached: reached,
			}
		}
	}

	return Hop{Ttl: ttl}
}

// ipFromAddr returns the IP address from a [net.Addr].
func ipFromAddr(remoteAddr net.Addr) net.IP {
	switch addr := remoteAddr.(type) {
	case *net.UDPAddr:
		return addr.IP
	case *net.TCPAddr:
		return addr.IP
	case *net.IPAddr:
		return addr.IP
	}
	return nil
}

// Hop represents a single hop in a traceroute
type Hop struct {
	Latency    time.Duration   `json:"latency" yaml:"latency" mapstructure:"latency"`
	Addr       HopAddress      `json:"addr" yaml:"addr" mapstructure:"addr"`
	Name       string          `json:"name" yaml:"name" mapstructure:"name"`
	Ttl        int             `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
	Reached    bool            `json:"reached" yaml:"reached" mapstructure:"reached"`
	NNMiDevice *NNMIDeviceInfo `json:"nnmi_device,omitempty" yaml:"nnmi_device,omitempty" mapstructure:"nnmi_device,omitempty"`
}

type NNMIDeviceInfo struct {
	UUID          string `json:"uuid" yaml:"uuid" mapstructure:"uuid"`
	Name          string `json:"name" yaml:"name" mapstructure:"name"`
	Hostname      string `json:"hostname" yaml:"hostname" mapstructure:"hostname"`
	Status        string `json:"status" yaml:"status" mapstructure:"status"`
	DeviceType    string `json:"device_type" yaml:"device_type" mapstructure:"device_type"`
	Vendor        string `json:"vendor" yaml:"vendor" mapstructure:"vendor"`
	ManagementURL string `json:"management_url" yaml:"management_url" mapstructure:"management_url"`
	InNNMi        bool   `json:"in_nnmi" yaml:"in_nnmi" mapstructure:"in_nnmi"`
}

// HopAddress represents an IP address and port
type HopAddress struct {
	IP   string `json:"ip" yaml:"ip" mapstructure:"ip"`
	Port int    `json:"port" yaml:"port" mapstructure:"port"`
}

// String returns the string representation of the [HopAddress].
func (a HopAddress) String() string {
	if a.Port != 0 {
		return fmt.Sprintf("%s:%d", a.IP, a.Port)
	}
	return a.IP
}

// logHops logs the hops in the mapHops map
func logHops(ctx context.Context, mapHops map[int][]Hop) {
	log := logger.FromContext(ctx)

	keys := []int{}
	for k := range mapHops {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, key := range keys {
		for _, hop := range mapHops[key] {
			out := fmt.Sprintf("%d %s %s %v ", key, hop.Addr.String(), hop.Name, hop.Latency)
			if hop.Reached {
				out += "( Reached )"
			}
			log.DebugContext(ctx, out)
		}
	}
}
