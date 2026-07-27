//go:build unit

package email

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestNewLoadsEveryEmailTemplate(t *testing.T) {
	module, err := New(nil)
	require.NoError(t, err)
	require.Len(t, module.textTemplates, len(templatePaths))
	require.Len(t, module.htmlTemplates, len(templatePaths))

	for _, templatePath := range templatePaths {
		assert.NotNil(t, module.textTemplates[templatePath])
		assert.NotNil(t, module.htmlTemplates[templatePath])
	}
}

func TestModuleSendsEveryEmailType(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	userEmail := "recipient@example.test"
	user := model.User{
		Base:      model.Base{ID: "email-recipient"},
		Username:  "email-recipient",
		Email:     &userEmail,
		FirstName: "Test",
		LastName:  "User",
	}
	require.NoError(t, db.Create(&user).Error)

	module, err := New(db)
	require.NoError(t, err)

	eventTime := time.Date(2030, time.January, 2, 15, 4, 5, 0, time.UTC)
	tests := []struct {
		name         string
		subject      string
		bodyContains []string
		send         func(ctx context.Context, config *appconfig.AppConfigModel) error
	}{
		{
			name:         "test email",
			subject:      "Test email",
			bodyContains: []string{"TEST EMAIL", "Your email setup is working correctly!"},
			send: func(ctx context.Context, config *appconfig.AppConfigModel) error {
				return module.SendTestEmail(ctx, config, user.ID)
			},
		},
		{
			name:         "email verification",
			subject:      "Verify your Pocket ID Test email address",
			bodyContains: []string{"EMAIL VERIFICATION", "Hello Test User", "https://id.example.test/verify-token"},
			send: func(ctx context.Context, config *appconfig.AppConfigModel) error {
				return module.SendEmailVerification(ctx, config, user.FullName(), userEmail, "https://id.example.test/verify-token")
			},
		},
		{
			name:         "one-time access",
			subject:      "Login Code",
			bodyContains: []string{"YOUR LOGIN CODE", "123456", "https://id.example.test/lc/123456", "15 minutes"},
			send: func(ctx context.Context, config *appconfig.AppConfigModel) error {
				return module.SendOneTimeAccessEmail(ctx, config, user.FullName(), userEmail, "123456", "https://id.example.test/lc", "https://id.example.test/lc/123456", "15 minutes")
			},
		},
		{
			name:         "new login",
			subject:      "New device login with Pocket ID Test",
			bodyContains: []string{"NEW SIGN-IN DETECTED", "Zurich, Switzerland", "192.0.2.10", "Firefox on Linux", "January 2, 2030 at 3:04 PM UTC"},
			send: func(ctx context.Context, config *appconfig.AppConfigModel) error {
				return module.SendNewLogin(ctx, config, user.FullName(), userEmail, "192.0.2.10", "Switzerland", "Zurich", "Firefox on Linux", eventTime)
			},
		},
		{
			name:         "API key expiration",
			subject:      `API Key "Automation" Expiring Soon`,
			bodyContains: []string{"API KEY EXPIRING SOON", "Hello Test", "Automation", "2030-01-02 15:04:05 UTC"},
			send: func(ctx context.Context, config *appconfig.AppConfigModel) error {
				return module.SendAPIKeyExpiringSoon(ctx, config, user.FullName(), userEmail, user.FirstName, "Automation", eventTime)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Use the real SMTP transport so the test covers module mapping, rendering, MIME generation, and delivery together
			server := newSMTPTestServer(t)
			config := newSMTPTestConfig(t, server.address())

			require.NoError(t, test.send(t.Context(), config))

			session, sessionErr := server.wait()
			require.NoError(t, sessionErr)
			assert.Equal(t, "<sender@example.test>", session.mailFrom)
			assert.Equal(t, "<recipient@example.test>", session.rcptTo)
			assert.Contains(t, session.message, "From: Pocket ID Test <sender@example.test>\r\n")
			assert.Contains(t, session.message, "To: Test User <recipient@example.test>\r\n")
			assert.Contains(t, session.message, "Subject: "+test.subject+"\r\n")
			assert.Contains(t, session.message, "Content-Type: multipart/alternative; boundary=")
			assert.Contains(t, session.message, "Content-Type: text/plain; charset=UTF-8")
			assert.Contains(t, session.message, "Content-Type: text/html; charset=UTF-8")
			for _, expected := range test.bodyContains {
				assert.Contains(t, session.message, expected)
			}
		})
	}
}

func TestSendTestEmailRequiresUserEmail(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{
		Base:     model.Base{ID: "user-without-email"},
		Username: "user-without-email",
	}
	require.NoError(t, db.Create(&user).Error)

	module, err := New(db)
	require.NoError(t, err)

	err = module.SendTestEmail(t.Context(), &appconfig.AppConfigModel{}, user.ID)
	var emailNotSetError *common.UserEmailNotSetError
	require.ErrorAs(t, err, &emailNotSetError)
}

func TestSMTPConnStringPreservesConfiguration(t *testing.T) {
	config := &appconfig.AppConfigModel{
		AppName:            "Pocket ID Test",
		SmtpHost:           "smtp.example.test",
		SmtpPort:           "2525",
		SmtpFrom:           "sender@example.test",
		SmtpUser:           "mailer",
		SmtpPassword:       "secret",
		SmtpTls:            "starttls",
		SmtpSkipCertVerify: "true",
	}

	connectionString, err := smtpConnString(config)
	require.NoError(t, err)

	smtpURL, err := url.Parse(connectionString)
	require.NoError(t, err)
	assert.Equal(t, "smtp", smtpURL.Scheme)
	assert.Equal(t, "smtp.example.test:2525", smtpURL.Host)
	assert.Equal(t, "mailer", smtpURL.User.Username())
	password, hasPassword := smtpURL.User.Password()
	assert.True(t, hasPassword)
	assert.Equal(t, "secret", password)
	assert.Equal(t, "sender@example.test", smtpURL.Query().Get("fromAddress"))
	assert.Equal(t, "Pocket ID Test", smtpURL.Query().Get("fromName"))
	assert.Equal(t, "starttls", smtpURL.Query().Get("tls"))
	assert.Equal(t, "true", smtpURL.Query().Get("insecureSkipVerify"))
}

func TestSMTPConnStringRequiresHostAndDefaultsTLS(t *testing.T) {
	_, err := smtpConnString(&appconfig.AppConfigModel{})
	require.ErrorContains(t, err, "SMTP host is not configured")

	connectionString, err := smtpConnString(&appconfig.AppConfigModel{SmtpHost: "smtp.example.test"})
	require.NoError(t, err)

	smtpURL, err := url.Parse(connectionString)
	require.NoError(t, err)
	assert.Equal(t, "none", smtpURL.Query().Get("tls"))
	assert.Empty(t, smtpURL.Query().Get("insecureSkipVerify"))
}

type smtpTestSession struct {
	mailFrom string
	rcptTo   string
	message  string
}

type smtpTestServer struct {
	listener  net.Listener
	sessionCh chan smtpTestSession
	errorCh   chan error
}

func newSMTPTestServer(t *testing.T) *smtpTestServer {
	t.Helper()

	// Bind an ephemeral loopback port so each delivery test gets an isolated SMTP endpoint
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &smtpTestServer{
		listener:  listener,
		sessionCh: make(chan smtpTestSession, 1),
		errorCh:   make(chan error, 1),
	}
	go server.serve()

	t.Cleanup(func() {
		_ = listener.Close()
	})

	return server
}

func newSMTPTestConfig(t *testing.T, address string) *appconfig.AppConfigModel {
	t.Helper()

	host, port, err := net.SplitHostPort(address)
	require.NoError(t, err)

	return &appconfig.AppConfigModel{
		AppName:  "Pocket ID Test",
		SmtpHost: appconfig.AppConfigValue(host),
		SmtpPort: appconfig.AppConfigValue(port),
		SmtpFrom: "sender@example.test",
		SmtpTls:  "none",
	}
}

func (s *smtpTestServer) address() string {
	return s.listener.Addr().String()
}

func (s *smtpTestServer) wait() (smtpTestSession, error) {
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
	conn, err := s.listener.Accept()
	if err != nil {
		s.errorCh <- err
		return
	}

	session, err := handleSMTPConnection(conn)
	if err != nil {
		s.errorCh <- err
		return
	}

	s.sessionCh <- session
}

func handleSMTPConnection(conn net.Conn) (smtpTestSession, error) {
	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	session := smtpTestSession{}

	err := writeSMTPResponse(writer, "220 localhost ESMTP test")
	if err != nil {
		return smtpTestSession{}, err
	}

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return smtpTestSession{}, readErr
		}
		line = strings.TrimRight(line, "\r\n")

		switch {
		case strings.HasPrefix(line, "EHLO "):
			err = writeSMTPResponse(writer, "250-localhost ESMTP test", "250 OK")
		case strings.HasPrefix(line, "HELO "):
			err = writeSMTPResponse(writer, "250 localhost")
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
			if err == nil {
				err = writeSMTPResponse(writer, "250 2.0.0 Ok: queued")
			}
		case line == "QUIT":
			err = writeSMTPResponse(writer, "221 2.0.0 Bye")
			return session, err
		default:
			err = writeSMTPResponse(writer, "250 2.0.0 Ok")
		}

		if err != nil {
			return smtpTestSession{}, err
		}
	}
}

func writeSMTPResponse(writer *bufio.Writer, lines ...string) error {
	for _, line := range lines {
		_, err := writer.WriteString(line + "\r\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func readSMTPData(reader *bufio.Reader) (string, error) {
	var message strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if line == ".\r\n" {
			return message.String(), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		_, err = io.WriteString(&message, line)
		if err != nil {
			return "", err
		}
	}
}
