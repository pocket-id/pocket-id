// Adapted from github.com/italypaleale/go-kit/emailer/smtp v1.2.5
// See LICENSE.txt for the upstream MIT license

package smtptransport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	stdsmtp "net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/italypaleale/go-kit/emailer"
	"github.com/italypaleale/go-kit/utils"
	"net/mail"
	"net/url"
)

const (
	smtpTLSAuto     = "auto"
	smtpTLSStartTLS = "starttls"
	smtpTLSImplicit = "implicit"
	smtpTLSNone     = "none"
)

// SMTPEmailer is an Emailer that uses a traditional SMTP server
type SMTPEmailer struct {
	host        string
	port        string
	address     string
	username    string
	password    string
	from        string
	fromAddress string
	localName   string
	tlsMode     string
	tlsConfig   *tls.Config
	transform   func(context.Context, []byte) ([]byte, error)
	dialContext func(ctx context.Context, network string, address string) (net.Conn, error)
	now         func() time.Time
}

// Init validates the SMTP connection string and stores the transport configuration for later sends
func (s *SMTPEmailer) Init(_ context.Context, opts InitOpts) error {
	s.transform = opts.Transform
	const connStringFormat = "smtp://<username>:<password>@<host>:<port>?fromAddress=<address>&fromName=<name>&tls=<auto|starttls|implicit|none>&insecureSkipVerify=<true|false>"

	// Validate the connection string scheme and the target server location
	if opts.ConnString == nil {
		return errors.New("invalid connection string: missing SMTP connection string")
	}
	if opts.ConnString.Scheme != "smtp" {
		return fmt.Errorf("invalid connection string scheme; required format is '%s'", connStringFormat)
	}
	host := opts.ConnString.Hostname()
	if host == "" {
		return fmt.Errorf("invalid connection string: missing SMTP host; required format is '%s'", connStringFormat)
	}

	// Normalize the TLS mode early because it affects the default port choice
	tlsMode, err := parseTLSMode(opts.ConnString.Query().Get("tls"))
	if err != nil {
		return fmt.Errorf("invalid connection string: %w; required format is '%s'", err, connStringFormat)
	}
	port := opts.ConnString.Port()
	if port == "" {
		port = defaultPortForTLSMode(tlsMode)
	}

	// Optionally disable TLS certificate verification for servers that use self-signed or internal-CA certificates
	insecureSkipVerify := utils.IsTruthy(opts.ConnString.Query().Get("insecureSkipVerify"))

	// Capture credentials when present and reject partial auth configuration
	username := ""
	password := ""
	if opts.ConnString.User != nil {
		username = opts.ConnString.User.Username()
		password, _ = opts.ConnString.User.Password()
	}
	if username != "" && password == "" {
		return fmt.Errorf("invalid connection string: missing SMTP password; required format is '%s'", connStringFormat)
	}

	// Preserve both the RFC5321 envelope sender and the user-visible From header
	fromAddress := opts.ConnString.Query().Get("fromAddress")
	fromName := opts.ConnString.Query().Get("fromName")
	if fromAddress == "" {
		return fmt.Errorf("invalid connection string: missing from address; required format is '%s'", connStringFormat)
	}

	// Reject CR/LF in the sender fields so they cannot inject extra headers into every message sent by this emailer
	err = validateHeaderValue("from address", fromAddress)
	if err != nil {
		return fmt.Errorf("invalid connection string: %w; required format is '%s'", err, connStringFormat)
	}
	err = validateHeaderValue("from name", fromName)
	if err != nil {
		return fmt.Errorf("invalid connection string: %w; required format is '%s'", err, connStringFormat)
	}
	err = validateEmailAddress("from address", fromAddress)
	if err != nil {
		return fmt.Errorf("invalid connection string: %w; required format is '%s'", err, connStringFormat)
	}

	// Use the operating-system hostname so SMTP relays receive a real client identity instead of net/smtp's localhost default
	localName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get SMTP client hostname: %w", err)
	}

	s.host = host
	s.port = port
	s.address = net.JoinHostPort(host, port)
	s.username = username
	s.password = password
	s.fromAddress = fromAddress
	s.localName = localName
	s.tlsMode = tlsMode
	s.from = formatFromAddress(fromName, fromAddress)

	// Prepare a reusable TLS config so both implicit TLS and STARTTLS use the same policy
	if s.tlsConfig == nil {
		s.tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         host,
			InsecureSkipVerify: insecureSkipVerify, //#nosec G402 -- user opt-in
		}
	}

	// Leave dial injection available for tests while defaulting to the standard net.Dialer path
	if s.dialContext == nil {
		var dialer net.Dialer
		s.dialContext = dialer.DialContext
	}

	// Leave clock injection available for tests while defaulting to the wall clock for the Date header
	if s.now == nil {
		s.now = time.Now
	}

	return nil
}

// SendEmail sends a MIME email over SMTP using the configured auth and TLS mode
func (s SMTPEmailer) SendEmail(ctx context.Context, to emailer.EmailAddress, subject string, message emailer.SendEmailMessage) error {
	// Build the MIME message first so transport errors are not mixed with formatting errors
	payload, err := s.buildMessage(to, subject, message)
	if err != nil {
		return fmt.Errorf("failed to build SMTP email: %w", err)
	}

	// Transform the final MIME bytes before opening a connection so failures cannot deliver an unsigned message
	if s.transform != nil {
		payload, err = s.transform(ctx, payload)
		if err != nil {
			return fmt.Errorf("failed to transform SMTP email: %w", err)
		}
	}

	// Establish the network connection using the requested TLS mode
	conn, err := s.dialSMTP(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// net/smtp performs blocking socket reads with no awareness of ctx, so honor the caller's context for the rest of the conversation
	// A deadline (when the caller set one) bounds an unresponsive server, and closing the connection on cancellation unblocks any in-flight read
	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetDeadline(deadline)
	}
	stopWatch := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopWatch()

	// Hand the connection to the SMTP client so the rest of the session can use standard commands
	client, err := stdsmtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() {
		_ = client.Close()
	}()

	// Set the client hostname before STARTTLS because net/smtp otherwise sends EHLO localhost and some relays reject it
	err = client.Hello(s.localName)
	if err != nil {
		return fmt.Errorf("failed to send SMTP greeting: %w", err)
	}

	// Upgrade the connection before auth when the selected mode requires transport security
	err = s.configureTLS(client)
	if err != nil {
		return fmt.Errorf("failed to configure SMTP transport security: %w", err)
	}

	// Authenticate only when credentials were provided in the connection string
	err = s.authenticate(client)
	if err != nil {
		return fmt.Errorf("failed to authenticate with SMTP server: %w", err)
	}

	// Send the RFC5321 envelope before streaming the MIME message body
	err = client.Mail(s.fromAddress)
	if err != nil {
		return fmt.Errorf("failed to set SMTP sender: %w", err)
	}
	err = client.Rcpt(to.Address)
	if err != nil {
		return fmt.Errorf("failed to set SMTP recipient: %w", err)
	}

	// Write the message body as a DATA segment and close the writer to finalize the message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open SMTP data writer: %w", err)
	}
	_, err = writer.Write(payload)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write SMTP message: %w", err)
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to finalize SMTP message: %w", err)
	}

	// Quit cleanly so the server can commit the message before the connection closes
	err = client.Quit()
	if err != nil {
		return fmt.Errorf("failed to close SMTP session: %w", err)
	}

	return nil
}

// buildMessage renders a UTF-8 MIME email with either one body part or multipart/alternative
func (s SMTPEmailer) buildMessage(to emailer.EmailAddress, subject string, message emailer.SendEmailMessage) ([]byte, error) {
	// Reject CR/LF in the caller-supplied header values so a crafted recipient or subject cannot inject extra headers or a second body
	err := validateHeaderValue("recipient name", to.Name)
	if err != nil {
		return nil, err
	}
	err = validateHeaderValue("recipient address", to.Address)
	if err != nil {
		return nil, err
	}
	err = validateEmailAddress("recipient address", to.Address)
	if err != nil {
		return nil, err
	}
	err = validateHeaderValue("subject", subject)
	if err != nil {
		return nil, err
	}

	// Build the MIME body first because the top-level headers depend on whether the message is multipart
	body, contentType, transferEncoding, err := buildMIMEBody(message)
	if err != nil {
		return nil, err
	}

	// Generate a Message-ID so relays and spam filters see a stable, unique identifier for the message
	messageID, err := generateMessageID(messageIDDomain(s.fromAddress))
	if err != nil {
		return nil, err
	}

	// Write the standard headers using CRLF so SMTP servers receive a valid RFC5322 message
	var payload bytes.Buffer
	_, err = fmt.Fprintf(&payload, "From: %s\r\n", s.from)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(&payload, "To: %s\r\n", to.Format())
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(&payload, "Subject: %s\r\n", encodeHeader(subject))
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(&payload, "Date: %s\r\n", s.now().Format(time.RFC1123Z))
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(&payload, "Message-ID: %s\r\n", messageID)
	if err != nil {
		return nil, err
	}
	_, err = payload.WriteString("MIME-Version: 1.0\r\n")
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(&payload, "Content-Type: %s\r\n", contentType)
	if err != nil {
		return nil, err
	}
	if transferEncoding != "" {
		_, err = fmt.Fprintf(&payload, "Content-Transfer-Encoding: %s\r\n", transferEncoding)
		if err != nil {
			return nil, err
		}
	}
	_, err = payload.WriteString("\r\n")
	if err != nil {
		return nil, err
	}
	_, err = payload.Write(body)
	if err != nil {
		return nil, err
	}

	return payload.Bytes(), nil
}

// dialSMTP opens the underlying TCP connection and applies implicit TLS when requested
func (s SMTPEmailer) dialSMTP(ctx context.Context) (net.Conn, error) {
	// Implicit TLS performs the handshake before the SMTP protocol starts
	if s.tlsMode == smtpTLSImplicit {
		tlsDialer := tls.Dialer{
			NetDialer: &net.Dialer{},
			Config:    s.tlsConfig,
		}
		return tlsDialer.DialContext(ctx, "tcp", s.address)
	}

	return s.dialContext(ctx, "tcp", s.address)
}

// configureTLS upgrades the SMTP session with STARTTLS when the selected mode requires it
func (s SMTPEmailer) configureTLS(client *stdsmtp.Client) error {
	// Implicit TLS has already completed before the SMTP session begins
	if s.tlsMode == smtpTLSImplicit || s.tlsMode == smtpTLSNone {
		return nil
	}

	// STARTTLS support is discovered from the server extensions announced after EHLO
	hasStartTLS, _ := client.Extension("STARTTLS")
	if !hasStartTLS {
		if s.tlsMode == smtpTLSStartTLS {
			return errors.New("SMTP server does not support STARTTLS")
		}

		// auto mode fails closed: silently downgrading to plaintext would send credentials unencrypted Callers that intentionally want no TLS must set tls=none explicitly
		return errors.New("SMTP server does not support STARTTLS - use tls=none to allow unencrypted connections")
	}

	return client.StartTLS(s.tlsConfig)
}

// authenticate sends SMTP AUTH only when the connection string included credentials
func (s SMTPEmailer) authenticate(client *stdsmtp.Client) error {
	// Allow anonymous SMTP relays or local test servers when no credentials were configured
	if s.username == "" {
		return nil
	}

	// Refuse to send credentials when the server does not advertise SMTP AUTH
	hasAuth, mechanisms := client.Extension("AUTH")
	if !hasAuth {
		return errors.New("SMTP server does not support AUTH")
	}

	// Prefer PLAIN and fall back to LOGIN for servers such as some Exchange deployments that only offer the older mechanism
	var auth stdsmtp.Auth
	switch {
	case hasMechanism(mechanisms, "PLAIN"):
		auth = stdsmtp.PlainAuth("", s.username, s.password, s.host)
	case hasMechanism(mechanisms, "LOGIN"):
		auth = &loginAuth{username: s.username, password: s.password, host: s.host}
	default:
		return fmt.Errorf("SMTP server does not support PLAIN or LOGIN authentication (offered: %s)", mechanisms)
	}

	return client.Auth(auth)
}

// hasMechanism reports whether the space-separated AUTH mechanism list advertised by the server includes the given mechanism
func hasMechanism(mechanisms string, want string) bool {
	for m := range strings.FieldsSeq(mechanisms) {
		if strings.ToUpper(m) == want {
			return true
		}
	}

	return false
}

// loginAuth implements the non-standard but widely deployed SMTP AUTH LOGIN mechanism
// net/smtp only ships PLAIN and CRAM-MD5, so servers that offer only LOGIN need this implementation
type loginAuth struct {
	username string
	password string
	host     string
}

// Start begins the LOGIN exchange after refusing to leak credentials over an unencrypted, non-local connection
func (a *loginAuth) Start(server *stdsmtp.ServerInfo) (string, []byte, error) {
	// Mirror net/smtp's PlainAuth safety check so credentials are never sent over an unencrypted, non-local connection
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}

	return "LOGIN", nil, nil
}

// Next answers the server's username and password prompts, which net/smtp delivers already base64-decoded
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	// The prompt wording varies between servers, so match on the leading token rather than the exact string
	prompt := strings.ToLower(string(fromServer))
	switch {
	case strings.HasPrefix(prompt, "user"):
		return []byte(a.username), nil
	case strings.HasPrefix(prompt, "pass"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected SMTP LOGIN challenge: %q", fromServer)
	}
}

// isLocalhost reports whether the server name refers to the local machine, matching net/smtp's own check
func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

// messageIDDomain extracts the domain from the sender address for use in the Message-ID header
func messageIDDomain(fromAddress string) string {
	// Fall back to localhost when the address has no usable domain so the Message-ID stays syntactically valid
	at := strings.LastIndex(fromAddress, "@")
	if at < 0 || at == len(fromAddress)-1 {
		return "localhost"
	}

	return fromAddress[at+1:]
}

// generateMessageID builds an RFC5322 Message-ID from random bytes so each message gets a unique identifier
func generateMessageID(domain string) (string, error) {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to generate message ID: %w", err)
	}

	return "<" + hex.EncodeToString(buf) + "@" + domain + ">", nil
}

// buildMIMEBody creates either a single-part body or a multipart/alternative body when HTML is present
func buildMIMEBody(message emailer.SendEmailMessage) ([]byte, string, string, error) {
	// HTML-only messages should still send a valid single-part body instead of an empty text section
	if message.HTML == "" {
		body, err := encodeBodyPart(message.Text)
		if err != nil {
			return nil, "", "", err
		}
		return body, "text/plain; charset=UTF-8", "quoted-printable", nil
	}
	if message.Text == "" {
		body, err := encodeBodyPart(message.HTML)
		if err != nil {
			return nil, "", "", err
		}
		return body, "text/html; charset=UTF-8", "quoted-printable", nil
	}

	// Multipart messages keep the text and HTML bodies separate so mail clients can choose the best rendering
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	err := writeMultipartPart(writer, "text/plain; charset=UTF-8", message.Text)
	if err != nil {
		return nil, "", "", err
	}
	err = writeMultipartPart(writer, "text/html; charset=UTF-8", message.HTML)
	if err != nil {
		return nil, "", "", err
	}
	err = writer.Close()
	if err != nil {
		return nil, "", "", err
	}

	return body.Bytes(), "multipart/alternative; boundary=" + writer.Boundary(), "", nil
}

// writeMultipartPart writes one quoted-printable MIME section into a multipart body
func writeMultipartPart(writer *multipart.Writer, contentType string, value string) error {
	// Each alternative part needs its own headers because mail clients parse them independently
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Type", contentType)
	headers.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := writer.CreatePart(headers)
	if err != nil {
		return err
	}

	qpWriter := quotedprintable.NewWriter(part)
	_, err = qpWriter.Write([]byte(value))
	if err != nil {
		_ = qpWriter.Close()
		return err
	}

	return qpWriter.Close()
}

// encodeBodyPart quoted-printable encodes a single text or HTML body for UTF-8 transport
func encodeBodyPart(value string) ([]byte, error) {
	// Single-part messages still use quoted-printable so non-ASCII content survives transit reliably
	var body bytes.Buffer
	qpWriter := quotedprintable.NewWriter(&body)
	_, err := qpWriter.Write([]byte(value))
	if err != nil {
		_ = qpWriter.Close()
		return nil, err
	}
	err = qpWriter.Close()
	if err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}

// validateHeaderValue rejects values containing CR or LF, which could otherwise inject additional SMTP headers or a second message body
func validateHeaderValue(field string, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("invalid %s: must not contain CR or LF characters", field)
	}

	return nil
}

// encodeHeader applies RFC2047 encoding only when the header contains non-ASCII content
func encodeHeader(value string) string {
	// Leave plain ASCII headers readable while still supporting UTF-8 subjects when needed
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("utf-8", value)
}

// isASCII reports whether a string can be emitted directly in a mail header without RFC2047 encoding
func isASCII(value string) bool {
	// Mail headers are safest when raw values are restricted to visible ASCII
	for _, r := range value {
		if r < 32 || r > 126 {
			return false
		}
	}

	return true
}

// parseTLSMode normalizes the tls query parameter into one of the supported SMTP transport modes
func parseTLSMode(value string) (string, error) {
	// Default to auto so standard port 587 servers can opportunistically upgrade with STARTTLS
	if value == "" {
		return smtpTLSAuto, nil
	}

	normalized := strings.ToLower(value)
	switch normalized {
	case smtpTLSAuto, smtpTLSStartTLS, smtpTLSImplicit, smtpTLSNone:
		return normalized, nil
	case "tls":
		return smtpTLSImplicit, nil
	default:
		return "", fmt.Errorf("unsupported SMTP tls mode '%s'", value)
	}
}

// defaultPortForTLSMode picks the common SMTP port for the selected TLS behavior
func defaultPortForTLSMode(tlsMode string) string {
	// Implicit TLS commonly listens on 465 while the rest of the SMTP modes usually use 587
	if tlsMode == smtpTLSImplicit {
		return "465"
	}

	return "587"
}

// InitOpts configures SMTP and an optional transformation of the complete message before delivery
type InitOpts struct {
	ConnString *url.URL
	Transform  func(context.Context, []byte) ([]byte, error)
}

func formatFromAddress(name, address string) string {
	if name == "" {
		return address
	}
	return (&mail.Address{Name: name, Address: address}).String()
}

func validateEmailAddress(field, address string) error {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", field, err)
	}
	return nil
}
