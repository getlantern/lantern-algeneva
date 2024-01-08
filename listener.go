package genevahttp

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// listener listens for websocket connections and converts them to net.Conn.
type listener struct {
	// underlying listener
	listener net.Listener
	mx       sync.Mutex
	srv      *http.Server

	connections chan net.Conn
	close       chan struct{}

	// encryptionKey is the key to use to encrypt the body of the request (wrapped request) using
	// AES. EncryptionKey must be 16, 24 or 32 bytes long which will use AES-128, AES-192, or
	// AES-256, respectively, with longer keys being more secure. If EncryptionKey is nil, the
	// connection will not be encrypted.
	encryptionKey []byte
}

// WrapListener wraps l in a net.Listener to handle requests sent by a lantern-algeneva client. If
// encryptionKey is not empty, the returned net.Listener will also decrypt the body of the request
// (wrapped request) using AES with the provided key. key must be 16, 24 or 32 bytes long to select
// AES-128, AES-192, or AES-256 respectively with longer keys being more secure.
func WrapListener(l net.Listener, encryptionKey string) (net.Listener, error) {
	var key []byte
	if encryptionKey != "" {
		var err error
		key, err = hex.DecodeString(encryptionKey)
		if err != nil {
			return nil, err
		}
	}

	l = &innerListener{l}
	ll := &listener{
		listener:      l,
		connections:   make(chan net.Conn, 100),
		close:         make(chan struct{}),
		encryptionKey: key,
	}

	// create a server that handles websocket connections and converts them to net.Conn
	srv := &http.Server{
		Handler:      http.HandlerFunc(ll.handleFunc),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go srv.Serve(l)
	ll.srv = srv

	return ll, nil
}

// Accept waits for and returns the next connection to the listener.
func (ll *listener) Accept() (net.Conn, error) {
	select {
	case c, ok := <-ll.connections:
		if !ok {
			return nil, errors.New("listener closed")
		}

		return c, nil
	case <-ll.close:
		return nil, errors.New("listener closed")
	}
}

// Close closes the listener.
func (ll *listener) Close() error {
	ll.mx.Lock()
	defer ll.mx.Unlock()
	select {
	case <-ll.close:
		return nil
	default:
		close(ll.close)
		return ll.srv.Close()
	}
}

// Addr returns the listener's network address.
func (ll *listener) Addr() net.Addr {
	return ll.listener.Addr()
}

// handleFunc handles websocket connections and converts them to net.Conn. If encryptionKey is set,
// it will try to encrypt the connection using the provided key. If the encryption fails, the
// connection will immediately be closed.
func (ll *listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	// TODO: handle errors. should we log them? or attach them to the conn and still send it to the
	// connections channel? This would allow the caller to see why it failed.
	wsc, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := websocket.NetConn(ctx, wsc, websocket.MessageBinary)

	if ll.encryptionKey != nil {
		ec, err := encryptConn(c, ll.encryptionKey)
		if err != nil {
			c.Close()
			cancel()
			return
		}

		c = ec
	}

	ll.connections <- c
}

// innerListener is a net.Listener that wraps connections with conn.
type innerListener struct {
	net.Listener
}

// Accept implements net.Listener and wraps the connection with conn.
func (il *innerListener) Accept() (net.Conn, error) {
	c, err := il.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &conn{Conn: c, isClient: false}, nil
}
