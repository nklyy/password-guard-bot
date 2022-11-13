package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"
)

// decrypt from base64 to decrypted string
func Decrypt(pin []byte, data string) string {
	ciphertext, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(data))

	block, err := aes.NewCipher(pin)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext)
}
