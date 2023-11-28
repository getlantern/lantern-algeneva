package genevahttp

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/getlantern/algeneva"
)

// ErrMissingRequest is the error returned when a request is not found after unwrapping.
var ErrMissingRequest = errors.New("missing request")

// Conn is a wrapper around net.Conn that encrypts/decrypts the body of each request using AES.
type Conn struct {
	conn *algeneva.Conn
	key  []byte
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	buf := make([]byte, len(b))
	n, err = c.conn.Read(buf)
	if err != nil {
		return 0, err
	}

	_, req, fnd := bytes.Cut(buf, []byte("\r\n\r\n"))
	if !fnd || len(req) == 0 {
		return 0, ErrMissingRequest
	}

	if req, err = decrypt(req, c.key); err != nil {
		return 0, err
	}

	copy(b, req)
	return len(req), nil
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	headers, req, fnd := bytes.Cut(b, []byte("\r\n\r\n"))
	if !fnd || len(req) == 0 {
		return 0, ErrMissingRequest
	}

	if req, err = encrypt(req, c.key); err != nil {
		return 0, err
	}

	b = append(headers, req...)
	return c.conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address, if known.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address, if known.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
