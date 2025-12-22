package plugins

//go:generate go tool go-jsonschema -p plugins --struct-name-from-title -o manifest_gen.go manifest.json

// AllowedHosts returns a list of allowed hosts for HTTP requests.
// Returns the hosts directly from the manifest's permissions.
func (m *Manifest) AllowedHosts() []string {
	if m.Permissions == nil || m.Permissions.Http == nil {
		return nil
	}
	return m.Permissions.Http.AllowedHosts
}

// TODO: ConfigPermission is defined in the schema but not currently enforced.
// Plugins always receive their config section. Implement permission checking or remove from schema.
