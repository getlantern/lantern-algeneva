package genevahttp

import (
	"bytes"
	"net"
	"sync/atomic"

	"github.com/getlantern/algeneva"
)

// httpTransformConn is a wrapper around a net.conn. httpTransformConn will apply the geneva
// strategy, httpTransform, to the first request before writing it to the wrapped net.Conn.
// Subsequent requests are written directly to the wrapped net.Conn.
type httpTransformConn struct {
	// underlying connection
	net.Conn
	// httpTransformConn is the geneva strategy to apply to the first request.
	httpTransform *algeneva.HTTPStrategy
	// buf is a buffer to write the first request into until we can apply the geneva strategy. Once
	// all of the request header is writen to buf, we'll apply the geneva strategy and write the
	// transformed request to net.Conn.
	buf *bytes.Buffer
	// transformedFirst is a flag to indicate if the first request has been transformed.
	transformedFirst atomic.Bool
}

// Write writes data to the connection. If the first request has not been transformed and
// c.httpTransform is not nil, Write will buffer the data until all the request headers have been
// written. Once all the headers have been written, Write will apply the geneva strategy and write
// the transformed request to the underlying connection. Otherwise, Write will write the data to
// directly to the wrapped net.Conn as is.
func (c *httpTransformConn) Write(b []byte) (n int, err error) {
	if c.transformedFirst.Load() || c.httpTransform == nil {
		// The first request has been transformed, so we write directly to c.Conn.
		return c.Conn.Write(b)
	}

	// The first request has not been transformed, so we write to buf and check if we recieved all
	// of the request headers.
	if c.buf == nil {
		c.buf = &bytes.Buffer{}
	}

	c.buf.Write(b)
	if !bytes.Contains(b, []byte("\r\n\r\n")) {
		// We haven't recieved all of the headers yet, so we return early.
		return len(b), nil
	}

	req, err := c.httpTransform.Apply(c.buf.Bytes())
	if err != nil {
		return len(b), err
	}

	_, err = c.Conn.Write(req)
	if err != nil {
		return len(b), err
	}

	// The first request has been transformed, so we set transformedFirst to true and clear the
	// buffer.
	c.transformedFirst.Store(true)
	c.buf.Reset()
	c.buf = nil
	return len(b), nil
}

// normalizationConn is a wrapper around a net.conn. normalizationConn will attempt to normalize
// the first request read from the wrapped net.Conn.
//
// Important note: Depending on the strategy the client used to transform the request, the exact
// original request may not be recoverable. normalizationConn makes no guarantees about the
// original request and only guarantees that the request will be valid and well-formed.
type normalizationConn struct {
	// underlying connection
	net.Conn
	// buf will hold the normalized first request and calls to Read will read from buf until it is
	// empty.
	buf *bytes.Buffer
	// normalizedFirst is a flag to indicate if the first request has been normalized.
	normalizedFirst atomic.Bool
}

// Read reads data from the connection. If the first request has not been normalized, Read will
// attempt to normalize it. The first call to Read may take slightly longer than expected as it
// must read the entire first request before Read can normalizing it.
func (nc *normalizationConn) Read(b []byte) (n int, err error) {
	if nc.normalizedFirst.Load() {
		// The first request has been normalized, so we read from buf if it's not empty.
		if nc.buf.Len() > 0 {
			n, _ = nc.buf.Read(b)
		}

		if n < len(b) {
			// The caller requested more data than what's in buf, so we read off the underlying
			// connection if there's more data available.
			nn, err := nc.Conn.Read(b[n:])
			return n + nn, err
		}

		return n, err
	}

	if nc.buf == nil {
		nc.buf = &bytes.Buffer{}
	}

	// read the whole first request so we can normalize it.
	n, err = nc.readAvailable()
	if err != nil {
		n, _ = nc.buf.Read(b)
		return n, err
	}

	norm, err := algeneva.NormalizeRequest(nc.buf.Bytes()[:n])
	if err != nil {
		n, _ = nc.buf.Read(b)
		return n, err
	}

	nc.normalizedFirst.Store(true)

	// Clear the buffer so we can reuse it for storing the normalized request.
	nc.buf.Reset()
	nc.buf.Write(norm)
	n, _ = nc.buf.Read(b)
	return n, err
}

// readAvailable reads all available data from the connection and writes it to the buffer. Unlike
// other Read/copy methods that read until an io.EOF is recieved, readAvailable will read until no
// more data is available to be read. readAvailable returns the total number of bytes read and any
// error that occurred.
func (nc *normalizationConn) readAvailable() (int, error) {
	buf := make([]byte, 1024)
	for {
		n, err := nc.Conn.Read(buf)
		nc.buf.Write(buf[:n])
		if err != nil || n < 1024 {
			return nc.buf.Len(), err
		}
	}
}
