package apiv1

import (
	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
)

func toScopes(in []string) []Scope {
	return slice.Map(in, func(s string) Scope { return Scope(s) })
}

// fromScopeRequests keeps nil (all scopes) apart from an empty list (no scopes).
func fromScopeRequests(in *[]ScopeRequest) []string {
	if in == nil {
		return nil
	}
	return append([]string{}, *in...)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toGrant(g model.Grant, currentID string) Grant {
	return Grant{
		Id:            g.ID,
		Name:          g.Name,
		Client:        g.Client,
		ClientVersion: nullable(g.ClientVersion),
		Scopes:        toScopes(g.Scopes),
		Provider:      g.Provider,
		CreatedAt:     g.CreatedAt,
		LastUsedAt:    g.LastUsedAt,
		LastUsedIp:    nullable(g.LastUsedIP),
		Current:       g.ID == currentID,
	}
}

func toGrantCreated(i *apiauth.Issued) GrantCreated {
	return GrantCreated{
		Secret: i.Secret,
		Grant:  toGrant(i.Grant, i.Grant.ID),
		User: AuthUser{
			Id:                 i.User.ID,
			UserName:           i.User.UserName,
			Name:               i.User.Name,
			IsAdmin:            i.User.IsAdmin,
			PasswordChangeable: apiauth.PasswordChangeable(i.User),
		},
	}
}

func clientMeta(c CredentialsRequest) apiauth.ClientMeta {
	return apiauth.ClientMeta{Client: c.Client, Name: gg.V(c.Name), ClientVersion: gg.V(c.ClientVersion)}
}

// principal fails closed if the gate did not attach a principal to the context.
func principal(p *apiauth.Principal, ok bool) (*apiauth.Principal, error) {
	if !ok || p == nil {
		return nil, model.ErrInvalidAuth
	}
	return p, nil
}
