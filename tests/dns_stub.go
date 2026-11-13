package tests

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/netip"

	"golang.org/x/net/dns/dnsmessage"
)

// StubResolver returns a net.Resolver that answers A queries from records (absolute name to IPv4,
// e.g. "localhost." to "127.0.0.1") instead of the system DNS. Other names do not resolve.
func StubResolver(records map[string]string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			client, server := net.Pipe()
			go serveStubDNS(server, records)
			return client, nil
		},
	}
}

// net.Pipe is not a PacketConn, so the resolver sends one query framed with a 2-byte length prefix.
func serveStubDNS(conn net.Conn, records map[string]string) {
	defer conn.Close()
	var size [2]byte
	if _, err := io.ReadFull(conn, size[:]); err != nil {
		return
	}
	buf := make([]byte, binary.BigEndian.Uint16(size[:]))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	var msg dnsmessage.Message
	if err := msg.Unpack(buf); err != nil || len(msg.Questions) != 1 {
		return
	}
	msg.Response = true
	q := msg.Questions[0]
	if ip, err := netip.ParseAddr(records[q.Name.String()]); err == nil && q.Type == dnsmessage.TypeA {
		msg.Answers = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{Name: q.Name, Type: q.Type, Class: q.Class},
			Body:   &dnsmessage.AResource{A: ip.As4()},
		}}
	}
	resp, err := msg.Pack()
	if err != nil {
		return
	}
	_, _ = conn.Write(append(binary.BigEndian.AppendUint16(nil, uint16(len(resp))), resp...))
}
