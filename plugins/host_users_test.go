//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UsersService", Ordered, func() {
	var (
		ctx     context.Context
		ds      model.DataStore
		service host.UsersService
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{}
	})

	Describe("GetUsers", func() {
		var mockUserRepo *tests.MockedUserRepo

		BeforeEach(func() {
			mockUserRepo = ds.User(ctx).(*tests.MockedUserRepo)
			// Add test users
			_ = mockUserRepo.Put(&model.User{
				ID:       "user1",
				UserName: "alice",
				Name:     "Alice Admin",
				IsAdmin:  true,
			})
			_ = mockUserRepo.Put(&model.User{
				ID:       "user2",
				UserName: "bob",
				Name:     "Bob User",
				IsAdmin:  false,
			})
			_ = mockUserRepo.Put(&model.User{
				ID:       "user3",
				UserName: "charlie",
				Name:     "Charlie User",
				IsAdmin:  false,
			})
		})

		Context("with allUsers=true", func() {
			BeforeEach(func() {
				service = newUsersService(ds, nil, true)
			})

			It("should return all users", func() {
				users, err := service.GetUsers(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(users).To(HaveLen(3))

				// Verify that the correct fields are returned
				userNames := make([]string, len(users))
				for i, u := range users {
					userNames[i] = u.UserName
				}
				Expect(userNames).To(ContainElements("alice", "bob", "charlie"))
			})

			It("should return correct user properties", func() {
				users, err := service.GetUsers(ctx)
				Expect(err).ToNot(HaveOccurred())

				// Find alice
				var alice *host.User
				for i := range users {
					if users[i].UserName == "alice" {
						alice = &users[i]
						break
					}
				}

				Expect(alice).ToNot(BeNil())
				Expect(alice.UserName).To(Equal("alice"))
				Expect(alice.Name).To(Equal("Alice Admin"))
				Expect(alice.IsAdmin).To(BeTrue())
			})
		})

		Context("with specific allowed users", func() {
			BeforeEach(func() {
				// Only allow access to user1 and user3
				service = newUsersService(ds, []string{"user1", "user3"}, false)
			})

			It("should return only allowed users", func() {
				users, err := service.GetUsers(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(users).To(HaveLen(2))

				userNames := make([]string, len(users))
				for i, u := range users {
					userNames[i] = u.UserName
				}
				Expect(userNames).To(ContainElements("alice", "charlie"))
				Expect(userNames).ToNot(ContainElement("bob"))
			})
		})

		Context("with empty allowed users and allUsers=false", func() {
			BeforeEach(func() {
				service = newUsersService(ds, []string{}, false)
			})

			It("should return no users", func() {
				users, err := service.GetUsers(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(users).To(BeEmpty())
			})
		})

		Context("when datastore returns error", func() {
			BeforeEach(func() {
				mockUserRepo.Error = model.ErrNotFound
				service = newUsersService(ds, nil, true)
			})

			It("should propagate the error", func() {
				_, err := service.GetUsers(ctx)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetAdmins", func() {
		var mockUserRepo *tests.MockedUserRepo

		BeforeEach(func() {
			mockUserRepo = ds.User(ctx).(*tests.MockedUserRepo)
			// Add test users - alice is admin, bob and charlie are not
			_ = mockUserRepo.Put(&model.User{
				ID:       "user1",
				UserName: "alice",
				Name:     "Alice Admin",
				IsAdmin:  true,
			})
			_ = mockUserRepo.Put(&model.User{
				ID:       "user2",
				UserName: "bob",
				Name:     "Bob User",
				IsAdmin:  false,
			})
			_ = mockUserRepo.Put(&model.User{
				ID:       "user3",
				UserName: "charlie",
				Name:     "Charlie User",
				IsAdmin:  false,
			})
		})

		Context("with allUsers=true", func() {
			BeforeEach(func() {
				service = newUsersService(ds, nil, true)
			})

			It("should return only admin users", func() {
				admins, err := service.GetAdmins(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(admins).To(HaveLen(1))
				Expect(admins[0].UserName).To(Equal("alice"))
				Expect(admins[0].IsAdmin).To(BeTrue())
			})
		})

		Context("with specific allowed users including admin", func() {
			BeforeEach(func() {
				// Allow access to user1 (admin) and user2 (non-admin)
				service = newUsersService(ds, []string{"user1", "user2"}, false)
			})

			It("should return only admin users from allowed list", func() {
				admins, err := service.GetAdmins(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(admins).To(HaveLen(1))
				Expect(admins[0].UserName).To(Equal("alice"))
			})
		})

		Context("with specific allowed users excluding admin", func() {
			BeforeEach(func() {
				// Only allow access to non-admin users
				service = newUsersService(ds, []string{"user2", "user3"}, false)
			})

			It("should return empty when no admins in allowed list", func() {
				admins, err := service.GetAdmins(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(admins).To(BeEmpty())
			})
		})

		Context("when datastore returns error", func() {
			BeforeEach(func() {
				mockUserRepo.Error = model.ErrNotFound
				service = newUsersService(ds, nil, true)
			})

			It("should propagate the error", func() {
				_, err := service.GetAdmins(ctx)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("UsersService Integration", Ordered, func() {
	var manager *Manager

	BeforeAll(func() {
		var cleanup func()
		manager, cleanup = setupUsersIntegrationManager(true, "")
		DeferCleanup(cleanup)
	})

	Describe("Plugin Loading", func() {
		It("should load plugin with users permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-users"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Users).ToNot(BeNil())
		})
	})

	Describe("Users Operations via Plugin", func() {
		It("should get all users when allUsers is true", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_users"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Users).To(HaveLen(3))

			// Verify user names
			userNames := make([]string, len(output.Users))
			for i, u := range output.Users {
				userNames[i] = u.UserName
			}
			Expect(userNames).To(ContainElements("alice", "bob", "charlie"))
		})

		It("should return correct user properties", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_users"})
			Expect(err).ToNot(HaveOccurred())

			// Find alice
			var alice *testUser
			for i := range output.Users {
				if output.Users[i].UserName == "alice" {
					alice = &output.Users[i]
					break
				}
			}

			Expect(alice).ToNot(BeNil())
			Expect(alice.UserName).To(Equal("alice"))
			Expect(alice.Name).To(Equal("Alice Admin"))
			Expect(alice.IsAdmin).To(BeTrue())
		})

		It("should return non-admin user correctly", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_users"})
			Expect(err).ToNot(HaveOccurred())

			// Find bob
			var bob *testUser
			for i := range output.Users {
				if output.Users[i].UserName == "bob" {
					bob = &output.Users[i]
					break
				}
			}

			Expect(bob).ToNot(BeNil())
			Expect(bob.UserName).To(Equal("bob"))
			Expect(bob.Name).To(Equal("Bob User"))
			Expect(bob.IsAdmin).To(BeFalse())
		})
	})

	Describe("GetAdmins Operations via Plugin", func() {
		It("should get only admin users when allUsers is true", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_admins"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Users).To(HaveLen(1))
			Expect(output.Users[0].UserName).To(Equal("alice"))
			Expect(output.Users[0].IsAdmin).To(BeTrue())
		})
	})
})

var _ = Describe("UsersService Integration with Specific Users", Ordered, func() {
	var manager *Manager

	BeforeAll(func() {
		var cleanup func()
		manager, cleanup = setupUsersIntegrationManager(false, `["user1", "user3"]`)
		DeferCleanup(cleanup)
	})

	Describe("Users Operations with Specific Allowed Users", func() {
		It("should only return allowed users", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_users"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Users).To(HaveLen(2))

			// Verify only alice and charlie are returned, not bob
			userNames := make([]string, len(output.Users))
			for i, u := range output.Users {
				userNames[i] = u.UserName
			}
			Expect(userNames).To(ContainElements("alice", "charlie"))
			Expect(userNames).ToNot(ContainElement("bob"))
		})

		It("should only return admin users from allowed list via GetAdmins", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_admins"})
			Expect(err).ToNot(HaveOccurred())
			// Only alice (user1) is admin, charlie (user3) is not
			Expect(output.Users).To(HaveLen(1))
			Expect(output.Users[0].UserName).To(Equal("alice"))
			Expect(output.Users[0].IsAdmin).To(BeTrue())
		})
	})
})

var _ = Describe("UsersService Integration GetAdmins with No Admins", Ordered, func() {
	var manager *Manager

	BeforeAll(func() {
		var cleanup func()
		// Only allow user2 (bob) and user3 (charlie), both non-admins
		manager, cleanup = setupUsersIntegrationManager(false, `["user2", "user3"]`)
		DeferCleanup(cleanup)
	})

	Describe("GetAdmins with no admin users in allowed list", func() {
		It("should return empty when no admins in allowed list", func() {
			output, err := callTestUsersPlugin(GinkgoT().Context(), manager, testUsersInput{Operation: "get_admins"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Users).To(BeEmpty())
		})
	})
})

var _ = Describe("UsersService Enable Gate", Ordered, func() {
	var manager *Manager

	BeforeAll(func() {
		var cleanup func()
		// Start with disabled plugin, no users configured
		manager, cleanup = setupUsersIntegrationManagerWithEnabled(false, false, "")
		DeferCleanup(cleanup)
	})

	Describe("Enable Gate Behavior", func() {
		It("should block enabling when no users configured and allUsers is false", func() {
			ctx := GinkgoT().Context()
			err := manager.EnablePlugin(ctx, "test-users")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("users permission requires configuration"))
		})

		It("should allow enabling when allUsers is true", func() {
			ctx := GinkgoT().Context()

			// Update the plugin to have allUsers=true
			err := manager.UpdatePluginUsers(ctx, "test-users", "", true)
			Expect(err).ToNot(HaveOccurred())

			// Now enabling should succeed
			err = manager.EnablePlugin(ctx, "test-users")
			Expect(err).ToNot(HaveOccurred())

			// Verify plugin is loaded
			manager.mu.RLock()
			_, ok := manager.plugins["test-users"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
		})

		It("should allow enabling when specific users are configured", func() {
			ctx := GinkgoT().Context()

			// First disable the plugin
			err := manager.DisablePlugin(ctx, "test-users")
			Expect(err).ToNot(HaveOccurred())

			// Update to have specific users (and allUsers=false)
			err = manager.UpdatePluginUsers(ctx, "test-users", `["user1"]`, false)
			Expect(err).ToNot(HaveOccurred())

			// Now enabling should succeed
			err = manager.EnablePlugin(ctx, "test-users")
			Expect(err).ToNot(HaveOccurred())

			// Verify plugin is loaded
			manager.mu.RLock()
			_, ok := manager.plugins["test-users"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
		})
	})
})

// testUsersSetup contains common setup data for users integration tests
type testUsersSetup struct {
	tmpDir   string
	destPath string
	hashHex  string
}

// setupTestUsersPlugin creates a temporary directory with the test-users plugin and returns setup info
func setupTestUsersPlugin() (*testUsersSetup, error) {
	tmpDir, err := os.MkdirTemp("", "users-integration-test-*")
	if err != nil {
		return nil, err
	}

	// Copy the test-users plugin
	srcPath := filepath.Join(testdataDir, "test-users"+PackageExtension)
	destPath := filepath.Join(tmpDir, "test-users"+PackageExtension)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	if err := os.WriteFile(destPath, data, 0600); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	// Compute SHA256 for the plugin
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	return &testUsersSetup{
		tmpDir:   tmpDir,
		destPath: destPath,
		hashHex:  hashHex,
	}, nil
}

// createTestUsers creates standard test users in the mock repo
func createTestUsers(mockUserRepo *tests.MockedUserRepo) {
	_ = mockUserRepo.Put(&model.User{
		ID:       "user1",
		UserName: "alice",
		Name:     "Alice Admin",
		IsAdmin:  true,
	})
	_ = mockUserRepo.Put(&model.User{
		ID:       "user2",
		UserName: "bob",
		Name:     "Bob User",
		IsAdmin:  false,
	})
	_ = mockUserRepo.Put(&model.User{
		ID:       "user3",
		UserName: "charlie",
		Name:     "Charlie User",
		IsAdmin:  false,
	})
}

// setupTestUsersConfig sets up common plugin configuration
func setupTestUsersConfig(tmpDir string) {
	conf.Server.Plugins.Enabled = true
	conf.Server.Plugins.Folder = tmpDir
	conf.Server.Plugins.AutoReload = false
	conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")
}

// testUsersInput represents input for test-users plugin calls
type testUsersInput struct {
	Operation string `json:"operation"`
}

// testUser represents a user returned from test-users plugin
type testUser struct {
	UserName string `json:"userName"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isAdmin"`
}

// testUsersOutput represents output from test-users plugin
type testUsersOutput struct {
	Users []testUser `json:"users,omitempty"`
	Error *string    `json:"error,omitempty"`
}

// callTestUsersPlugin calls the test-users plugin with given input
func callTestUsersPlugin(ctx context.Context, manager *Manager, input testUsersInput) (*testUsersOutput, error) {
	manager.mu.RLock()
	p := manager.plugins["test-users"]
	manager.mu.RUnlock()

	instance, err := p.instance(ctx)
	if err != nil {
		return nil, err
	}
	defer instance.Close(ctx)

	inputBytes, _ := json.Marshal(input)
	_, outputBytes, err := instance.Call("nd_test_users", inputBytes)
	if err != nil {
		return nil, err
	}

	var output testUsersOutput
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, errors.New(*output.Error)
	}
	return &output, nil
}

// setupUsersIntegrationManager creates a Manager for users integration tests with the given plugin settings.
// The plugin is enabled by default.
func setupUsersIntegrationManager(allUsers bool, allowedUsers string) (*Manager, func()) {
	return setupUsersIntegrationManagerWithEnabled(true, allUsers, allowedUsers)
}

// setupUsersIntegrationManagerWithEnabled creates a Manager for users integration tests with full control over plugin state
func setupUsersIntegrationManagerWithEnabled(enabled, allUsers bool, allowedUsers string) (*Manager, func()) {
	setup, err := setupTestUsersPlugin()
	Expect(err).ToNot(HaveOccurred())

	// Setup config
	cleanupConfig := configtest.SetupConfig()
	setupTestUsersConfig(setup.tmpDir)

	// Setup mock DataStore with plugin and users
	mockPluginRepo := tests.CreateMockPluginRepo()
	mockPluginRepo.Permitted = true
	mockPluginRepo.SetData(model.Plugins{{
		ID:       "test-users",
		Path:     setup.destPath,
		SHA256:   setup.hashHex,
		Enabled:  enabled,
		AllUsers: allUsers,
		Users:    allowedUsers,
	}})

	mockUserRepo := tests.CreateMockUserRepo()
	createTestUsers(mockUserRepo)

	dataStore := &tests.MockDataStore{
		MockedPlugin: mockPluginRepo,
		MockedUser:   mockUserRepo,
	}

	// Create and start manager
	manager := &Manager{
		plugins:        make(map[string]*plugin),
		ds:             dataStore,
		subsonicRouter: http.NotFoundHandler(),
	}
	err = manager.Start(GinkgoT().Context())
	Expect(err).ToNot(HaveOccurred())

	cleanup := func() {
		_ = manager.Stop()
		_ = os.RemoveAll(setup.tmpDir)
		cleanupConfig()
	}

	return manager, cleanup
}
