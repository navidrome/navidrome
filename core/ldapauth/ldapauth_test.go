package ldapauth

import "testing"

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
