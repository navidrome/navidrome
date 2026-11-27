package request

import (
	"context"
	"sync/atomic"

	"github.com/navidrome/navidrome/model"
)

type contextKey string

const (
	User             = contextKey("user")
	Username         = contextKey("username")
	Client           = contextKey("client")
	Version          = contextKey("version")
	Player           = contextKey("player")
	Transcoding      = contextKey("transcoding")
	ClientUniqueId   = contextKey("clientUniqueId")
	ReverseProxyIp   = contextKey("reverseProxyIp")
	InternalAuth     = contextKey("internalAuth") // Used for internal API calls, e.g., from the plugins
	TokenEpochHolder = contextKey("tokenEpochHolder")
	ServerAddress    = contextKey("serverAddress")
)

var allKeys = []contextKey{
	User,
	Username,
	Client,
	Version,
	Player,
	Transcoding,
	ClientUniqueId,
	ReverseProxyIp,
	InternalAuth,
	ServerAddress,
}

func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, User, u)
}

func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, Username, username)
}

func WithClient(ctx context.Context, client string) context.Context {
	return context.WithValue(ctx, Client, client)
}

func WithVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, Version, version)
}

func WithPlayer(ctx context.Context, player model.Player) context.Context {
	return context.WithValue(ctx, Player, player)
}

func WithTranscoding(ctx context.Context, t model.Transcoding) context.Context {
	return context.WithValue(ctx, Transcoding, t)
}

func WithClientUniqueId(ctx context.Context, clientUniqueId string) context.Context {
	return context.WithValue(ctx, ClientUniqueId, clientUniqueId)
}

func WithReverseProxyIp(ctx context.Context, reverseProxyIp string) context.Context {
	return context.WithValue(ctx, ReverseProxyIp, reverseProxyIp)
}

func WithInternalAuth(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, InternalAuth, username)
}

// serverAddress is the public scheme and host the client used to reach this server,
// so code running without an http.Request can still build absolute URLs.
type serverAddress struct {
	scheme string
	host   string
}

func WithServerAddress(ctx context.Context, scheme, host string) context.Context {
	return context.WithValue(ctx, ServerAddress, serverAddress{scheme: scheme, host: host})
}

func ServerAddressFrom(ctx context.Context) (scheme, host string, ok bool) {
	a, ok := ctx.Value(ServerAddress).(serverAddress)
	if !ok || a.host == "" {
		return "", "", false
	}
	return a.scheme, a.host, true
}

func UserFrom(ctx context.Context) (model.User, bool) {
	v, ok := ctx.Value(User).(model.User)
	return v, ok
}

func UsernameFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(Username).(string)
	return v, ok
}

func ClientFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(Client).(string)
	return v, ok
}

func VersionFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(Version).(string)
	return v, ok
}

func PlayerFrom(ctx context.Context) (model.Player, bool) {
	v, ok := ctx.Value(Player).(model.Player)
	return v, ok
}

func TranscodingFrom(ctx context.Context) (model.Transcoding, bool) {
	v, ok := ctx.Value(Transcoding).(model.Transcoding)
	return v, ok
}

func ClientUniqueIdFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ClientUniqueId).(string)
	return v, ok
}

func ReverseProxyIpFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ReverseProxyIp).(string)
	return v, ok
}

func InternalAuthFrom(ctx context.Context) (string, bool) {
	if v := ctx.Value(InternalAuth); v != nil {
		if username, ok := v.(string); ok {
			return username, true
		}
	}
	return "", false
}

func AddValues(ctx, requestCtx context.Context) context.Context {
	for _, key := range allKeys {
		if v := requestCtx.Value(key); v != nil {
			ctx = context.WithValue(ctx, key, v)
		}
	}
	return ctx
}

type tokenEpochHolder struct {
	value atomic.Int64
}

// WithTokenEpochHolder installs a slot a handler can use to report a bumped token epoch
// back to middleware that has already returned from the handler's perspective.
func WithTokenEpochHolder(ctx context.Context) context.Context {
	h := &tokenEpochHolder{}
	h.value.Store(-1)
	return context.WithValue(ctx, TokenEpochHolder, h)
}

func SetTokenEpoch(ctx context.Context, epoch int) {
	if h, ok := ctx.Value(TokenEpochHolder).(*tokenEpochHolder); ok {
		h.value.Store(int64(epoch))
	}
}

func TokenEpochFrom(ctx context.Context) (int, bool) {
	h, ok := ctx.Value(TokenEpochHolder).(*tokenEpochHolder)
	if !ok {
		return 0, false
	}
	if v := h.value.Load(); v >= 0 {
		return int(v), true
	}
	return 0, false
}
