package apiauth

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/navidrome/navidrome/model/id"
)

const secretPrefix = "ndg_"

func newSecret() (secret, hash string) {
	secret = secretPrefix + id.NewRandom()
	return secret, hashSecret(secret)
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
