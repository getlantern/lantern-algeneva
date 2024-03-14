package genevahttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/getlantern/algeneva"
)

// httpTransformConn is a wrapper around a net.conn. httpTransformConn will apply the geneva
// strategy, httpTransform, to the first request before writing it to the wrapped net.Conn.
// Subsequent requests are written directly to the wrapped net.Conn.
type httpTransformConn struct {
	// Wrapped connection
	net.Conn
	// httpTransformConn is the geneva strategy to apply to the first request.
	httpTransform *algeneva.HTTPStrategy
	// buf is a buffer to write the first request into until we can apply the geneva strategy. Once
	// all of the request header is writen to buf, we'll apply the geneva strategy and write the
	// transformed request to net.Conn.
	buf *bytes.Buffer
	// eohCheckPtr is the index in the buffer where we last checked for the end of the headers. We
	// use this to avoid rechecking the entire buffer for the end of the headers on each write
	eohCheckPtr int
	// transformedFirst is a flag to indicate if the first request has been transformed.
	transformedFirst bool
}

// Write writes data to the connection. If the first request has not been transformed and
// c.httpTransform is not nil, Write will buffer the data until all the request headers have been
// written. Once all the headers have been written, Write will apply the geneva strategy and write
// the transformed request to the wrapped connection. Otherwise, Write will write the data directly
// to the wrapped net.Conn as is.
func (c *httpTransformConn) Write(b []byte) (n int, err error) {
	if c.transformedFirst || c.httpTransform == nil || len(b) == 0 {
		// The first request has been transformed, or the caller didn't pass any data to write, so we
		// just forward b to Conn.
		return c.Conn.Write(b)
	}

	// The first request has not been transformed, so we write to buf and check if we recieved all
	// of the request headers.
	if c.buf == nil {
		c.buf = &bytes.Buffer{}
	}

	c.buf.Write(b)
	// We need to check if we've recieved all of the headers before we can apply the geneva
	// strategy. Since the headers are terminated by a string and not just one byte, we need to
	// check c.buf, as '\r\n\r\n' may be split between two writes.
	if !bytes.Contains(c.buf.Bytes()[c.eohCheckPtr:], []byte("\r\n\r\n")) {
		// We haven't recieved all of the headers yet, so update eohCheckPtr to the end of the buffer
		// but back up 3 bytes in case some of the token was written already.
		c.eohCheckPtr += len(b) - 3
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
	c.transformedFirst = true
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
	// wrapped connection
	net.Conn
	// buf will hold the normalized first request and calls to Read will read from buf until it is
	// empty.
	buf *bytes.Buffer
	// normalizedFirst is a flag to indicate if the first request has been normalized.
	normalizedFirst bool
}

// Read reads data from the connection. If the first request has not been normalized, Read will
// attempt to normalize it. The first call to Read may take slightly longer than expected as it
// must read at least the request-line and headers to normalize the request.
func (nc *normalizationConn) Read(b []byte) (n int, err error) {
	if nc.normalizedFirst {
		// The first request has been normalized, so we read from buf if it's not empty.
		if nc.buf.Len() > 0 {
			return nc.buf.Read(b)
		}

		return nc.Conn.Read(b)
	}

	if nc.buf == nil {
		nc.buf = &bytes.Buffer{}
	}

	// We don't need the whole request to normalize it, just the request-line and headers.
	n, err = readAtLeastUntil(nc.Conn, nc.buf, []byte("\r\n\r\n"))
	if err != nil {
		return 0, err
	}

	norm, err := algeneva.NormalizeRequest(nc.buf.Bytes()[:n])
	if err != nil {
		return 0, err
	}

	nc.normalizedFirst = true

	// Clear the buffer so we can reuse it for storing the normalized request.
	nc.buf.Reset()
	nc.buf.Write(norm)
	// we can ignore the error here since bytes.Buffer.Read will only return an error if the buffer
	//	is empty, which we just wrote to.
	n, _ = nc.buf.Read(b)
	return n, nil
}

// readAtLeastUntil reads from the provided src Reader until it encounters the specified token,
// writing the read data to dst. readAtLeastUntil reads and writes in chunks, so dst will also
// contain all data following token from the last read. If an io.EOF is encountered and the token
// is found, a nil error is returned and the number of bytes written to dst. Otherwise, the first
// error encountered will be returned and the number of bytes written to dst up to the point of
// the error.
func readAtLeastUntil(src io.Reader, dst io.Writer, token []byte) (int, error) {
	var (
		// buf is the buffer used for reading data from src.
		buf = make([]byte, 1024)
		// wptr is the index in buf where we should start writing the next read. We copy the last
		// len(token) bytes of the previous read to the beginning of buf so we can account for an
		// edge case where the token is split between two reads.
		wptr int
		// written is the total number of bytes written to dst.
		written int
	)
	for {
		// Read data from src into buf starting at wptr.
		nr, er := src.Read(buf[wptr:])
		if nr > 0 {
			nw, ew := dst.Write(buf[wptr : wptr+nr])
			written += nw
			wptr += nw

			switch {
			case er != nil && er != io.EOF:
				// Error encountered while reading from src and it's not EOF.
				return written, fmt.Errorf("error reading from src: %w", er)
			case ew != nil:
				// Error encountered while writing to dst.
				return written, fmt.Errorf("error writing to dst: %w", ew)
			case nr != nw:
				// special case where we didn't write all of the data to dst but no error was returned.
				return written, errors.New("failed to write all data to dst")
			case bytes.Contains(buf[:wptr], token):
				// Token found in the read data.
				return written, nil
			}

			// Shift the last len(token) bytes to the beginning of buf in case the token was split
			// between two reads.
			j := max(wptr-len(token), 0)
			wptr = copy(buf[:len(token)], buf[j:wptr])
		}

		if er != nil {
			if er == io.EOF {
				// We reached the end of the src and the token was not found.
				return written, fmt.Errorf("token not found: %w", io.EOF)
			}

			return written, er
		}
	}
}
