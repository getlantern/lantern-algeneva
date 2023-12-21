package genevahttp

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net"
)

// ErrEncryptionKey is returned when the encryption key is invalid.
var ErrEncryptionKey = errors.New("encryption key error")

// encrypter is a wrapper around a net.Conn that encrypts all data sent and received with the
// given key.
type encrypter struct {
	net.Conn
	// reader decrypts data read from the connection
	reader *cipher.StreamReader
	// writer encrypts data written to the connection
	writer *cipher.StreamWriter
}

// encryptConn wraps conn with an encrypter that encrypts all data sent and received with the
// given key. The key must be 16, 24 or 32 bytes long which will use AES-128, AES-192, or AES-256
// respectively.
func encryptConn(conn net.Conn, key []byte) (net.Conn, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionKey, err)
	}

	var riv, wiv [aes.BlockSize]byte
	rstream := cipher.NewOFB(block, riv[:])
	wstream := cipher.NewOFB(block, wiv[:])
	return &encrypter{
		Conn:   conn,
		reader: &cipher.StreamReader{S: rstream, R: conn},
		writer: &cipher.StreamWriter{S: wstream, W: conn},
	}, nil
}

// Read decrypts data read from the connection.
func (e *encrypter) Read(p []byte) (int, error) {
	return e.reader.Read(p)
}

// Write encrypts data written to the connection.
func (e *encrypter) Write(p []byte) (int, error) {
	return e.writer.Write(p)
}
