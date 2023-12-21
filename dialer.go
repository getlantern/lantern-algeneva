package genevahttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/algeneva"
	"nhooyr.io/websocket"
)

// DialerOpts contains options for the Dialer.
type DialerOpts struct {
	// AlgenevaStrategy is the geneva HTTPStrategy to apply to the connect request.
	AlgenevaStrategy string
	strategy         *algeneva.HTTPStrategy

	// EncryptionKey is the key to use to encrypt the body of the request (wrapped request)
	// using AES. EncryptionKey must be 16, 24 or 32 bytes long which will use AES-128, AES-192, or
	// AES-256, respectively, with longer keys being more secure. If EncryptionKey is nil, the
	// connection will not be encrypted, or if TLSConfig is set, that will be used instead.
	EncryptionKey []byte
	// TLSConfig is the TLS configuration to use for the connection.
	// NOTE: currently not supported.
	TLSConfig *tls.Config
}

// Dial performs a websocket handshake with the given address. If opts.AlgenevaStrategy is set, it
// will apply the geneva strategy to the connect request. Dial uses the background context, to
// specify a context, use DialContext.
func Dial(network, address string, opts DialerOpts) (net.Conn, error) {
	return DialContext(context.Background(), network, address, opts)
}

// DialContext performs a websocket handshake with the given address using the provided context.
// If opts.AlgenevaStrategy is set, it will apply the geneva strategy to the connect request.
func DialContext(ctx context.Context, network, address string, opts DialerOpts) (net.Conn, error) {
	switch proto, _, _ := strings.Cut(address, "://"); proto {
	case "http", "https", "ws", "wss":
	case address:
		address = "http://" + address
	default:
		return nil, errors.New("unsupported protocol")
	}

	if opts.AlgenevaStrategy != "" {
		strategy, err := algeneva.NewHTTPStrategy(opts.AlgenevaStrategy)
		if err != nil {
			return nil, fmt.Errorf("failed to create geneva strategy: %w", err)
		}
		opts.strategy = strategy
	}
	wsopts := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{DialContext: dialContext(opts)},
		},
	}
	wsc, _, err := websocket.Dial(ctx, address, wsopts)
	if err != nil {
		return nil, err
	}

	c := websocket.NetConn(ctx, wsc, websocket.MessageBinary)
	if opts.TLSConfig == nil && opts.EncryptionKey != nil {
		// TODO: should we close the connection if we fail to encrypt the connection? or should we just return the
		// unencrypted connection with an error specifying that the connection is not encrypted and let the caller
		// decide?
		if c, err = encryptConn(c, opts.EncryptionKey); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

// dialContext returns a dial function that wraps the connection with a conn that applies the
// geneva strategy if set.
func dialContext(opts DialerOpts) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		cc, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return &conn{Conn: cc, httpTransform: opts.strategy, isClient: true}, nil
	}
}
