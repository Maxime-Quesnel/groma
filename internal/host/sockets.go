package host

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

type Socket struct {
	Addr netip.Addr
	Port uint16
	// Processes is empty when groma may not see the owner, which happens for
	// other users' sockets unless groma runs as root.
	Processes []Process
}

type Process struct {
	Name string
	PID  int
}

func (s Socket) String() string {
	return netip.AddrPortFrom(s.Addr, s.Port).String()
}

func (s Socket) Loopback() bool {
	return s.Addr.IsLoopback()
}

func (s Socket) AllInterfaces() bool {
	return s.Addr.IsUnspecified()
}

func ListeningSockets(ctx context.Context, r Runner) ([]Socket, error) {
	out, err := run(ctx, r, "ss", "-H", "-t", "-l", "-n", "-p")
	if err != nil {
		return nil, err
	}
	return parseSS(string(out))
}

var ssProcess = regexp.MustCompile(`\("([^"]*)",pid=(\d+)`)

func parseSS(out string) ([]Socket, error) {
	var sockets []Socket
	for line := range strings.Lines(out) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 5 {
			return nil, fmt.Errorf("ss: unexpected line %q", strings.TrimSpace(line))
		}
		addr, port, err := parseSSAddr(fields[3])
		if err != nil {
			return nil, fmt.Errorf("ss: %w", err)
		}
		socket := Socket{Addr: addr, Port: port}
		for _, m := range ssProcess.FindAllStringSubmatch(strings.Join(fields[5:], " "), -1) {
			pid, _ := strconv.Atoi(m[2])
			socket.Processes = append(socket.Processes, Process{Name: m[1], PID: pid})
		}
		sockets = append(sockets, socket)
	}
	return sockets, nil
}

// parseSSAddr reads ss local addresses: 0.0.0.0:22, [::]:22, :::22, *:22,
// 127.0.0.53%lo:53, [fe80::1%eth0]:123, [::ffff:127.0.0.1]:8080.
func parseSSAddr(s string) (netip.Addr, uint16, error) {
	i := strings.LastIndexByte(s, ':')
	if i < 0 {
		return netip.Addr{}, 0, fmt.Errorf("address without port %q", s)
	}
	port, err := strconv.ParseUint(s[i+1:], 10, 16)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("invalid port in %q", s)
	}
	host := strings.TrimSuffix(strings.TrimPrefix(s[:i], "["), "]")
	if zone := strings.IndexByte(host, '%'); zone >= 0 {
		host = host[:zone]
	}
	if host == "*" {
		return netip.IPv6Unspecified(), uint16(port), nil
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("invalid address in %q", s)
	}
	return addr.Unmap(), uint16(port), nil
}
