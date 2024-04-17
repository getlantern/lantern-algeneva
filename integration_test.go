package genevahttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

const (
	certPEM = `-----BEGIN CERTIFICATE-----
MIIDWTCCAkGgAwIBAgIUbHYEeCl2wo1zQrlJOfqSfGHruQkwDQYJKoZIhvcNAQEL
BQAwPDELMAkGA1UEBhMCVVMxDzANBgNVBAgMBk9yZWdvbjENMAsGA1UEBwwEVGVz
dDENMAsGA1UECgwEVGVzdDAeFw0yNDA0MTYxNzEzMzVaFw0zNDA0MTQxNzEzMzVa
MDwxCzAJBgNVBAYTAlVTMQ8wDQYDVQQIDAZPcmVnb24xDTALBgNVBAcMBFRlc3Qx
DTALBgNVBAoMBFRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCI
DViDsv0oqfNe8C/2OZv6WaxNaPd9ZEQ+Gm5ExxkxhcLoCKDKvKnKO9WUmpHOE/wm
dCRlC0AWjJfxadX1vSwOb40I1AZJvwFht4+jotXsXJdsyLzldkILdIPDAcGltJp2
kQl6si5FSSfvREnLkbJooUBEeRQPG8/USUGg9uoRuY5uuaTeGgREngzjuGAyI3O4
zrIPcitVC/cKwqAFwQrA5i2/Ax+JardkYmfpECmq351cGtq10w5/r2d7c6aC3TCu
FXlHUoLWC2FBEg6ENkcPWrVc/4T1qprQpZXHAWSz88dLeyCMnfqkLrj/fxvx8MH9
C1FR1/8NqAkdgqF5csCLAgMBAAGjUzBRMB0GA1UdDgQWBBTfBNWqfSYRYNvqk+3G
TIsqa1/7RjAfBgNVHSMEGDAWgBTfBNWqfSYRYNvqk+3GTIsqa1/7RjAPBgNVHRMB
Af8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQB8JpOcCGndN8IyzLkO1VGQrlGH
XbGefanmWXOV7XPoq09OdJGuf8mbkMARLi0MtoN3jJgJWICo4fDKAdEMw76X1RWp
Hh3GS23I6fKWW/AANsK+ctMZgwnkMWPOcKZosWR4yLdblTMp2wtxND0WINnLL4D7
8/BPT1d0Ikb5SUb55cMBeGY4KBT0F/ZSfPxKNuB24RHVYhRkH+3dn4Dzpf7Fq5eT
eVWrL9DTIm0gObJVqZuqrxgokIxo6KusjD602Zixph2G35L9TUbDZ3A8Cpl4fxx6
HxxRyPFc01MTjCIY+DS36pZNshz6wZhFIOrjDuMOm0ph7Ki7g/7c5axe/DWI
-----END CERTIFICATE-----`
	keyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCIDViDsv0oqfNe
8C/2OZv6WaxNaPd9ZEQ+Gm5ExxkxhcLoCKDKvKnKO9WUmpHOE/wmdCRlC0AWjJfx
adX1vSwOb40I1AZJvwFht4+jotXsXJdsyLzldkILdIPDAcGltJp2kQl6si5FSSfv
REnLkbJooUBEeRQPG8/USUGg9uoRuY5uuaTeGgREngzjuGAyI3O4zrIPcitVC/cK
wqAFwQrA5i2/Ax+JardkYmfpECmq351cGtq10w5/r2d7c6aC3TCuFXlHUoLWC2FB
Eg6ENkcPWrVc/4T1qprQpZXHAWSz88dLeyCMnfqkLrj/fxvx8MH9C1FR1/8NqAkd
gqF5csCLAgMBAAECggEAINDTQkTocif39zTI5MOFf0+k0zEXzOtj2HTolvdM+Nhy
KCR4oB38eDaRcBwOQh4o6h+GbcbWaPn1ZjnobTL5TuwSIQh/Eceb7jVn1Ijgv3ef
4JHUmiY5jOjIJT+ltTHINgQKvMkAhx67nqcig5L7bOhEB6AKuhAzw1j+FvSnhamZ
hbQFiB78hRu35yh79XEbUjPmFts9QCnuqtAOI+xCjvlgEhNS2UhmAzXwSNuYS4Ip
tCIC/rDimY/O5V8imo3Dx+Jf2lbfULLsNHDD6v+/3uSE/c0rdqGs85mfkroFBsYF
zaijBURNTMLilC4Xx9PzSjY7Sx5DV474m8Quegy1AQKBgQC/dEoHp5CFa1wUzCwt
g15qugOgO0ikEHUBvzF/baHkBwigX3eJP/3mf98MGrkYxM1gKLdvKa2Ko6NGtEWQ
dNZNh4bcMV2Us9QJSi/X9zzZrNo6xkV4Y0OkQS8+VHqdl08NoCeHAfUt76GDH9MP
j9KNMjlofuIzHL34I9VGAY1PAQKBgQC164DuQgC0FnIwZ0joEO8EGXRZgMUHkKdm
EGMotUxeGbtdPB7prB7beWSjF0aJZRBVLjXrKuol7imtHI8Jd+UNBRy6RgT3/LBa
YS/k2fx3RwuYLf35XjZJxeygPM6CPRKlgwbTQOcf2TzvaK17dPcNo3t6PXC5aLHF
NDFK8iPbiwKBgE5hvrk5ifqFhLJjEKclhG8vbrKX8tpwfmbTruEbsk7X7lkyHI9N
apaGvXuIKUWRtP9sTAUvzAPZkMwum9hTbTVaigT2FPj/UoznGYVSjFAV61ZqvCBY
i2Xg5gWfsn94Zf4PFn+4dndzBu3XBqL1X988s7IrWFJSrxe7G+LIWeEBAoGALajj
XmmghZLQrEdwLBb79rpw0noYedKbwWlBihkfBstMlJUfaSTzRcDNOoYABUIhfE+x
5smJpWWGflWZrRWznrX2xOYIHzoEBVs5SyZPUJy7U0HP6gP0ekW8I2e/qT6s7G/b
ibBTklHTEn/icwcjbv/mYQMExPR7EfUMnjPyPgsCgYBPS3FIH0FseVR4f7Sy/Ru/
tsQthtVVAtqmp7aHnWehhWOn8+v4Lrl7UqGu1oSkzX2eSz9oTTyjljej4vtcf0KI
CeST/HTEWkTNddorjhnZhk1hHN4T9FNr3X89bPZK4lfeOgtyaWQtBLnHInfT1rnr
v8YjBPMaZmsPom2vPUa9qQ==
-----END PRIVATE KEY-----`
)

var testMsgs = map[string]string{
	"up-up-down-down-left-right-left-right-b-a-start": "cheat activated",
	"void insert(s *char) { list[i++] = s; }":         "segfault",
}

func TestWebsocket(t *testing.T) {
	// Create a listener that will accept connections from the client.
	l, err := net.Listen("tcp", ":8080")
	require.NoError(t, err, "Failed to create listener")

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	require.NoError(t, err, "Failed to create tls keypair")

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	ll, _ := WrapListener(l, tlsConfig)
	require.NoError(t, err, "Failed to wrap listener")

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

	rootCertPool := x509.NewCertPool()
	require.True(t, rootCertPool.AppendCertsFromPEM([]byte(certPEM)))

	tlsConfig = &tls.Config{
		RootCAs:            rootCertPool,
		InsecureSkipVerify: true,
	}
	dialer := &mockDialer{}
	opts := DialerOpts{
		AlgenevaStrategy: algeneva.Strategies["China"][17],
		Dialer:           dialer,
		TLSConfig:        tlsConfig,
	}
	c, err := DialContext(ctx, "tcp", "localhost:8080", opts)
	require.NoError(t, err, "Failed to dial")
	defer c.Close()

	assert.True(t, dialer.used, "mockDialer was not used")

	require.IsType(t, &tls.Conn{}, c, "Dial returned a non-tls.Conn")
	tlsConn := c.(*tls.Conn)
	connState := tlsConn.ConnectionState()
	require.True(t, connState.HandshakeComplete, "TLS handshake failed")

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

type mockDialer struct {
	used bool
}

func (m *mockDialer) Dial(network, addr string) (net.Conn, error) {
	return m.DialContext(context.Background(), network, addr)
}

func (m *mockDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	m.used = true
	return (&net.Dialer{}).DialContext(ctx, network, addr)
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
