package genevahttp

import (
	"context"
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
	// srv is the server that listens for websocket connections and converts them to a net.Conn.
	srv *http.Server

	// connections is a channel of net.Conns that the listener will hand out.
	connections chan net.Conn
	// closed is closed when srv is closed.
	closed chan struct{}
	// wsConnErrC is a channel that will receive any errors from srv when accepting a websocket
	// connection.
	wsConnErrC chan error
	// srvErr will hold any error explaining why the server was closed.
	srvErr error
}

// WrapListener wraps l in a net.Listener to handle requests sent by a lantern-algeneva client.
// WrapListener returns the wrapped listener and a channel to receive any errors encountered when
// a client tries to connect.
func WrapListener(l net.Listener) (net.Listener, <-chan error) {
	l = &innerListener{l}
	ll := &listener{
		listener:    l,
		connections: make(chan net.Conn),
		closed:      make(chan struct{}),
		wsConnErrC:  make(chan error, 20),
	}

	// Start a server to accept websocket connections and convert them to a normalizationConn.
	// The connections are then added to ll.connections to be handed out by ll.Accept. We could
	// implement the listener without an underlying server, but we would have to implement a
	// http.ResponseWriter and http.Hijacker for the websocket handshake. This just seems simpler.
	srv := &http.Server{
		Handler:      http.HandlerFunc(ll.handleFunc),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		ll.srvErr = srv.Serve(l)
		close(ll.closed)
	}()

	ll.srv = srv

	return ll, ll.wsConnErrC
}

// Accept implements net.Listener. It is the caller's responsibility to close the connection when
// done.
func (ll *listener) Accept() (net.Conn, error) {
	select {
	case c := <-ll.connections:
		return c, nil
	case <-ll.closed:
		return nil, ll.srvErr
	}
}

// Close implements net.Listener. Any connections handed out by ll.Accept will not be closed and
// must be closed manually.
func (ll *listener) Close() error {
	ll.mx.Lock()
	defer ll.mx.Unlock()
	select {
	case <-ll.closed:
		return nil
	default:
		return ll.srv.Close()
	}
}

// Addr implements net.Listener.
func (ll *listener) Addr() net.Addr {
	return ll.listener.Addr()
}

// handleFunc handles websocket connections and converts them to net.Conn. Any errors encountered
// during the process will be sent to ll.wsConnErrC.
func (ll *listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	wsc, err := websocket.Accept(w, r, nil)
	if err != nil {
		sendError(err, ll.wsConnErrC)
		return
	}

	c := websocket.NetConn(context.Background(), wsc, websocket.MessageBinary)

	// Wait for someone to call ll.Accept to hand out the connection or for the server to close.
	rctx := r.Context()
	select {
	case ll.connections <- c:
	case <-rctx.Done():
		c.Close()
	}
}

// sendError sends err to c if c is not full. If c is full, the error is dropped.
func sendError(err error, c chan<- error) {
	select {
	case c <- err:
	default:
	}
}

// innerListener is a net.Listener that wraps connections in a normalizationConn.
type innerListener struct {
	net.Listener
}

// Accept implements net.Listener and wraps the connection in a normalizationConn.
func (il *innerListener) Accept() (net.Conn, error) {
	c, err := il.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &normalizationConn{Conn: c}, nil
}
