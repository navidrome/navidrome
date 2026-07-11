package ldapauth

import (
	"testing"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
)

func TestLoginUserFilter(t *testing.T) {
	t.Parallel()

	t.Run("defaults to username attribute equality", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid"}, "firehawk")
		want := "(uid=firehawk)"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("preserves explicit placeholder filters", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(&(objectClass=person)(%s=%s))"}, "firehawk")
		want := "(&(objectClass=person)(uid=firehawk))"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("supports one-placeholder username filters", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(uid=%s)"}, "firehawk")
		want := "(uid=firehawk)"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("adds username assertion to discovery filters", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(objectClass=person)"}, "firehawk")
		want := "(&(objectClass=person)(uid=firehawk))"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})
}

func TestDedupeStrings(t *testing.T) {
	t.Parallel()
	got := dedupeStrings([]string{"cn=users", "CN=users", "", "cn=admins"})
	want := []string{"cn=users", "cn=admins"}
	if len(got) != len(want) {
		t.Fatalf("dedupeStrings() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupeStrings() = %#v, want %#v", got, want)
		}
	}
}

func TestAuthInternalSkipsExternalUsersForFallback(t *testing.T) {
	t.Parallel()

	repo := tests.CreateMockUserRepo()
	repo.Data["firehawk"] = &model.User{ID: "1", UserName: "firehawk", Password: "generated", AuthSource: "ldap", AuthSourceID: "freeipa"}
	ds := &tests.MockDataStore{MockedUser: repo}

	user, found, err := authInternal(t.Context(), ds, "firehawk", "ldap-password", true)
	if err != nil {
		t.Fatalf("authInternal() unexpected error: %v", err)
	}
	if found || user != nil {
		t.Fatalf("authInternal() found=%v user=%#v, want external user to be skipped for fallback", found, user)
	}

	user, found, err = authInternal(t.Context(), ds, "firehawk", "ldap-password", false)
	if err != nil {
		t.Fatalf("authInternal() unexpected error: %v", err)
	}
	if !found || user != nil {
		t.Fatalf("authInternal() found=%v user=%#v, want explicit internal auth to stop", found, user)
	}
}
