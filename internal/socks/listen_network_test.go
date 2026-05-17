package socks

import (
	"net"
	"testing"
	"time"
)

func TestListenNetwork(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{name: "ipv4 wildcard", addr: "0.0.0.0:1080", want: "tcp4"},
		{name: "ipv4 loopback", addr: "127.0.0.1:1080", want: "tcp4"},
		{name: "ipv4 explicit", addr: "192.168.1.10:1080", want: "tcp4"},
		{name: "ipv6 wildcard", addr: "[::]:1080", want: "tcp6"},
		{name: "ipv6 loopback", addr: "[::1]:1080", want: "tcp6"},
		{name: "hostname", addr: "localhost:1080", want: "tcp"},
		{name: "bare port", addr: ":1080", want: "tcp"},
		{name: "malformed", addr: "no-port-at-all", want: "tcp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := listenNetwork(tc.addr)
			if got != tc.want {
				t.Fatalf("listenNetwork(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}

// TestListen_IPv4AddrBindsAF_INET verifies that the actual net.Listen call
// using the network chosen by listenNetwork produces a listener whose
// resolved address is an IPv4 address. Before the fix, "tcp" + "0.0.0.0"
// caused Go to bind AF_INET6 with V4MAPPED, which on Linux hosts with
// net.ipv6.bindv6only=1 refuses IPv4 connections (#94, #111).
func TestListen_IPv4AddrBindsAF_INET(t *testing.T) {
	addr := "127.0.0.1:0" // ephemeral port
	ln, err := net.Listen(listenNetwork(addr), addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("addr is not *net.TCPAddr: %T", ln.Addr())
	}
	if tcp.IP.To4() == nil {
		t.Fatalf("expected AF_INET binding, got %v (To4 was nil)", tcp.IP)
	}
	// Sanity: an IPv4 dialer must be able to reach this listener.
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		c.Close()
	}()
	c, err := net.DialTimeout("tcp4", tcp.String(), 2*time.Second)
	if err != nil {
		t.Fatalf("ipv4 dial: %v", err)
	}
	c.Close()
}
