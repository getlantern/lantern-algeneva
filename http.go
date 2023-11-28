package genevahttp

import (
	"context"
	"net"

	"github.com/getlantern/algeneva"
)

type Client struct {
	geneva *algeneva.Client
	key    []byte
}

// NewClient wraps an algeneva client to provide a net.Conn that encrypts/decrypts the body of the request (or inner
// request) using AES with the provided key. key must be 16, 24 or 32 bytes long to select AES-128, AES-192, or
// AES-256 respectively with longer keys being more secure.
func NewClient(strategy string, key []byte) (*Client, error) {
	client, err := algeneva.NewClient(strategy)
	if err != nil {
		return nil, err
	}

	return &Client{geneva: client, key: key}, nil
}

// Dial connects to the address on the named network and returns a genevahttp.Conn.
// See net.Dial (https://pkg.go.dev/net#Dial) for more information about network and address parameters.
// Dial uses context.Background() internally; use DialContext to specify a context.
func (c *Client) Dial(network, address string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network using the provided context and returns genevahttp.Conn.
//
// See net.Dial (https://pkg.go.dev/net#Dial) for more information about network and address parameters.
//
// The provided Context must be non-nil. If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not affect the connection.
//
// When using TCP, and the host in the address parameter resolves to multiple network addresses, any dial timeout
// (from d.Timeout or ctx) is spread over each consecutive dial, such that each is given an appropriate fraction of
// the time to connect. For example, if a host has 4 IP addresses and the timeout is 1 minute, the connect to each
// single address will be given 15 seconds to complete before trying the next one.
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	cc, err := c.geneva.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: cc, key: c.key}, nil
}

// listener is a wrapper around net.Listener that encrypts/decrypts the body of each request using AES.
type listener struct {
	net.Listener
	key []byte
}

// NewListener wraps a net.Listener that encrypts/decrypts the body of the request (or inner request) using AES with
// the provided key. key must be 16, 24 or 32 bytes long to select AES-128, AES-192, or AES-256 respectively with
// longer keys being more secure.
func NewListener(network, address string, key []byte) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return &listener{Listener: l, key: key}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &Conn{conn: c, key: l.key}, nil
}
