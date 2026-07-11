package ldapauth

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
)

type Config struct {
	Sources []Source `json:"sources"`
}
type Source struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Enabled              bool       `json:"enabled"`
	URL                  string     `json:"url"`
	StartTLS             bool       `json:"startTLS"`
	InsecureSkipVerify   bool       `json:"insecureSkipVerify"`
	BindDN               string     `json:"bindDN"`
	BindPassword         string     `json:"bindPassword,omitempty"`
	UserBaseDN           string     `json:"userBaseDN"`
	UserFilter           string     `json:"userFilter"`
	UserNameAttribute    string     `json:"userNameAttribute"`
	DisplayNameAttribute string     `json:"displayNameAttribute"`
	EmailAttribute       string     `json:"emailAttribute"`
	GroupBaseDN          string     `json:"groupBaseDN"`
	GroupFilter          string     `json:"groupFilter"`
	GroupNameAttribute   string     `json:"groupNameAttribute"`
	GroupMemberAttribute string     `json:"groupMemberAttribute"`
	RequiredGroupDNs     []string   `json:"requiredGroupDNs"`
	AdminGroupDNs        []string   `json:"adminGroupDNs"`
	DirectBindDNTemplate string     `json:"directBindDNTemplate"`
	LastSyncAt           *time.Time `json:"lastSyncAt,omitempty"`
	Cache                Cache      `json:"cache,omitempty"`
}
type Cache struct {
	Users  []DiscoveredUser  `json:"users,omitempty"`
	Groups []DiscoveredGroup `json:"groups,omitempty"`
}
type DiscoveredUser struct {
	DN       string   `json:"dn"`
	UserName string   `json:"userName"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups,omitempty"`
}
type DiscoveredGroup struct {
	DN      string   `json:"dn"`
	Name    string   `json:"name"`
	Members []string `json:"members,omitempty"`
}
type AuthSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

var ErrUserNotFound = errors.New("ldap user not found")
var ErrBadPassword = errors.New("ldap invalid password")

type Store struct {
	path string
	key  []byte
}

func NewStore() *Store {
	return &Store{path: filepath.Join(conf.Server.DataFolder.String(), "ldap.json"), key: keyTo32Bytes(cmpKey())}
}
func cmpKey() string {
	if conf.Server.PasswordEncryptionKey != "" {
		return conf.Server.PasswordEncryptionKey
	}
	return consts.DefaultEncryptionKey
}
func keyTo32Bytes(input string) []byte { s := sha256.Sum256([]byte(input)); return s[:] }
func (s *Store) Load(ctx context.Context) (Config, error) {
	var c Config
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if len(b) == 0 {
		return c, nil
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}
	for i := range c.Sources {
		if c.Sources[i].BindPassword != "" {
			if p, e := utils.Decrypt(ctx, s.key, c.Sources[i].BindPassword); e == nil {
				c.Sources[i].BindPassword = p
			}
		}
	}
	return c, nil
}
func (s *Store) Save(ctx context.Context, c Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	var out Config
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	for i := range out.Sources {
		if out.Sources[i].ID == "" {
			out.Sources[i].ID = id.NewRandom()
		}
		if out.Sources[i].UserNameAttribute == "" {
			out.Sources[i].UserNameAttribute = "uid"
		}
		if out.Sources[i].DisplayNameAttribute == "" {
			out.Sources[i].DisplayNameAttribute = "cn"
		}
		if out.Sources[i].EmailAttribute == "" {
			out.Sources[i].EmailAttribute = "mail"
		}
		if out.Sources[i].GroupNameAttribute == "" {
			out.Sources[i].GroupNameAttribute = "cn"
		}
		if out.Sources[i].GroupMemberAttribute == "" {
			out.Sources[i].GroupMemberAttribute = "member"
		}
		if out.Sources[i].BindPassword != "" {
			enc, err := utils.Encrypt(ctx, s.key, out.Sources[i].BindPassword)
			if err != nil {
				return err
			}
			out.Sources[i].BindPassword = enc
		}
	}
	b, err = json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}
func Sources(ctx context.Context) []AuthSource {
	cfg, _ := NewStore().Load(ctx)
	res := []AuthSource{{ID: "internal", Name: "Internal", Type: "internal"}}
	for _, src := range cfg.Sources {
		if src.Enabled {
			res = append(res, AuthSource{ID: src.ID, Name: src.Name, Type: "ldap"})
		}
	}
	return res
}

func Authenticate(ctx context.Context, ds model.DataStore, sourceID, username, password string) (*model.User, error) {
	if sourceID == "" || sourceID == "internal" {
		u, found, err := authInternal(ctx, ds, username, password)
		if err != nil || found {
			return u, err
		}
		if sourceID == "internal" {
			return nil, nil
		}
	}
	cfg, _ := NewStore().Load(ctx)
	for _, src := range cfg.Sources {
		if !src.Enabled {
			continue
		}
		if sourceID != "" && sourceID != "internal" && sourceID != src.ID && sourceID != src.Name {
			continue
		}
		u, err := authLDAP(ctx, ds, src, username, password)
		if errors.Is(err, ErrUserNotFound) {
			continue
		}
		if err != nil {
			return nil, nil
		}
		return u, nil
	}
	return nil, nil
}
func authInternal(ctx context.Context, ds model.DataStore, username, password string) (*model.User, bool, error) {
	u, err := ds.User(ctx).FindByUsernameWithPassword(username)
	if errors.Is(err, model.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if u.Password != password {
		return nil, true, nil
	}
	_ = ds.User(ctx).UpdateLastLoginAt(u.ID)
	return u, true, nil
}

func authLDAP(ctx context.Context, ds model.DataStore, src Source, username, password string) (*model.User, error) {
	if password == "" {
		return nil, ErrBadPassword
	}
	du, err := lookupAndBind(src, username, password)
	if err != nil {
		return nil, err
	}
	if !allowed(src, du) {
		return nil, ErrBadPassword
	}
	repo := ds.User(ctx)
	u, err := repo.FindByUsername(du.UserName)
	if errors.Is(err, model.ErrNotFound) {
		u = &model.User{ID: id.NewRandom(), UserName: du.UserName, NewPassword: consts.PasswordAutogenPrefix + id.NewRandom()}
	} else if err != nil {
		return nil, err
	}
	u.Name = first(du.Name, du.UserName)
	u.Email = du.Email
	u.IsAdmin = memberAny(du.Groups, src.AdminGroupDNs)
	if err := repo.Put(u); err != nil {
		return nil, err
	}
	_ = repo.UpdateLastLoginAt(u.ID)
	return repo.FindByUsernameWithPassword(u.UserName)
}
func lookupAndBind(src Source, username, password string) (DiscoveredUser, error) {
	l, err := ldap.DialURL(src.URL)
	if err != nil {
		return DiscoveredUser{}, ErrUserNotFound
	}
	defer l.Close()
	if src.StartTLS {
		_ = l.StartTLS(&tls.Config{InsecureSkipVerify: src.InsecureSkipVerify}) //nolint:gosec
	}
	if src.DirectBindDNTemplate != "" {
		dn := fmt.Sprintf(src.DirectBindDNTemplate, ldap.EscapeFilter(username))
		if err = l.Bind(dn, password); err != nil {
			return DiscoveredUser{}, ErrBadPassword
		}
		return DiscoveredUser{DN: dn, UserName: username}, nil
	}
	if src.BindDN != "" {
		if err = l.Bind(src.BindDN, src.BindPassword); err != nil {
			return DiscoveredUser{}, err
		}
	}
	filt := src.UserFilter
	if filt == "" {
		filt = "(%s=%s)"
	}
	filt = fmt.Sprintf(filt, src.UserNameAttribute, ldap.EscapeFilter(username))
	req := ldap.NewSearchRequest(src.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 30, false, filt, []string{src.UserNameAttribute, src.DisplayNameAttribute, src.EmailAttribute, "memberOf"}, nil)
	res, err := l.Search(req)
	if err != nil || len(res.Entries) == 0 {
		return DiscoveredUser{}, ErrUserNotFound
	}
	e := res.Entries[0]
	if err = l.Bind(e.DN, password); err != nil {
		return DiscoveredUser{}, ErrBadPassword
	}
	du := DiscoveredUser{DN: e.DN, UserName: first(e.GetAttributeValue(src.UserNameAttribute), username), Name: e.GetAttributeValue(src.DisplayNameAttribute), Email: e.GetAttributeValue(src.EmailAttribute), Groups: e.GetAttributeValues("memberOf")}
	du.Groups = append(du.Groups, groupsForUser(src, e.DN)...)
	return du, nil
}
func groupsForUser(src Source, userDN string) []string {
	if src.GroupBaseDN == "" {
		return nil
	}
	l, err := ldap.DialURL(src.URL)
	if err != nil {
		return nil
	}
	defer l.Close()
	if src.StartTLS {
		_ = l.StartTLS(&tls.Config{InsecureSkipVerify: src.InsecureSkipVerify}) //nolint:gosec
	}
	if src.BindDN != "" {
		_ = l.Bind(src.BindDN, src.BindPassword)
	}
	gf := src.GroupFilter
	if gf == "" {
		gf = "(|(objectClass=groupOfNames)(objectClass=groupOfUniqueNames)(objectClass=group))"
	}
	req := ldap.NewSearchRequest(src.GroupBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 30, false, gf, []string{src.GroupMemberAttribute}, nil)
	res, err := l.Search(req)
	if err != nil {
		return nil
	}
	var out []string
	for _, g := range res.Entries {
		if slices.Contains(g.GetAttributeValues(src.GroupMemberAttribute), userDN) {
			out = append(out, g.DN)
		}
	}
	return out
}
func allowed(src Source, u DiscoveredUser) bool {
	return len(src.RequiredGroupDNs) == 0 || memberAny(u.Groups, src.RequiredGroupDNs) || memberAny(u.Groups, src.AdminGroupDNs)
}
func memberAny(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}
func first(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func TestAndCache(ctx context.Context, src Source) (Source, error) {
	l, err := ldap.DialURL(src.URL)
	if err != nil {
		return src, err
	}
	defer l.Close()
	if src.StartTLS {
		_ = l.StartTLS(&tls.Config{InsecureSkipVerify: src.InsecureSkipVerify}) //nolint:gosec
	}
	if src.BindDN != "" {
		if err = l.Bind(src.BindDN, src.BindPassword); err != nil {
			return src, err
		}
	}
	attrs := []string{src.UserNameAttribute, src.DisplayNameAttribute, src.EmailAttribute, "memberOf"}
	uf := src.UserFilter
	if uf == "" {
		uf = "(objectClass=person)"
	}
	ur := ldap.NewSearchRequest(src.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 60, false, uf, attrs, nil)
	users, _ := l.Search(ur)
	grps := map[string][]string{}
	for _, e := range users.Entries {
		src.Cache.Users = append(src.Cache.Users, DiscoveredUser{DN: e.DN, UserName: e.GetAttributeValue(src.UserNameAttribute), Name: e.GetAttributeValue(src.DisplayNameAttribute), Email: e.GetAttributeValue(src.EmailAttribute), Groups: e.GetAttributeValues("memberOf")})
	}
	gf := src.GroupFilter
	if gf == "" {
		gf = "(|(objectClass=groupOfNames)(objectClass=groupOfUniqueNames)(objectClass=group))"
	}
	if src.GroupBaseDN != "" {
		gr := ldap.NewSearchRequest(src.GroupBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 60, false, gf, []string{src.GroupNameAttribute, src.GroupMemberAttribute}, nil)
		gres, _ := l.Search(gr)
		for _, e := range gres.Entries {
			m := e.GetAttributeValues(src.GroupMemberAttribute)
			grps[e.DN] = m
			src.Cache.Groups = append(src.Cache.Groups, DiscoveredGroup{DN: e.DN, Name: e.GetAttributeValue(src.GroupNameAttribute), Members: m})
		}
	}
	for i, u := range src.Cache.Users {
		for g, m := range grps {
			if slices.Contains(m, u.DN) {
				src.Cache.Users[i].Groups = append(src.Cache.Users[i].Groups, g)
			}
		}
	}
	now := time.Now()
	src.LastSyncAt = &now
	log.Info(ctx, "LDAP test/cache completed", "source", src.Name, "users", len(src.Cache.Users), "groups", len(src.Cache.Groups))
	return src, nil
}
