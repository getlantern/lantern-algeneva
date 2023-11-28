package genevahttp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

// encrypt encrypts plaintext with key using AES. key must be 16, 24 or 32 bytes long to select AES-128, AES-192,
// or AES-256 respectively. Longer keys are more secure. The IV will be generated and prepended to the plaintext and
// then padding will be added, if necessary, upto block size (16 bytes).
func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, so we randomly generate it and then prepend it to the plaintext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// decrypt decrypts ciphertext with key and returns the plaintext. ciphertext should still have the IV prepended that
// was added during encryption. ErrMissingRequest is returned if the ciphertext is less than the block size (16 bytes).
func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// If the ciphertext is less than the block size (16 bytes), then the request is missing.
	if len(ciphertext) < aes.BlockSize {
		return nil, ErrMissingRequest
	}

	iv, ciphertext := ciphertext[:aes.BlockSize], ciphertext[aes.BlockSize:]
	plaintext := make([]byte, len(ciphertext))

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}
