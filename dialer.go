package genevahttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/getlantern/algeneva"
	"nhooyr.io/websocket"
)

// Dialer is the interface used to establish connections to the server.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DialerOpts contains options for the Dialer.
type DialerOpts struct {
	// AlgenevaStrategy is the geneva HTTPStrategy to apply to the connect request.
	AlgenevaStrategy string
	strategy         *algeneva.HTTPStrategy
	// Dialer is the dialer used to connect to the server. If AlgenevaStrategy is not empty, the
	// strategy will be applied to the request made by Dialer.Dial for all connections. If nil, the
	// default dialer is used.
	Dialer    Dialer
	TLSConfig *tls.Config
}

// Dial performs a websocket handshake over TCP with the given address. If opts.AlgenevaStrategy is
// not empty, it will apply the geneva strategy to the connect request.
// Dial uses the background context; to specify a context, use DialContext.
func Dial(network, address string, opts DialerOpts) (net.Conn, error) {
	return DialContext(context.Background(), "TCP", address, opts)
}

// DialContext performs a websocket handshake over TCP with the given address using the provided
// context. If opts.AlgenevaStrategy is not empty, it will be applied to the handshake request.
func DialContext(ctx context.Context, network, address string, opts DialerOpts) (net.Conn, error) {
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
	wsc, _, err := websocket.Dial(ctx, "ws://"+address, wsopts)
	if err != nil {
		return nil, err
	}

	conn := websocket.NetConn(context.Background(), wsc, websocket.MessageBinary)
	if opts.TLSConfig == nil {
		return conn, nil
	}

	tlsConn := tls.Client(conn, opts.TLSConfig)
	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close() // not sure if this is necessary or if it's done by Handshake
		return nil, err
	}

	return tlsConn, nil
}

// dialContext returns a dial function that connects to the given address and wraps the resulting
// connection with a httpTransformConn. If opts.Dialer is not nil, dialContext will use it to
// establish the connection. Otherwise, the default dialer is used.
func dialContext(opts DialerOpts) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		dialer := opts.Dialer
		if dialer == nil {
			dialer = &net.Dialer{}
		}

		cc, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return &httpTransformConn{Conn: cc, httpTransform: opts.strategy}, nil
	}
}
