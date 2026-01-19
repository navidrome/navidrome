// Test Users plugin for Navidrome plugin system integration tests.
// This plugin tests user metadata access via the Users host service.
// Build with: tinygo build -o ../test-users.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// TestUsersInput is the input for nd_test_users callback.
type TestUsersInput struct {
	Operation string `json:"operation"` // "get_users", "get_admins"
}

// TestUsersOutput is the output from nd_test_users callback.
type TestUsersOutput struct {
	Users []host.User `json:"users,omitempty"`
	Error *string     `json:"error,omitempty"`
}

// nd_test_users is the test callback that tests the users host functions.
//
//go:wasmexport nd_test_users
func ndTestUsers() int32 {
	var input TestUsersInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestUsersOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "get_users":
		users, err := host.UsersGetUsers()
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestUsersOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestUsersOutput{Users: users})
		return 0

	case "get_admins":
		admins, err := host.UsersGetAdmins()
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestUsersOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestUsersOutput{Users: admins})
		return 0

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestUsersOutput{Error: &errStr})
		return 0
	}
}

func main() {}
