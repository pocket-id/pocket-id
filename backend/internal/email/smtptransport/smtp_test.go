//go:build unit

package smtptransport

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	emaillib "github.com/italypaleale/go-kit/emailer"
)

func TestInit(t *testing.T) {
	// Use a full connection string so Init validates the same fields production code depends on
	connString, err := url.Parse("smtp://mailer:secret@mail.example.com:2525?fromAddress=sender@example.com&fromName=Sender+Name&tls=none")
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	// Verify the normalized SMTP configuration is preserved for the later send path
	assert.Equal(t, "mail.example.com", emailer.host)
	assert.Equal(t, "2525", emailer.port)
	assert.Equal(t, "mail.example.com:2525", emailer.address)
	assert.Equal(t, "mailer", emailer.username)
	assert.Equal(t, "secret", emailer.password)
	assert.Equal(t, "\"Sender Name\" <sender@example.com>", emailer.from)
	assert.Equal(t, "sender@example.com", emailer.fromAddress)
	hostname, err := os.Hostname()
	require.NoError(t, err)
	assert.Equal(t, hostname, emailer.localName)
	assert.Equal(t, smtpTLSNone, emailer.tlsMode)
	require.NotNil(t, emailer.tlsConfig)
	require.NotNil(t, emailer.dialContext)
}

func TestInitInsecureSkipVerify(t *testing.T) {
	t.Run("enabled sets InsecureSkipVerify on the TLS config", func(t *testing.T) {
		connString, err := url.Parse("smtp://mail.example.com:2525?fromAddress=sender@example.com&tls=implicit&insecureSkipVerify=true")
		require.NoError(t, err)

		var emailer SMTPEmailer
		err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
		require.NoError(t, err)
		require.NotNil(t, emailer.tlsConfig)
		assert.True(t, emailer.tlsConfig.InsecureSkipVerify)
	})

	t.Run("defaults to certificate verification enabled", func(t *testing.T) {
		connString, err := url.Parse("smtp://mail.example.com:2525?fromAddress=sender@example.com&tls=implicit")
		require.NoError(t, err)

		var emailer SMTPEmailer
		err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
		require.NoError(t, err)
		require.NotNil(t, emailer.tlsConfig)
		assert.False(t, emailer.tlsConfig.InsecureSkipVerify)
	})
}

func TestBuildMessageHeaders(t *testing.T) {
	// Pin the clock so the Date header is deterministic and drive buildMessage directly to avoid networking
	fixedTime := time.Date(2026, 7, 4, 15, 4, 5, 0, time.UTC)
	emailer := SMTPEmailer{
		from:        "\"Sender Name\" <sender@example.com>",
		fromAddress: "sender@example.com",
		now:         func() time.Time { return fixedTime },
	}

	payload, err := emailer.buildMessage(emaillib.EmailAddress{Name: "Recipient Name", Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
	require.NoError(t, err)

	message := string(payload)
	assert.Contains(t, message, "From: \"Sender Name\" <sender@example.com>\r\n")
	assert.Contains(t, message, "To: \"Recipient Name\" <recipient@example.com>\r\n")
	assert.Contains(t, message, "Date: "+fixedTime.Format(time.RFC1123Z)+"\r\n")
	// The Message-ID uses random bytes but is always scoped to the sender's domain
	assert.Regexp(t, `Message-ID: <[0-9a-f]{32}@example\.com>\r\n`, message)
}

func TestSendEmailUsesLoginAuthWhenServerOffersLogin(t *testing.T) {
	// A server that advertises only AUTH LOGIN must trigger the LOGIN fallback instead of PLAIN
	server := newSMTPTestServer(t, "LOGIN")
	host, port, err := net.SplitHostPort(server.address())
	require.NoError(t, err)

	connString, err := url.Parse(fmt.Sprintf("smtp://mailer:secret@%s:%s?fromAddress=sender@example.com&tls=none", host, port))
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	err = emailer.SendEmail(t.Context(), emaillib.EmailAddress{Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
	require.NoError(t, err)

	// Confirm the client authenticated with LOGIN and sent the credentials the server decoded
	session, sessionErr := server.wait()
	require.NoError(t, sessionErr)
	assert.Equal(t, "AUTH LOGIN", session.authCommand)
	assert.Equal(t, "mailer", session.authUsername)
	assert.Equal(t, "secret", session.authPassword)
}

func TestSendEmail(t *testing.T) {
	// Start a local SMTP server so the test can verify the wire-level SMTP session and MIME payload
	server := newSMTPTestServer(t, "PLAIN")
	host, port, err := net.SplitHostPort(server.address())
	require.NoError(t, err)

	connString, err := url.Parse(fmt.Sprintf("smtp://mailer:secret@%s:%s?fromAddress=sender@example.com&fromName=Sender+Name&tls=none", host, port))
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	// Send a multipart message so the test covers auth, envelope, and MIME body generation together
	err = emailer.SendEmail(t.Context(), emaillib.EmailAddress{Name: "Recipient Name", Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{
		Text: "Plain body",
		HTML: "<p>HTML body</p>",
	})
	require.NoError(t, err)

	// Validate the full SMTP session after the handler goroutine has finished processing the connection
	session, sessionErr := server.wait()
	require.NoError(t, sessionErr)
	assert.Contains(t, session.authCommand, "AUTH PLAIN")
	assert.Equal(t, "<sender@example.com>", session.mailFrom)
	assert.Equal(t, "<recipient@example.com>", session.rcptTo)
	assert.Contains(t, session.message, "From: \"Sender Name\" <sender@example.com>\r\n")
	// The recipient display name must appear in the To header while the envelope keeps the bare address
	assert.Contains(t, session.message, "To: \"Recipient Name\" <recipient@example.com>\r\n")
	assert.Contains(t, session.message, "Subject: Hello\r\n")
	// A Date header and a Message-ID scoped to the sender domain are emitted so relays and spam filters see a complete message
	assert.Contains(t, session.message, "Date: ")
	assert.Regexp(t, `Message-ID: <[0-9a-f]{32}@example\.com>\r\n`, session.message)
	assert.Contains(t, session.message, "Content-Type: multipart/alternative; boundary=")
	assert.Contains(t, session.message, "Content-Type: text/plain; charset=UTF-8")
	assert.Contains(t, session.message, "Content-Type: text/html; charset=UTF-8")
	assert.Contains(t, session.message, "Plain body")
	assert.Contains(t, session.message, "<p>HTML body</p>")
	assert.Contains(t, session.commands, "MAIL FROM:<sender@example.com>")
	assert.Contains(t, session.commands, "RCPT TO:<recipient@example.com>")
}

func TestSendEmailSetsHostnameBeforeStartTLS(t *testing.T) {
	// Reject the first greeting after recording it so the test covers the pre-STARTTLS command without needing a TLS certificate
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	type probeResult struct {
		command string
		err     error
	}
	resultCh := make(chan probeResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			resultCh <- probeResult{err: acceptErr}
			return
		}
		defer func() { _ = conn.Close() }()

		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)
		responseErr := writeSMTPResponse(writer, "220 localhost ESMTP test")
		if responseErr != nil {
			resultCh <- probeResult{err: responseErr}
			return
		}

		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			resultCh <- probeResult{err: readErr}
			return
		}
		responseErr = writeSMTPResponse(writer, "421 greeting rejected")
		if responseErr != nil {
			resultCh <- probeResult{err: responseErr}
			return
		}

		resultCh <- probeResult{command: strings.TrimRight(line, "\r\n")}
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	connString, err := url.Parse(fmt.Sprintf("smtp://%s:%s?fromAddress=sender@example.com&tls=starttls", host, port))
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)
	emailer.localName = "pocket-id.example.test"

	err = emailer.SendEmail(t.Context(), emaillib.EmailAddress{Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
	require.ErrorContains(t, err, "failed to send SMTP greeting")

	var result probeResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP greeting probe timed out")
	}
	require.NoError(t, result.err)
	assert.Equal(t, "EHLO pocket-id.example.test", result.command)
}

func TestInitRejectsHeaderInjection(t *testing.T) {
	t.Run("CRLF in from address is rejected", func(t *testing.T) {
		// %0d%0a decodes to CR/LF in the query value, simulating an attacker-controlled fromAddress
		connString, err := url.Parse("smtp://mail.example.com:2525?fromAddress=sender@example.com%0d%0aBcc:+attacker@evil.example&tls=none")
		require.NoError(t, err)

		var emailer SMTPEmailer
		err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
		require.ErrorContains(t, err, "from address")
		require.ErrorContains(t, err, "must not contain CR or LF")
	})

	t.Run("CRLF in from name is rejected", func(t *testing.T) {
		connString, err := url.Parse("smtp://mail.example.com:2525?fromAddress=sender@example.com&fromName=Joe%0d%0aBcc:+attacker@evil.example&tls=none")
		require.NoError(t, err)

		var emailer SMTPEmailer
		err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
		require.ErrorContains(t, err, "from name")
		require.ErrorContains(t, err, "must not contain CR or LF")
	})
}

func TestSendEmailRejectsHeaderInjection(t *testing.T) {
	// Configure a valid emailer
	// BuildMessage validates before any network dial, so no server is needed
	connString, err := url.Parse("smtp://mail.example.com:2525?fromAddress=sender@example.com&tls=none")
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	t.Run("CRLF in recipient is rejected", func(t *testing.T) {
		err := emailer.SendEmail(t.Context(), emaillib.EmailAddress{Address: "victim@example.com\r\nBcc: attacker@evil.example"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
		require.ErrorContains(t, err, "recipient address")
		require.ErrorContains(t, err, "must not contain CR or LF")
	})

	t.Run("CRLF in recipient name is rejected", func(t *testing.T) {
		err := emailer.SendEmail(t.Context(), emaillib.EmailAddress{Name: "Victim\r\nBcc: attacker@evil.example", Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
		require.ErrorContains(t, err, "recipient name")
		require.ErrorContains(t, err, "must not contain CR or LF")
	})

	t.Run("bare LF in subject is rejected", func(t *testing.T) {
		err := emailer.SendEmail(t.Context(), emaillib.EmailAddress{Address: "recipient@example.com"}, "Hello\nBcc: attacker@evil.example", emaillib.SendEmailMessage{Text: "Body"})
		require.ErrorContains(t, err, "subject")
		require.ErrorContains(t, err, "must not contain CR or LF")
	})
}

func TestSendEmailHonorsContextDeadline(t *testing.T) {
	// A server that accepts the connection but never sends the SMTP greeting, so the client blocks reading it
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	release := make(chan struct{})
	defer close(release)
	go func() {
		conn, aerr := listener.Accept()
		if aerr != nil {
			return
		}
		// Hold the connection open and silent until the test releases it
		<-release
		_ = conn.Close()
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	connString, err := url.Parse(fmt.Sprintf("smtp://%s:%s?fromAddress=sender@example.com&tls=none", host, port))
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	// A short deadline must abort the silent session promptly instead of blocking forever
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- emailer.SendEmail(ctx, emaillib.EmailAddress{Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
	}()

	select {
	case sendErr := <-done:
		require.Error(t, sendErr, "SendEmail should fail once the context deadline elapses")
	case <-time.After(5 * time.Second):
		t.Fatal("SendEmail ignored the context deadline and hung")
	}
}

type smtpTestSession struct {
	authCommand  string
	authUsername string
	authPassword string
	mailFrom     string
	rcptTo       string
	message      string
	commands     []string
}

type smtpTestServer struct {
	listener       net.Listener
	authMechanisms string
	sessionCh      chan smtpTestSession
	errorCh        chan error
}

func newSMTPTestServer(t *testing.T, authMechanisms string) *smtpTestServer {
	// Bind an ephemeral local port so tests can run in parallel without conflicting listeners
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &smtpTestServer{
		listener:       listener,
		authMechanisms: authMechanisms,
		sessionCh:      make(chan smtpTestSession, 1),
		errorCh:        make(chan error, 1),
	}
	go server.serve()

	t.Cleanup(func() {
		_ = listener.Close()
	})

	return server
}

func (s *smtpTestServer) address() string {
	return s.listener.Addr().String()
}

func (s *smtpTestServer) wait() (smtpTestSession, error) {
	// Wait for either a recorded SMTP session or a server-side error from the goroutine
	select {
	case session := <-s.sessionCh:
		return session, nil
	case err := <-s.errorCh:
		return smtpTestSession{}, err
	case <-time.After(5 * time.Second):
		return smtpTestSession{}, context.DeadlineExceeded
	}
}

func (s *smtpTestServer) serve() {
	// Accept a single client because each test only performs one SMTP delivery
	conn, err := s.listener.Accept()
	if err != nil {
		s.errorCh <- err
		return
	}

	session, err := s.handleConn(conn)
	if err != nil {
		s.errorCh <- err
		return
	}

	s.sessionCh <- session
}

func (s *smtpTestServer) handleConn(conn net.Conn) (smtpTestSession, error) {
	// Close the connection on every exit path so the test server never leaks sockets
	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	session := smtpTestSession{}

	// Start with the standard SMTP greeting so the client can proceed with EHLO
	err := writeSMTPResponse(writer, "220 localhost ESMTP test")
	if err != nil {
		return smtpTestSession{}, err
	}

	// Process commands until the client sends QUIT after the message is accepted
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return smtpTestSession{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		session.commands = append(session.commands, line)

		switch {
		case strings.HasPrefix(line, "EHLO "):
			err = writeSMTPResponse(writer, "250-localhost ESMTP test", "250 AUTH "+s.authMechanisms)
		case strings.HasPrefix(line, "AUTH LOGIN"):
			session.authCommand = line
			// LOGIN is a multi-step exchange where the server prompts for the base64-encoded username and password in turn
			session.authUsername, session.authPassword, err = handleLoginAuth(reader, writer)
		case strings.HasPrefix(line, "AUTH "):
			session.authCommand = line
			err = writeSMTPResponse(writer, "235 2.7.0 Authentication successful")
		case strings.HasPrefix(line, "MAIL FROM:"):
			session.mailFrom = strings.TrimPrefix(line, "MAIL FROM:")
			err = writeSMTPResponse(writer, "250 2.1.0 Ok")
		case strings.HasPrefix(line, "RCPT TO:"):
			session.rcptTo = strings.TrimPrefix(line, "RCPT TO:")
			err = writeSMTPResponse(writer, "250 2.1.5 Ok")
		case line == "DATA":
			err = writeSMTPResponse(writer, "354 End data with <CR><LF>.<CR><LF>")
			if err != nil {
				return smtpTestSession{}, err
			}
			session.message, err = readSMTPData(reader)
			if err != nil {
				return smtpTestSession{}, err
			}
			err = writeSMTPResponse(writer, "250 2.0.0 Ok: queued")
		case line == "QUIT":
			err = writeSMTPResponse(writer, "221 2.0.0 Bye")
			if err != nil {
				return smtpTestSession{}, err
			}
			return session, nil
		default:
			err = writeSMTPResponse(writer, "250 2.0.0 Ok")
		}

		if err != nil {
			return smtpTestSession{}, err
		}
	}
}

func writeSMTPResponse(writer *bufio.Writer, lines ...string) error {
	// Flush after every response so the client does not block waiting for server output
	for _, line := range lines {
		_, err := writer.WriteString(line + "\r\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func readSMTPData(reader *bufio.Reader) (string, error) {
	// DATA ends on a single dot line, with doubled leading dots used for transparent transport
	var body strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if line == ".\r\n" {
			return body.String(), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		_, err = io.WriteString(&body, line)
		if err != nil {
			return "", err
		}
	}
}

// handleLoginAuth drives the server side of the SMTP AUTH LOGIN exchange and returns the decoded credentials
func handleLoginAuth(reader *bufio.Reader, writer *bufio.Writer) (username string, password string, err error) {
	// Prompt for the username, then decode the client's base64-encoded reply
	err = writeSMTPResponse(writer, "334 "+base64.StdEncoding.EncodeToString([]byte("Username:")))
	if err != nil {
		return "", "", err
	}
	username, err = readBase64Line(reader)
	if err != nil {
		return "", "", err
	}

	// Prompt for the password next, following the same base64 challenge/response pattern
	err = writeSMTPResponse(writer, "334 "+base64.StdEncoding.EncodeToString([]byte("Password:")))
	if err != nil {
		return "", "", err
	}
	password, err = readBase64Line(reader)
	if err != nil {
		return "", "", err
	}

	err = writeSMTPResponse(writer, "235 2.7.0 Authentication successful")
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

// readBase64Line reads one CRLF-terminated line and decodes its base64 payload
func readBase64Line(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimRight(line, "\r\n"))
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func TestSendEmailAutoTLSFailsIfStartTLSNotSupported(t *testing.T) {
	// The test server's EHLO response does not include STARTTLS tls=auto must refuse to send rather than silently downgrade to plaintext
	server := newSMTPTestServer(t, "PLAIN")
	host, port, err := net.SplitHostPort(server.address())
	require.NoError(t, err)

	connString, err := url.Parse(fmt.Sprintf("smtp://mailer:secret@%s:%s?fromAddress=sender@example.com&tls=auto", host, port))
	require.NoError(t, err)

	var emailer SMTPEmailer
	err = emailer.Init(t.Context(), InitOpts{ConnString: connString})
	require.NoError(t, err)

	err = emailer.SendEmail(t.Context(), emaillib.EmailAddress{Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
	require.ErrorContains(t, err, "STARTTLS")
}

func TestSendEmailEncodesDisplayNames(t *testing.T) {
	for _, name := range []string{"Max Müller", "Doe, Jane"} {
		t.Run(name, func(t *testing.T) {
			// Capture the real SMTP payload without advertising SMTPUTF8 support
			server := newSMTPTestServer(t, "PLAIN")
			connString := &url.URL{
				Scheme: "smtp",
				Host:   server.address(),
				RawQuery: url.Values{
					"fromAddress": {"sender@example.com"},
					"fromName":    {name},
					"tls":         {"none"},
				}.Encode(),
			}
			var emailer SMTPEmailer
			err := emailer.Init(t.Context(), InitOpts{ConnString: connString})
			require.NoError(t, err)
			err = emailer.SendEmail(t.Context(), emaillib.EmailAddress{Name: name, Address: "recipient@example.com"}, "Hello", emaillib.SendEmailMessage{Text: "Body"})
			require.NoError(t, err)

			// Display names belong only in headers while SMTP envelope addresses stay unchanged
			session, err := server.wait()
			require.NoError(t, err)
			assert.Equal(t, "<sender@example.com>", session.mailFrom)
			assert.Equal(t, "<recipient@example.com>", session.rcptTo)
			message, err := mail.ReadMessage(strings.NewReader(session.message))
			require.NoError(t, err)
			for header, address := range map[string]string{"From": "sender@example.com", "To": "recipient@example.com"} {
				value := message.Header.Get(header)
				assert.NotRegexp(t, `[^\x00-\x7F]`, value, "%s must encode non-ASCII display names", header)
				addresses, err := message.Header.AddressList(header)
				require.NoError(t, err)
				require.Len(t, addresses, 1)
				assert.Equal(t, name, addresses[0].Name)
				assert.Equal(t, address, addresses[0].Address)
			}
		})
	}
}

func TestTransformFailurePreventsSMTPConnection(t *testing.T) {
	connection, err := url.Parse("smtp://localhost:25?fromAddress=sender@example.com&tls=none")
	require.NoError(t, err)
	failure := errors.New("signing failed")
	transport := &SMTPEmailer{}
	require.NoError(t, transport.Init(t.Context(), InitOpts{
		ConnString: connection,
		Transform: func(_ context.Context, payload []byte) ([]byte, error) {
			require.Contains(t, string(payload), "Content-Type:")
			return nil, failure
		},
	}))
	connected := false
	transport.dialContext = func(context.Context, string, string) (net.Conn, error) {
		connected = true
		return nil, errors.New("unexpected connection")
	}
	err = transport.SendEmail(t.Context(), emaillib.EmailAddress{Address: "recipient@example.com"}, "Test", emaillib.SendEmailMessage{Text: "Hello"})
	require.ErrorIs(t, err, failure)
	require.False(t, connected, "no unsigned message may reach SMTP when the transformation fails")
}
