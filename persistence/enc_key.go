package persistence

// encryptionKey returns the 32-byte AES-GCM key used to encrypt user passwords
// and app-password secrets.
//
// The key is initialised by NewUserRepository's sync.Once on first call to any
// user repository constructor; that path also performs the one-time
// re-encryption migration when conf.Server.PasswordEncryptionKey is rotated.
// Repositories that depend on this key (e.g. app_password_repository) must
// therefore ensure a user repository has been constructed at least once before
// they invoke this accessor — see appPasswordRepository's constructor for the
// canonical pattern.
func encryptionKey() []byte {
	return encKey
}
