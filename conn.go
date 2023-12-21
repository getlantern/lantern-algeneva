package genevahttp

import (
	"bytes"
	"errors"
	"log"
	"net"

	"github.com/getlantern/algeneva"
)

// conn is a wrapper around a net.conn. conn behaves differently depending on whether it is a
// client or server connection. For client connections, conn will apply the configured geneva
// strategy to the request if it hasn't already been upgraded to a websocket connection. For server
// connections, conn will unwrap the request, removing the modified false headers added by clients.
type conn struct {
	// underlying connection
	net.Conn
	// geneva strategy to apply to requests if we're a client. This is ignored if we're a server.
	httpTransform *algeneva.HTTPStrategy

	isClient bool
	upgraded bool
}

// Read reads data from the connection. For server connections, read will unwrap the request,
// removing the modified false headers added by clients. Once the connection is upgraded, read
// will read the data as is.
func (c *conn) Read(b []byte) (n int, err error) {
	if c.upgraded {
		// we don't need to modify the request so just read
		return c.Conn.Read(b)
	}

	if c.isClient {
		// if we're a client, we need to check if the connection was upgraded
		n, err = c.Conn.Read(b)
		if err == nil && bytes.Contains(b, []byte("101 Switching Protocols")) {
			c.upgraded = true
		}

		{ ///////////// >>>>> DEBUG
			log.Printf("client read: \r\n%s", b[:n])
			if c.upgraded {
				log.Printf("client upgraded to websocket")
			}
		} ///////////// <<<<< DEBUG

		return n, err
	}

	// if we're a server, we need to check if the request is wrapped and unwrap it
	// some of the geneva strategies increase the size of the request significantly and the buffer
	// passed in may not be large enough so we have to create our own buffer to read into. We copy
	// the unwrapped request into the buffer later.
	buf := make([]byte, 16384) // 16kb
	n, err = c.Conn.Read(buf)
	if err != nil {
		{ ///////////// >>>>> DEBUG
			log.Printf("server conn read error: %v\r\n", err)
		} ///////////// <<<<< DEBUG

		return n, err
	}

	unwrapped, err := unwrap(buf[:n])
	if err != nil {
		return n, err
	}

	{ ///////////// >>>>> DEBUG
		log.Printf("server conn read: \r\n%s", unwrapped)
	} ///////////// <<<<< DEBUG

	n = copy(b, unwrapped)
	return n, err
}

// Write writes data to the connection. For client connections, write will apply the configured
// geneva strategy to the request if it hasn't already been upgraded to a websocket connection.
// Once the connection is upgraded, write will write the data as is.
func (c *conn) Write(b []byte) (n int, err error) {
	if c.upgraded {
		// we don't need to modify the request so just write
		return c.Conn.Write(b)
	}

	{ //////////////// >>>> DEBUG
		if c.isClient {
			log.Printf("client conn write: \r\n%s", b)

			if bytes.Contains(b, []byte("Connection: Upgrade")) {
				log.Printf("client upgrading to websocket")
			}
		} else {
			log.Printf("server conn write: \r\n%s", b)
		}
	} //////////////// <<<<< DEBUG

	// apply the transform if we're a client
	if c.isClient && c.httpTransform != nil {
		b, err = applyTransformAndWrap(b, c.httpTransform)
		if err != nil {
			return 0, err
		}
	}

	n, err = c.Conn.Write(b)
	if err != nil {
		// TODO: should we return the number of bytes of the original request that were written or
		// the transformed request?
		return n, err
	}

	// if we're a server, mark the connection as upgraded if the response contains the upgrade header
	if !c.isClient && bytes.Contains(b, []byte("101 Switching Protocols")) {
		c.upgraded = true
	}

	return n, err
}

// /////////// >>>>> DEBUG

// this is just for debugging. Unless we want to log when connections are closed.
func (c *conn) Close() error {
	if c.isClient {
		log.Printf("client closing connection")
	} else {
		log.Printf("server closing connection")
	}

	return c.Conn.Close()
} ///////////// <<<<< DEBUG

// applyTransformAndWrap uses the geneva strategy to transform a copy of the startline and headers
// and returns a new request with the transformed headers and the original request as the body.
func applyTransformAndWrap(b []byte, transform *algeneva.HTTPStrategy) ([]byte, error) {
	modb, err := transform.Apply(b)
	if err != nil {
		return b, err
	}

	modb = append(modb, b...)
	return modb, nil
}

// unwrap unwraps the body of a request discarding the headers. If the request is not wrapped, the
// original request is returned.
func unwrap(b []byte) ([]byte, error) {
	idx := bytes.Index(b, []byte("\r\n\r\n"))
	if idx == -1 {
		return b, errors.New("invalid request")
	}

	if idx+4 == len(b) {
		// request is not wrapped
		return b, nil
	}

	return b[idx+4:], nil
}
