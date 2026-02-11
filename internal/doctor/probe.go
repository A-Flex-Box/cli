package doctor

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/A-Flex-Box/cli/internal/doctor/instrument"
	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

const (
	// BannerWait is the max time to wait for a banner after connecting.
	BannerWait = 500 * time.Millisecond
)

// MeasureLatency performs a TCP handshake to target (host:port) and returns the elapsed duration.
// Does NOT use ICMP (Ping) as it requires root. Uses net.DialTimeout.
func MeasureLatency(target string) time.Duration {
	logger.Debug("MeasureLatency: dialing",
		zap.String("component", "doctor.probe"),
		zap.String("target", target))

	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, 10*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		logger.Warn("MeasureLatency: dial failed",
			zap.String("component", "doctor.probe"),
			zap.String("target", target),
			zap.Duration("elapsed", elapsed),
			zap.Error(err))
		return 0
	}
	_ = conn.Close()
	logger.Debug("MeasureLatency: success",
		zap.String("component", "doctor.probe"),
		zap.String("target", target),
		zap.Duration("elapsed", elapsed))
	return elapsed
}

// GrabBanner connects to target, waits up to BannerWait for a banner (e.g. SSH, SMTP),
// or sends "HEAD / HTTP/1.0\r\n\r\n" if HTTP is suspected. Returns the banner string.
func GrabBanner(target string) string {
	logger.Debug("GrabBanner: connecting",
		zap.String("component", "doctor.probe"),
		zap.String("target", target))

	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		logger.Warn("GrabBanner: dial failed",
			zap.String("component", "doctor.probe"),
			zap.String("target", target),
			zap.Error(err))
		return ""
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(BannerWait))
	rd := bufio.NewReader(conn)
	// Peek first bytes to detect protocol
	peek, err := rd.Peek(instrument.MaxHeaderBytes)
	if err != nil && err.Error() != "EOF" {
		// May get timeout or connection reset
		peek = []byte{}
	}
	if len(peek) > 0 {
		proto := instrument.DetectProtocol(peek)
		logger.Debug("GrabBanner: detected protocol from peek",
			zap.String("component", "doctor.probe"),
			zap.String("target", target),
			zap.String("protocol", proto))
		if proto == instrument.ProtocolHTTP {
			// Reconnect and send HEAD for HTTP
			_ = conn.Close()
			return grabHTTPBanner(target)
		}
		// Return first line(s) as banner
		lines := strings.Split(string(peek), "\n")
		banner := strings.TrimSpace(lines[0])
		if len(banner) > 200 {
			banner = banner[:200] + "..."
		}
		logger.Debug("GrabBanner: returning banner",
			zap.String("component", "doctor.probe"),
			zap.String("target", target),
			zap.String("banner", instrument.TruncateForDisplay([]byte(banner), 80)))
		return banner
	}
	// No data in time; try HTTP HEAD as fallback
	return grabHTTPBanner(target)
}

func grabHTTPBanner(target string) string {
	logger.Debug("GrabBanner: attempting HTTP HEAD",
		zap.String("component", "doctor.probe"),
		zap.String("target", target))

	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	req := "HEAD / HTTP/1.0\r\nHost: " + stripPort(target) + "\r\n\r\n"
	_, err = conn.Write([]byte(req))
	if err != nil {
		logger.Debug("GrabBanner: HTTP HEAD write failed",
			zap.String("component", "doctor.probe"),
			zap.String("target", target),
			zap.Error(err))
		return ""
	}
	_ = conn.SetReadDeadline(time.Now().Add(BannerWait))
	rd := bufio.NewReader(conn)
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		line, err := rd.ReadString('\n')
		if err != nil || line == "\r\n" || line == "\n" {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if sb.Len() > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(line)
	}
	banner := sb.String()
	if len(banner) > 200 {
		banner = banner[:200] + "..."
	}
	logger.Debug("GrabBanner: HTTP response header",
		zap.String("component", "doctor.probe"),
		zap.String("target", target),
		zap.String("banner", instrument.TruncateForDisplay([]byte(banner), 80)))
	return banner
}

func stripPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// CheckDNSResolve checks if a hostname resolves (internet connectivity hint).
func CheckDNSResolve(host string) error {
	logger.Debug("CheckDNSResolve: resolving",
		zap.String("component", "doctor.probe"),
		zap.String("host", host))

	addrs, err := net.LookupHost(host)
	if err != nil {
		logger.Warn("CheckDNSResolve: lookup failed",
			zap.String("component", "doctor.probe"),
			zap.String("host", host),
			zap.Error(err))
		return fmt.Errorf("DNS lookup %s: %w", host, err)
	}
	if len(addrs) == 0 {
		logger.Warn("CheckDNSResolve: no addresses",
			zap.String("component", "doctor.probe"),
			zap.String("host", host))
		return fmt.Errorf("DNS lookup %s: no addresses", host)
	}
	logger.Debug("CheckDNSResolve: success",
		zap.String("component", "doctor.probe"),
		zap.String("host", host),
		zap.Strings("addrs", addrs))
	return nil
}
