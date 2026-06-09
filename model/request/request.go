package request

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type contextKey string

const (
	User           = contextKey("user")
	Username       = contextKey("username")
	Client         = contextKey("client")
	Version        = contextKey("version")
	Player         = contextKey("player")
	Transcoding    = contextKey("transcoding")
	ClientUniqueId = contextKey("clientUniqueId")
	ReverseProxyIp = contextKey("reverseProxyIp")
	InternalAuth   = contextKey("internalAuth") // Used for internal API calls, e.g., from the plugins
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
