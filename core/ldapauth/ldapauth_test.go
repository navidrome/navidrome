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
		got := loginUserFilter(Source{UserNameAttribute: "uid"}, "directory-user")
		want := "(uid=directory-user)"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("replaces every username placeholder", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(|(uid=%s)(mail=%s))"}, "directory-user")
		want := "(|(uid=directory-user)(mail=directory-user))"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("supports one-placeholder username filters", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(uid=%s)"}, "directory-user")
		want := "(uid=directory-user)"
		if got != want {
			t.Fatalf("loginUserFilter() = %q, want %q", got, want)
		}
	})

	t.Run("adds username assertion to discovery filters", func(t *testing.T) {
		t.Parallel()
		got := loginUserFilter(Source{UserNameAttribute: "uid", UserFilter: "(objectClass=person)"}, "directory-user")
		want := "(&(objectClass=person)(uid=directory-user))"
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
	repo.Data["directory-user"] = &model.User{ID: "1", UserName: "directory-user", Password: "generated", AuthSource: "ldap", AuthSourceID: "freeipa"}
	ds := &tests.MockDataStore{MockedUser: repo}

	user, found, err := authInternal(t.Context(), ds, "directory-user", "ldap-password", true)
	if err != nil {
		t.Fatalf("authInternal() unexpected error: %v", err)
	}
	if found || user != nil {
		t.Fatalf("authInternal() found=%v user=%#v, want external user to be skipped for fallback", found, user)
	}

	user, found, err = authInternal(t.Context(), ds, "directory-user", "ldap-password", false)
	if err != nil {
		t.Fatalf("authInternal() unexpected error: %v", err)
	}
	if !found || user != nil {
		t.Fatalf("authInternal() found=%v user=%#v, want explicit internal auth to stop", found, user)
	}
}
