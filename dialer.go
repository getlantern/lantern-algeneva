package genevahttp

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/getlantern/algeneva"
	"nhooyr.io/websocket"
)

// DialerOpts contains options for the Dialer.
type DialerOpts struct {
	// AlgenevaStrategy is the geneva HTTPStrategy to apply to the connect request.
	AlgenevaStrategy string
	strategy         *algeneva.HTTPStrategy
}

// Dial performs a websocket handshake over TCP with the given address. If opts.AlgenevaStrategy is
// set, it will apply the geneva strategy to the connect request. Dial uses the background context;
// to specify a context, use DialContext.
func Dial(_, address string, opts DialerOpts) (net.Conn, error) {
	return DialContext(context.Background(), "TCP", address, opts)
}

// DialContext performs a websocket handshake over TCP with the given address using the provided
// context. If opts.AlgenevaStrategy is set, it will be applied to the handshake request.
func DialContext(ctx context.Context, _, address string, opts DialerOpts) (net.Conn, error) {
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

	return websocket.NetConn(ctx, wsc, websocket.MessageBinary), nil
}

// dialContext returns a dial function that wraps the connection with a httpTransformConn.
func dialContext(opts DialerOpts) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		cc, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return &httpTransformConn{Conn: cc, httpTransform: opts.strategy}, nil
	}
}
