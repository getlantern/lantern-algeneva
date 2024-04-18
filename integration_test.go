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
MIIC7TCCAdWgAwIBAgIUPdNVO5iYFFIOZkNR7PP2M+Vn4fswDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI0MDQxODE5MDQ1NVoXDTM0MDQx
NjE5MDQ1NVowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEArGzK/NbZkfJzKAMryAiDJwqYMmZUL1okHDshfAD1ECMP
4yCq3DouIXBGNY0s9Y11j/N4qzToCfMWfsnTT0l8nyWbHuAlcMPGC8pFzCr4ZrsU
q3QgvLM0czqv58TD8XO0DvbgsHs3EFB6kZEyLVzp4UThveUNcUFMz9+AskegKtJx
/s/N9xHh50aHTKeiyZ6sbL6O61Ojthc7wfwUrlHsKY9kInv/lyriLSaz6agwZR7P
zgSRP06SFkzOV6wsqAEOyEroeFjjyw7xD6AeRqp4HRIFNb3jh25bTdQ5xBHv6rpi
Q+VMQE8wXNFHSbBpZGi1A0hMH4MDNFniy+KhV8bHSQIDAQABozcwNTAUBgNVHREE
DTALgglsb2NhbGhvc3QwHQYDVR0OBBYEFG76xcV0Ggi/GwSxHzHmosDoy/UtMA0G
CSqGSIb3DQEBCwUAA4IBAQCVTNmUoca7p+bu9F8VguFZOlDcXDAw7k2ihbR4IHVZ
Jahubgo5sktkxZ7tbJJpN3IFQR1ItDaTqFNHL8pruKUwCqOoIkaEa7tFeVZx1JbW
VfpwyDS1eqokBB6W2O7cIi7dOnH12YfLXU0Tm4l3Mn4Nh7RZqzixm0frH7ntLzw2
ioVad2EEYqB8DwCZrmEmA5tiDgqt0zBonynmlj+5V85mHlGVbVMsYr9f5ZKOi7Xl
dxwxX30SquVfeIxwMUCudM6TQ6DVPYjTaqK32iSkpuWi03i9jmWoQIqPwk6qaja+
WufWjiCRIm/9mSnXKSTzp0uQ+L09bcIbQ4ELGjSxu2BN
-----END CERTIFICATE-----`
	keyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCsbMr81tmR8nMo
AyvICIMnCpgyZlQvWiQcOyF8APUQIw/jIKrcOi4hcEY1jSz1jXWP83irNOgJ8xZ+
ydNPSXyfJZse4CVww8YLykXMKvhmuxSrdCC8szRzOq/nxMPxc7QO9uCwezcQUHqR
kTItXOnhROG95Q1xQUzP34CyR6Aq0nH+z833EeHnRodMp6LJnqxsvo7rU6O2FzvB
/BSuUewpj2Qie/+XKuItJrPpqDBlHs/OBJE/TpIWTM5XrCyoAQ7ISuh4WOPLDvEP
oB5GqngdEgU1veOHbltN1DnEEe/qumJD5UxATzBc0UdJsGlkaLUDSEwfgwM0WeLL
4qFXxsdJAgMBAAECggEAEBpO8ML96bfrUkLNjW5iFTzTju1okk2ITsyk6WhLerjT
jIIqAsw6L6xFGk43cy1FW+7Ah7i2rOszYB7oKaDyzwgbjwwe4wOdlM9Mqm8e6LUz
DnoXbpgL33ENKYeCRyPnnngm7sRrFY52i+6z8XGadAvTS0E/eqK/EjDM25l911HN
zlKcvbePI0z2APUsWGl/13Qd7ks+WDytg1efFJCOPbiYYM5vpKfy9bzf2RQ7IRsr
5EPyHhOyFVd6UrzvkvjBvaouygM7u6CTWkYd3Z8ARxRdGPJN7iMzTZFDAx03Eq3A
gf29ffbHN8k70973X6KH32LUd1NZvrGTlFGeH+WLgQKBgQDTdP5j4ckQ5oQwMqM4
k9Ihx/ILGIFiG1WUUwLtSv3IN0YeyNpR1QBZwP+zSoRIZQkbKuFII+xxBa2RTrFu
MyRTf7rAK/R0AYP/htt9TdFWHssXDg/fnGqUVfJwjNMYZqwcpKND4eDynB3ThdKV
Of8uWbpttIjScfAMCEtkepTuYQKBgQDQvvZS4GHQXErplPrtsEt5jz0uCnGGvqno
LihA/8Ewy+rHxQzXOclYBW7bZdr5B9S/uVVrchkHx3bz1FRtnATgUwCG0YaOk9ho
Q/brZIP6W/Ei7WCkIy62+rPbZQlBDIXDTQ7mexIRj2kJsDdz+ybwDExlN/o1W8c/
UGmw3qFx6QJ/Fk1Ah0hI7H9jcbHlhRISF33/CSyMeMxpOjuHE3/VREiQHK8SV48f
elfgoAg762a8jyD2oaUoSsNOiwTBsd2y9xuBlsMMBTAju899VrneWjblNIlHI05b
70khSL2RhgFOJbc3gPFRyESu4KA8lYCIaVsNToS76XYa2yoEyZQkIQKBgQCGk4Zk
eco1tTqKioSXdj/CV8k+hHcaQpNxX0iOVxQqrFxpfC1CGDwpJh+JDIp2YEkVbZuX
UJC4hiy3F51yqNIv+PLu9+fCxagP2Dk5Gq1HW70DInxadWApkUkg2Wt052jZNzWy
+4bzkTxLhbLKcBFzUspxuvvxKIE03Ve2MmFs+QKBgQChvuTCmiHoO7el2HfR8NOZ
phEh0yWTALHVnSJOcrxlB30NrSC25v7AWQ08tkPlxr6Znrs8UPS/klFDBJKc5S9V
IlLr8ERn4mNnlqeIjsKLuB6LSGZZvAzaomNJ6oEdEq0MLCletAheWiJNt+20zwVB
G02fP/oUEPDTYjlf6xAS1g==
-----END PRIVATE KEY-----`
)

var testMsgs = map[string]string{
	"up-up-down-down-left-right-left-right-b-a-start": "cheat activated",
	"void insert(s *char) { list[i++] = s; }":         "segfault",
}

func TestWebsocket(t *testing.T) {
	// Create a listener that will accept connections from the client.
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to create listener")

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	require.NoError(t, err, "Failed to create tls keypair")

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	ll, _ := WrapListener(l, tlsConfig)

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
		RootCAs:    rootCertPool,
		ServerName: "localhost",
	}
	dialer := &mockDialer{}
	opts := DialerOpts{
		AlgenevaStrategy: algeneva.Strategies["China"][17],
		Dialer:           dialer,
		TLSConfig:        tlsConfig,
	}
	c, err := DialContext(ctx, "tcp", l.Addr().String(), opts)
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
