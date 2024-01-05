package genevahttp

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/getlantern/algeneva"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func TestWebsocket(t *testing.T) {
	firstMsg := "up-up-down-down-left-right-left-right-b-a-start"
	secongMsg := "void insert(s *char) { list[i++] = s; }"
	firstResp := "cheat activated"
	secondResp := "segfault"

	// Create a listener that will accept connections from the client.
	ll, err := NewListener("tcp", ":8080", "")
	require.NoError(t, err, "Failed to create listener")

	go startTestServer(ll, t, []string{firstResp, secondResp})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer func() {
		cancel()
		ll.Close()
	}()

	time.Sleep(1 * time.Second)

	// create dialer with the strategy we want to use. should we test with multiple strategies?
	opts := DialerOpts{
		AlgenevaStrategy: algeneva.Strategies["China"][8],
	}
	c, err := DialContext(ctx, "tcp", "localhost:8080", opts)
	require.NoError(t, err, "Failed to dial")

	_, err = c.Write([]byte(firstMsg))
	require.NoError(t, err, "client: Failed to write")

	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	require.NoError(t, err, "client: Failed to read")
	require.Equal(t, firstResp, string(buf[:n]))

	_, err = c.Write([]byte(secongMsg))
	require.NoError(t, err, "client: Failed to write")

	n, err = c.Read(buf)
	require.NoError(t, err, "client: Failed to read")
	require.Equal(t, secondResp, string(buf[:n]))

	c.Close()
}

// startTestServer starts a test server that will reply to messages for testing
func startTestServer(ll net.Listener, t *testing.T, resps []string) error {
	for {
		c, err := ll.Accept()
		if err != nil {
			return fmt.Errorf("Failed to accept: %w", err)
		}

		h := &handler{resps: resps}
		go h.handleConn(c, t)
	}
}

type handler struct {
	resps []string
}

func (h *handler) handleConn(c net.Conn, t *testing.T) error {
	defer c.Close()

	for _, resp := range h.resps {
		err := reply(c, resp, t)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return nil
		}

		if err != nil {
			return fmt.Errorf("server failed to reply: %w. Closing connection", err)
		}
	}

	return nil
}

func reply(c net.Conn, resp string, t *testing.T) error {
	buf := make([]byte, 1024)
	_, err := c.Read(buf)
	if err != nil {
		return fmt.Errorf("Failed to read: %w", err)
	}

	_, err = c.Write([]byte(resp))
	if err != nil {
		return fmt.Errorf("Failed to write: %w", err)
	}

	return nil
}
