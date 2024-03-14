package genevahttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/getlantern/algeneva"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

var testMsgs = map[string]string{
	"up-up-down-down-left-right-left-right-b-a-start": "cheat activated",
	"void insert(s *char) { list[i++] = s; }":         "segfault",
}

func TestWebsocket(t *testing.T) {
	// Create a listener that will accept connections from the client.
	l, err := net.Listen("tcp", ":8080")
	require.NoError(t, err)

	ll, _ := WrapListener(l)
	require.NoError(t, err, "Failed to create listener")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

	done := make(chan error)
	go func() {
		t.Log("starting test server")
		done <- startTestServer(ctx, ll)
	}()

	defer func() {
		cancel()

		ll.Close()
		<-ll.(*listener).closed // wait for the listener to close
		t.Log("listener closed")

		err := <-done // wait for the test server to close
		t.Log("server closed")
		if errors.Is(err, io.EOF) {
			return
		}

		assert.NoError(t, err, "server close err")
	}()

	// give the server time to start
	time.Sleep(time.Second)

	opts := DialerOpts{AlgenevaStrategy: algeneva.Strategies["China"][17]}
	c, err := DialContext(ctx, "tcp", "localhost:8080", opts)
	require.NoError(t, err, "Failed to dial")
	defer c.Close()

	t.Log("dialer connected")
	t.Log("testing communication..")

	buf := make([]byte, 1024)
	for msg, resp := range testMsgs {
		_, err = c.Write([]byte(msg))
		require.NoError(t, err, "client: Failed to write")
		n, err := c.Read(buf)
		require.NoError(t, err, "client: Failed to read")
		require.Equal(t, resp, string(buf[:n]))
	}
}

// startTestServer starts a test server to handle a websocket connection. ctx is used to close the
// connection when the test is done.
func startTestServer(ctx context.Context, ll net.Listener) error {
	c, err := ll.Accept()
	switch {
	case websocket.CloseStatus(err) == websocket.StatusNormalClosure:
		return errors.New("testServer: accepted closed connection")
	case err != nil:
		return err
	}

	go func() {
		<-ctx.Done()
		c.Close()
	}()

	for {
		select {
		case err := <-ll.(*listener).wsConnErrC:
			return err
		default:
		}

		err := reply(c)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}

			return err
		}
	}
}

func reply(c net.Conn) error {
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		return fmt.Errorf("Failed to read: %w", err)
	}
	// get the expected response for the message received
	msg := string(buf[:n])
	resp, ok := testMsgs[msg]
	if !ok {
		resp = "unknown message: " + msg
	}
	if _, err = c.Write([]byte(resp)); err != nil {
		return fmt.Errorf("Failed to write: %w", err)
	}

	return nil
}
