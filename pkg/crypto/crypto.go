package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

type CryptoService interface {
	Encrypt(pin []byte, data []byte) (string, error)
	Decrypt(pin []byte, data string) (string, error)
	GenerateNormalSizeCode(currentCode string) []byte
}

type crypto struct {
}

func NewCryptoService() (CryptoService, error) {
	return &crypto{}, nil
}

func (c *crypto) Encrypt(pin []byte, data []byte) (string, error) {
	block, err := aes.NewCipher(pin)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	// convert to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *crypto) Decrypt(pin []byte, data string) (string, error) {
	ciphertext, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(data))

	block, err := aes.NewCipher(pin)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

func (c *crypto) GenerateNormalSizeCode(currentCode string) []byte {
	dk := pbkdf2.Key([]byte(currentCode), []byte{}, 4096, 32, sha1.New)

	return dk
}
