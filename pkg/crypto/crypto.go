package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
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
	iteration int
}

func NewCryptoService(iteration *int) (CryptoService, error) {
	if iteration == nil {
		return nil, errors.New("incorrect iteration")
	}

	return &crypto{iteration: *iteration}, nil
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
	dk := pbkdf2.Key([]byte(currentCode), []byte{}, c.iteration, 32, sha512.New)

	return dk
}
