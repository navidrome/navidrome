package id

import (
	"crypto/md5"
	"fmt"
	"math/big"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
)

func NewRandom() string {
	id, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 22)
	if err != nil {
		log.Error("Could not generate new ID", err)
	}
	return id
}

func NewHash(data ...string) string {
	hash := md5.New()
	for _, d := range data {
		hash.Write([]byte(d))
		hash.Write([]byte(string('\u200b')))
	}
	h := hash.Sum(nil)
	bi := big.NewInt(0)
	bi.SetBytes(h)
	s := bi.Text(62)
	return fmt.Sprintf("%022s", s)
}

func NewTagID(name, value string) string {
	return NewHash(strings.ToLower(name), strings.ToLower(value))
}
