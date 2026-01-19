package tests

import (
	"context"
)

// MockPluginManager is a mock implementation of plugins.PluginManager for testing.
// It implements EnablePlugin, DisablePlugin, UpdatePluginConfig, ValidatePluginConfig, UpdatePluginUsers, UpdatePluginLibraries and RescanPlugins methods.
type MockPluginManager struct {
	// EnablePluginFn is called when EnablePlugin is invoked. If nil, returns EnableError.
	EnablePluginFn func(ctx context.Context, id string) error
	// DisablePluginFn is called when DisablePlugin is invoked. If nil, returns DisableError.
	DisablePluginFn func(ctx context.Context, id string) error
	// UpdatePluginConfigFn is called when UpdatePluginConfig is invoked. If nil, returns ConfigError.
	UpdatePluginConfigFn func(ctx context.Context, id, configJSON string) error
	// ValidatePluginConfigFn is called when ValidatePluginConfig is invoked. If nil, returns ValidateError.
	ValidatePluginConfigFn func(ctx context.Context, id, configJSON string) error
	// UpdatePluginUsersFn is called when UpdatePluginUsers is invoked. If nil, returns UsersError.
	UpdatePluginUsersFn func(ctx context.Context, id, usersJSON string, allUsers bool) error
	// UpdatePluginLibrariesFn is called when UpdatePluginLibraries is invoked. If nil, returns LibrariesError.
	UpdatePluginLibrariesFn func(ctx context.Context, id, librariesJSON string, allLibraries bool) error
	// RescanPluginsFn is called when RescanPlugins is invoked. If nil, returns RescanError.
	RescanPluginsFn func(ctx context.Context) error

	// Default errors to return when Fn callbacks are not set
	EnableError    error
	DisableError   error
	ConfigError    error
	ValidateError  error
	UsersError     error
	LibrariesError error
	RescanError    error

	// Track calls for assertions
	EnablePluginCalls       []string
	DisablePluginCalls      []string
	UpdatePluginConfigCalls []struct {
		ID         string
		ConfigJSON string
	}
	ValidatePluginConfigCalls []struct {
		ID         string
		ConfigJSON string
	}
	UpdatePluginUsersCalls []struct {
		ID        string
		UsersJSON string
		AllUsers  bool
	}
	UpdatePluginLibrariesCalls []struct {
		ID            string
		LibrariesJSON string
		AllLibraries  bool
	}
	RescanPluginsCalls int
}

func (m *MockPluginManager) EnablePlugin(ctx context.Context, id string) error {
	m.EnablePluginCalls = append(m.EnablePluginCalls, id)
	if m.EnablePluginFn != nil {
		return m.EnablePluginFn(ctx, id)
	}
	return m.EnableError
}

func (m *MockPluginManager) DisablePlugin(ctx context.Context, id string) error {
	m.DisablePluginCalls = append(m.DisablePluginCalls, id)
	if m.DisablePluginFn != nil {
		return m.DisablePluginFn(ctx, id)
	}
	return m.DisableError
}

func (m *MockPluginManager) UpdatePluginConfig(ctx context.Context, id, configJSON string) error {
	m.UpdatePluginConfigCalls = append(m.UpdatePluginConfigCalls, struct {
		ID         string
		ConfigJSON string
	}{ID: id, ConfigJSON: configJSON})
	if m.UpdatePluginConfigFn != nil {
		return m.UpdatePluginConfigFn(ctx, id, configJSON)
	}
	return m.ConfigError
}

func (m *MockPluginManager) ValidatePluginConfig(ctx context.Context, id, configJSON string) error {
	m.ValidatePluginConfigCalls = append(m.ValidatePluginConfigCalls, struct {
		ID         string
		ConfigJSON string
	}{ID: id, ConfigJSON: configJSON})
	if m.ValidatePluginConfigFn != nil {
		return m.ValidatePluginConfigFn(ctx, id, configJSON)
	}
	return m.ValidateError
}

func (m *MockPluginManager) UpdatePluginUsers(ctx context.Context, id, usersJSON string, allUsers bool) error {
	m.UpdatePluginUsersCalls = append(m.UpdatePluginUsersCalls, struct {
		ID        string
		UsersJSON string
		AllUsers  bool
	}{ID: id, UsersJSON: usersJSON, AllUsers: allUsers})
	if m.UpdatePluginUsersFn != nil {
		return m.UpdatePluginUsersFn(ctx, id, usersJSON, allUsers)
	}
	return m.UsersError
}

func (m *MockPluginManager) UpdatePluginLibraries(ctx context.Context, id, librariesJSON string, allLibraries bool) error {
	m.UpdatePluginLibrariesCalls = append(m.UpdatePluginLibrariesCalls, struct {
		ID            string
		LibrariesJSON string
		AllLibraries  bool
	}{ID: id, LibrariesJSON: librariesJSON, AllLibraries: allLibraries})
	if m.UpdatePluginLibrariesFn != nil {
		return m.UpdatePluginLibrariesFn(ctx, id, librariesJSON, allLibraries)
	}
	return m.LibrariesError
}

func (m *MockPluginManager) RescanPlugins(ctx context.Context) error {
	m.RescanPluginsCalls++
	if m.RescanPluginsFn != nil {
		return m.RescanPluginsFn(ctx)
	}
	return m.RescanError
}

func (m *MockPluginManager) UnloadDisabledPlugins(ctx context.Context) {
	// No-op for mock - plugins are not actually loaded in tests
}
