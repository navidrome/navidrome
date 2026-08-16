package playback

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/core/playback/cast"
	"github.com/navidrome/navidrome/core/playback/mpv"
	"github.com/navidrome/navidrome/model"
)

type TrackFactory func(
	ctx context.Context,
	playbackDone chan bool,
	deviceName string,
	mf model.MediaFile,
) (Track, error)

var castTrackFactory = func(ctx context.Context, playbackDone chan bool, target cast.Target, mf model.MediaFile) (Track, error) {
	return cast.NewTrack(ctx, playbackDone, target, mf)
}

func defaultTrackFactory(
	ctx context.Context,
	playbackDone chan bool,
	deviceName string,
	mf model.MediaFile,
) (Track, error) {
	return mpv.NewTrack(ctx, playbackDone, deviceName, mf)
}

func resolveTrackFactory(name, rawTarget string, fallback TrackFactory) (TrackFactory, error) {
	if !strings.HasPrefix(rawTarget, "cast:") {
		return fallback, nil
	}

	target, err := parseCastTarget(name, rawTarget)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, playbackDone chan bool, _ string, mf model.MediaFile) (Track, error) {
		return castTrackFactory(ctx, playbackDone, target, mf)
	}, nil
}

func parseCastTarget(name, raw string) (cast.Target, error) {
	const prefix = "cast:"
	if !strings.HasPrefix(raw, prefix) {
		return cast.Target{}, fmt.Errorf("invalid cast target %q", raw)
	}

	addr := strings.TrimPrefix(raw, prefix)
	if addr == "" {
		return cast.Target{}, fmt.Errorf("invalid cast target %q: host is required", raw)
	}
	if strings.HasPrefix(addr, "//") {
		return cast.Target{}, fmt.Errorf("invalid cast target %q: use cast:host[:port]", raw)
	}

	host := addr
	port := 8009
	if strings.HasPrefix(addr, "[") {
		end := strings.Index(addr, "]")
		if end < 0 {
			return cast.Target{}, fmt.Errorf("invalid cast target %q: malformed IPv6 address", raw)
		}
		host = addr[1:end]
		if hasASCIISpace(host) {
			return cast.Target{}, fmt.Errorf("invalid cast target %q: host must not contain whitespace", raw)
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.To4() != nil {
			return cast.Target{}, fmt.Errorf("invalid cast target %q: bracketed host must be IPv6", raw)
		}
		rest := addr[end+1:]
		switch {
		case rest == "":
		case strings.HasPrefix(rest, ":"):
			parsedPort, err := parseCastPort(rest[1:], raw)
			if err != nil {
				return cast.Target{}, err
			}
			port = parsedPort
		default:
			return cast.Target{}, fmt.Errorf("invalid cast target %q: malformed IPv6 address", raw)
		}
	} else {
		switch strings.Count(addr, ":") {
		case 0:
		case 1:
			var portText string
			host, portText, _ = strings.Cut(addr, ":")
			parsedPort, err := parseCastPort(portText, raw)
			if err != nil {
				return cast.Target{}, err
			}
			port = parsedPort
		default:
			return cast.Target{}, fmt.Errorf("invalid cast target %q: bracket IPv6 addresses", raw)
		}
	}

	if host == "" {
		return cast.Target{}, fmt.Errorf("invalid cast target %q: host is required", raw)
	}
	if hasASCIISpace(host) {
		return cast.Target{}, fmt.Errorf("invalid cast target %q: host must not contain whitespace", raw)
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return cast.Target{}, fmt.Errorf("invalid cast target %q: malformed IPv6 address", raw)
	}
	return cast.Target{Name: name, Host: host, Port: port}, nil
}

func parseCastPort(portText, raw string) (int, error) {
	if portText == "" {
		return 0, fmt.Errorf("invalid cast target %q: port is required after ':'", raw)
	}
	if !isASCIIDigits(portText) {
		return 0, fmt.Errorf("invalid cast target %q: invalid port %q", raw, portText)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, fmt.Errorf("invalid cast target %q: invalid port %q", raw, portText)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid cast target %q: port %d out of range", raw, port)
	}
	return port, nil
}

func hasASCIISpace(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			return true
		}
	}
	return false
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
