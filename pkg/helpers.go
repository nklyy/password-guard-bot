package pkg

import (
	"crypto/sha1"

	"golang.org/x/crypto/pbkdf2"
)

func GenerateNormalSizeCode(currentCode string) []byte {
	dk := pbkdf2.Key([]byte(currentCode), []byte{}, 4096, 32, sha1.New)

	return dk
}
