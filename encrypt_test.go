package genevahttp

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_encryptConn(t *testing.T) {
	key := []byte("0123456789abcdef")
	plainText := "don't only practice your art, but force your way into its secrets"
	cipherText := "6ff47bfd3f64cf9b7964efb4b27e56a1d09e30bd19072d953b36a456fc5b44645c2c03c658ecc22c213e32deb1cc0fd7cfc61d3d6a8ecdc6683f938999a2537a26"

	tc := &testConn{}
	ec, err := encryptConn(tc, key)
	require.NoError(t, err)

	_, err = ec.Write([]byte(plainText))
	require.NoError(t, err)

	// convert to hex string to compare
	encrypted := fmt.Sprintf("%x", tc.cipherText)
	assert.Equal(t, cipherText, encrypted)

	buf := make([]byte, len(plainText))
	_, err = ec.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, plainText, string(buf))
}

type testConn struct {
	net.Conn
	cipherText []byte
}

func (c *testConn) Read(b []byte) (n int, err error) {
	n = copy(b, c.cipherText)
	return n, nil
}

func (c *testConn) Write(b []byte) (n int, err error) {
	c.cipherText = b
	return len(b), nil
}
