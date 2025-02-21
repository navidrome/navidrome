package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/navidrome/navidrome/log"
)

func Encrypt(ctx context.Context, encKey []byte, data string) (string, error) {
	plaintext := []byte(data)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		log.Error(ctx, "Could not create a cipher", err)
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(ctx, "Could not create a GCM", "user", err)
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		log.Error(ctx, "Could generate nonce", err)
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ctx context.Context, encKey []byte, encData string) (value string, err error) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("decryption panicked")
		}
	}()

	enc, _ := base64.StdEncoding.DecodeString(encData)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		log.Error(ctx, "Could not create a cipher", err)
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(ctx, "Could not create a GCM", err)
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Error(ctx, "Could not decrypt password", err)
		return "", err
	}

	return string(plaintext), nil
}
