package bootstrap

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxyProtocolListenerDisabled(t *testing.T) {
	listener, _ := newProxyProtocolTestConnection(t, "192.0.2.10")

	wrapped, err := newProxyProtocolListener(listener, nil)

	require.NoError(t, err)
	assert.Same(t, listener, wrapped)
}

func TestNewProxyProtocolListenerRequiresTCP(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})
	listener := &singleConnListener{
		conn:   serverConn,
		addr:   &net.UnixAddr{Name: "/tmp/pocket-id.sock", Net: "unix"},
		closed: make(chan struct{}),
	}

	_, err := newProxyProtocolListener(listener, []string{"127.0.0.1"})

	require.Error(t, err)
	assert.ErrorContains(t, err, "PROXY protocol requires a TCP listener")
}

func TestNewProxyProtocolListenerUsesStrictTrustedProxyPolicy(t *testing.T) {
	listener, _ := newProxyProtocolTestConnection(t, "192.0.2.10")

	wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)
	proxyListener := requireProxyProtocolListener(t, wrapped)
	assert.Equal(t, proxyProtocolReadHeaderTimeout, proxyListener.ReadHeaderTimeout)

	policy, err := proxyListener.ConnPolicy(proxyproto.ConnPolicyOptions{
		Upstream: &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 12345},
	})
	require.NoError(t, err)
	assert.Equal(t, proxyproto.REQUIRE, policy)

	_, err = proxyListener.ConnPolicy(proxyproto.ConnPolicyOptions{
		Upstream: &net.TCPAddr{IP: net.ParseIP("198.51.100.10"), Port: 12345},
	})
	require.ErrorIs(t, err, proxyproto.ErrInvalidUpstream)

	policy, err = proxyListener.ConnPolicy(proxyproto.ConnPolicyOptions{
		Upstream: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	})
	require.NoError(t, err)
	assert.Equal(t, proxyproto.SKIP, policy)
}

func TestProxyProtocolListenerAllowsLoopbackWithoutHeader(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "127.0.0.1")
	wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)

	serverConn, err := wrapped.Accept()
	require.NoError(t, err)

	writeErr := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte("healthcheck"))
		writeErr <- err
	}()

	payload := make([]byte, len("healthcheck"))
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	require.NoError(t, <-writeErr)
	assert.Equal(t, "healthcheck", string(payload))
	assert.Equal(t, "127.0.0.1:12345", serverConn.RemoteAddr().String())
}

func TestProxyProtocolListenerAcceptsHeaderFromTrustedLoopback(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "::1")
	wrapped, err := newProxyProtocolListener(listener, []string{"::1"})
	require.NoError(t, err)

	serverConn, err := wrapped.Accept()
	require.NoError(t, err)

	source := &net.TCPAddr{IP: net.ParseIP("2001:db8::15"), Port: 42300}
	dest := &net.TCPAddr{IP: net.ParseIP("::1"), Port: 1411}
	writeErr := make(chan error, 1)
	go func() {
		header := proxyproto.HeaderProxyFromAddrs(2, source, dest)
		if _, err := header.WriteTo(clientConn); err != nil {
			writeErr <- err
			return
		}
		_, err := clientConn.Write([]byte("payload"))
		writeErr <- err
	}()

	payload := make([]byte, len("payload"))
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	require.NoError(t, <-writeErr)
	assert.Equal(t, "payload", string(payload))
	assert.Equal(t, source.String(), serverConn.RemoteAddr().String())
}

func TestProxyProtocolListenerAllowsTrustedLoopbackWithoutHeader(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "::1")
	wrapped, err := newProxyProtocolListener(listener, []string{"::1"})
	require.NoError(t, err)

	serverConn, err := wrapped.Accept()
	require.NoError(t, err)

	writeErr := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte("healthcheck"))
		writeErr <- err
	}()

	payload := make([]byte, len("healthcheck"))
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	require.NoError(t, <-writeErr)
	assert.Equal(t, "healthcheck", string(payload))
	assert.Equal(t, "[::1]:12345", serverConn.RemoteAddr().String())
}

func TestProxyProtocolListenerDropsUntrustedPeerAndContinues(t *testing.T) {
	untrustedServerConn, untrustedClientConn := newAddressedPipe("198.51.100.10")
	trustedServerConn, trustedClientConn := newAddressedPipe("192.0.2.10")
	listener := &queuedConnListener{
		conns: []net.Conn{untrustedServerConn, trustedServerConn},
		addr:  trustedServerConn.LocalAddr(),
	}
	t.Cleanup(func() {
		_ = listener.Close()
		_ = untrustedServerConn.Close()
		_ = trustedServerConn.Close()
		_ = untrustedClientConn.Close()
		_ = trustedClientConn.Close()
	})

	wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)
	serverConn, err := wrapped.Accept()
	require.NoError(t, err)

	source := &net.TCPAddr{IP: net.ParseIP("203.0.113.15"), Port: 42300}
	dest := &net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 443}
	writeErr := make(chan error, 1)
	go func() {
		header := proxyproto.HeaderProxyFromAddrs(1, source, dest)
		if _, err := header.WriteTo(trustedClientConn); err != nil {
			writeErr <- err
			return
		}
		_, err := trustedClientConn.Write([]byte("payload"))
		writeErr <- err
	}()

	payload := make([]byte, len("payload"))
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	require.NoError(t, <-writeErr)
	assert.Equal(t, source.String(), serverConn.RemoteAddr().String())

	_, err = untrustedClientConn.Write([]byte("payload"))
	require.Error(t, err)
}

func TestProxyProtocolListenerExtractsClientAddress(t *testing.T) {
	testCases := []struct {
		name    string
		version byte
		source  *net.TCPAddr
		dest    *net.TCPAddr
	}{
		{
			name:    "version 1 IPv4",
			version: 1,
			source:  &net.TCPAddr{IP: net.ParseIP("203.0.113.15"), Port: 42300},
			dest:    &net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 443},
		},
		{
			name:    "version 2 IPv6",
			version: 2,
			source:  &net.TCPAddr{IP: net.ParseIP("2001:db8::15"), Port: 42300},
			dest:    &net.TCPAddr{IP: net.ParseIP("2001:db8::20"), Port: 443},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			listener, clientConn := newProxyProtocolTestConnection(t, "192.0.2.10")
			wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
			require.NoError(t, err)

			serverConn, err := wrapped.Accept()
			require.NoError(t, err)

			writeErr := make(chan error, 1)
			go func() {
				header := proxyproto.HeaderProxyFromAddrs(testCase.version, testCase.source, testCase.dest)
				if _, err := header.WriteTo(clientConn); err != nil {
					writeErr <- err
					return
				}
				_, err := clientConn.Write([]byte("payload"))
				writeErr <- err
			}()

			payload := make([]byte, len("payload"))
			_, err = io.ReadFull(serverConn, payload)
			require.NoError(t, err)
			require.NoError(t, <-writeErr)
			assert.Equal(t, "payload", string(payload))
			assert.Equal(t, testCase.source.String(), serverConn.RemoteAddr().String())
			assert.Equal(t, testCase.dest.String(), serverConn.LocalAddr().String())
		})
	}
}

func TestProxyProtocolListenerRejectsMissingHeader(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "192.0.2.10")
	wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)

	serverConn, err := wrapped.Accept()
	require.NoError(t, err)

	go func() {
		_, _ = clientConn.Write([]byte("GET /healthz HTTP/1.1\r\n"))
	}()

	_, err = serverConn.Read(make([]byte, 1))
	require.ErrorIs(t, err, proxyproto.ErrNoProxyProtocol)
}

func TestProxyProtocolListenerSetsGinClientIP(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "192.0.2.10")
	wrapped, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)

	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	clientIP := make(chan string, 1)
	engine.GET("/client-ip", func(c *gin.Context) {
		clientIP <- c.ClientIP()
		c.Status(http.StatusNoContent)
	})
	server := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: time.Second,
	}
	t.Cleanup(func() {
		_ = server.Close()
	})
	go func() {
		_ = server.Serve(wrapped)
	}()

	source := &net.TCPAddr{IP: net.ParseIP("203.0.113.15"), Port: 42300}
	dest := &net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 443}
	writeErr := make(chan error, 1)
	go func() {
		header := proxyproto.HeaderProxyFromAddrs(1, source, dest)
		if _, err := header.WriteTo(clientConn); err != nil {
			writeErr <- err
			return
		}
		_, err := clientConn.Write([]byte("GET /client-ip HTTP/1.1\r\nHost: pocket-id.example\r\nConnection: close\r\n\r\n"))
		writeErr <- err
	}()

	select {
	case actualClientIP := <-clientIP:
		assert.Equal(t, "203.0.113.15", actualClientIP)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Gin to handle the proxied request")
	}
	require.NoError(t, <-writeErr)
}

func TestProxyProtocolHeaderIsReadBeforeTLS(t *testing.T) {
	listener, clientConn := newProxyProtocolTestConnection(t, "192.0.2.10")
	proxyListener, err := newProxyProtocolListener(listener, []string{"192.0.2.0/24"})
	require.NoError(t, err)

	serverTLSConfig, clientTLSConfig := newProxyProtocolTestTLSConfigs(t)
	tlsListener := tls.NewListener(proxyListener, serverTLSConfig)
	serverConn, err := tlsListener.Accept()
	require.NoError(t, err)

	source := &net.TCPAddr{IP: net.ParseIP("203.0.113.15"), Port: 42300}
	dest := &net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 443}
	ctx := t.Context()
	clientErr := make(chan error, 1)
	go func() {
		header := proxyproto.HeaderProxyFromAddrs(2, source, dest)
		if _, err := header.WriteTo(clientConn); err != nil {
			clientErr <- err
			return
		}

		tlsClient := tls.Client(clientConn, clientTLSConfig)
		if err := tlsClient.HandshakeContext(ctx); err != nil {
			clientErr <- err
			return
		}
		_, err := tlsClient.Write([]byte("payload"))
		clientErr <- err
	}()

	tlsServerConn, ok := serverConn.(*tls.Conn)
	require.True(t, ok)
	require.NoError(t, tlsServerConn.HandshakeContext(ctx))
	payload := make([]byte, len("payload"))
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	require.NoError(t, <-clientErr)
	assert.Equal(t, "payload", string(payload))
	assert.Equal(t, source.String(), serverConn.RemoteAddr().String())
}

func requireProxyProtocolListener(t *testing.T, listener net.Listener) *proxyproto.Listener {
	t.Helper()

	proxyListener, ok := listener.(*proxyproto.Listener)
	require.True(t, ok)
	return proxyListener
}

func newProxyProtocolTestConnection(t *testing.T, upstreamIP string) (*singleConnListener, net.Conn) {
	t.Helper()

	serverConn, clientConn := newAddressedPipe(upstreamIP)
	listener := &singleConnListener{
		conn:   serverConn,
		addr:   serverConn.LocalAddr(),
		closed: make(chan struct{}),
	}

	t.Cleanup(func() {
		_ = listener.Close()
		_ = clientConn.Close()
	})
	return listener, clientConn
}

func newAddressedPipe(upstreamIP string) (net.Conn, net.Conn) {
	serverConn, clientConn := net.Pipe()
	return &addressedConn{
		Conn:       serverConn,
		localAddr:  &net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 1411},
		remoteAddr: &net.TCPAddr{IP: net.ParseIP(upstreamIP), Port: 12345},
	}, clientConn
}

func newProxyProtocolTestTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(certificateDER)
	require.NoError(t, err)

	roots := x509.NewCertPool()
	roots.AddCert(certificate)

	serverConfig := &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certificateDER},
			PrivateKey:  privateKey,
		}},
		MinVersion: tls.VersionTLS13,
	}
	clientConfig := &tls.Config{
		RootCAs:    roots,
		ServerName: "localhost",
		MinVersion: tls.VersionTLS13,
	}
	return serverConfig, clientConfig
}

type singleConnListener struct {
	conn      net.Conn
	addr      net.Addr
	accepted  bool
	closed    chan struct{}
	closeOnce sync.Once
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.accepted {
		<-l.closed
		return nil, net.ErrClosed
	}
	l.accepted = true
	return l.conn, nil
}

func (l *singleConnListener) Close() error {
	var closeErr error
	l.closeOnce.Do(func() {
		close(l.closed)
		closeErr = l.conn.Close()
	})
	return closeErr
}

func (l *singleConnListener) Addr() net.Addr {
	return l.addr
}

type addressedConn struct {
	net.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *addressedConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *addressedConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

type queuedConnListener struct {
	conns []net.Conn
	addr  net.Addr
}

func (l *queuedConnListener) Accept() (net.Conn, error) {
	if len(l.conns) == 0 {
		return nil, net.ErrClosed
	}

	conn := l.conns[0]
	l.conns = l.conns[1:]
	return conn, nil
}

func (l *queuedConnListener) Close() error {
	for _, conn := range l.conns {
		_ = conn.Close()
	}
	return nil
}

func (l *queuedConnListener) Addr() net.Addr {
	return l.addr
}
