//go:build unit

package email

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digitorus/pkcs7"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

func signingFixture(t *testing.T, algorithm string) (crypto.Signer, *x509.Certificate, []byte) {
	t.Helper()
	var key crypto.Signer
	var err error
	if algorithm == "RSA" {
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Pocket ID Test"},
		EmailAddresses: []string{"sender@example.test"}, NotBefore: time.Now().Add(-time.Hour),
		NotAfter: time.Now().Add(365 * 24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pfx, err := pkcs12.Modern.Encode(key, cert, nil, " password ")
	require.NoError(t, err)
	return key, cert, pfx
}

func decodeSignedMessage(t *testing.T, message []byte) ([]byte, []byte) {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(message))
	require.NoError(t, err)
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/signed", mediaType)
	require.Equal(t, "application/pkcs7-signature", params["protocol"])
	require.Equal(t, "sha-256", params["micalg"])
	reader := multipart.NewReader(msg.Body, params["boundary"])
	// NextRawPart retains quoted-printable bytes so verification uses exactly what was signed
	part, err := reader.NextRawPart()
	require.NoError(t, err)
	body, err := io.ReadAll(part)
	require.NoError(t, err)
	// Preserve the original MIME part headers as well as its body
	all, err := io.ReadAll(bytes.NewReader(message))
	require.NoError(t, err)
	marker := []byte("--" + params["boundary"] + "\r\n")
	start := bytes.Index(all, marker) + len(marker)
	end := bytes.Index(all[start:], []byte("\r\n--"+params["boundary"]+"\r\n"))
	require.GreaterOrEqual(t, end, 0)
	content := all[start : start+end]
	require.True(t, bytes.HasSuffix(content, body))
	part, err = reader.NextRawPart()
	require.NoError(t, err)
	require.Equal(t, "base64", part.Header.Get("Content-Transfer-Encoding"))
	sig, err := io.ReadAll(part)
	require.NoError(t, err)
	signature, err := base64.StdEncoding.DecodeString(string(sig))
	require.NoError(t, err)
	_, err = reader.NextRawPart()
	require.ErrorIs(t, err, io.EOF)
	return content, signature
}

func verifyMessage(t *testing.T, message []byte) ([]byte, []byte) {
	t.Helper()
	content, signature := decodeSignedMessage(t, message)
	signed, err := pkcs7.Parse(signature)
	require.NoError(t, err)
	require.Empty(t, signed.Content, "signature must be detached")
	signed.Content = content
	require.NoError(t, signed.Verify())
	// Tampering must invalidate the signature even if MIME structure stays intact
	signed.Content = append([]byte(nil), content...)
	signed.Content[len(signed.Content)-1] ^= 1
	require.Error(t, signed.Verify())
	return content, signature
}

func TestSigningUploadLifecycleAndSecretStorage(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	module, err := New(db)
	require.NoError(t, err)
	_, cert, pfx := signingFixture(t, "RSA")
	require.Error(t, module.SetSigningEnabled(t.Context(), true, "sender@example.test"))
	require.Error(t, module.UploadSigningCertificate(t.Context(), pfx, "wrong", "sender@example.test"))
	require.Error(t, module.UploadSigningCertificate(t.Context(), pfx, " password ", "other@example.test"))
	require.NoError(t, module.UploadSigningCertificate(t.Context(), pfx, " password ", "sender@example.test"))
	require.NoError(t, module.SetSigningEnabled(t.Context(), true, "sender@example.test"))
	status, err := module.SigningStatus(t.Context(), "sender@example.test")
	require.NoError(t, err)
	require.True(t, status.Enabled)
	require.True(t, status.Certificate.SenderMatches)
	require.Equal(t, cert.Issuer.String(), status.Certificate.Issuer)
	encoded, err := json.Marshal(status)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "privateKey")
	assert.NotContains(t, string(encoded), " password ")
	var ciphertext string
	require.NoError(t, db.Raw("SELECT private_key FROM email_signing_settings WHERE id = 1").Scan(&ciphertext).Error)
	settings, err := module.loadSigningSettings(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, settings.PrivateKey.String())
	assert.NotEqual(t, settings.PrivateKey.String(), ciphertext)
	// Invalid replacement must leave the active certificate and enabled setting intact
	require.Error(t, module.UploadSigningCertificate(t.Context(), []byte("invalid"), "", "sender@example.test"))
	after, err := module.SigningStatus(t.Context(), "sender@example.test")
	require.NoError(t, err)
	assert.Equal(t, status, after)
	require.NoError(t, module.RemoveSigningCertificate(t.Context()))
	after, err = module.SigningStatus(t.Context(), "sender@example.test")
	require.NoError(t, err)
	assert.False(t, after.Enabled)
	assert.Nil(t, after.Certificate)
	settings, err = module.loadSigningSettings(t.Context())
	require.NoError(t, err)
	assert.Empty(t, settings.PrivateKey)
}

func TestSignMessageInteroperability(t *testing.T) {
	for _, algorithm := range []string{"RSA", "ECDSA"} {
		t.Run(algorithm, func(t *testing.T) {
			key, cert, _ := signingFixture(t, algorithm)
			for _, body := range []string{
				"Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nGr=C3=BC=C3=9Fe\r\n.Line\r\n",
				"Content-Type: multipart/alternative; boundary=inner\r\n\r\n--inner\r\nContent-Type: text/plain\r\n\r\nHello\r\n--inner\r\nContent-Type: text/html\r\n\r\n<b>Hello</b>\r\n--inner--\r\n",
			} {
				message := []byte("From: sender@example.test\r\nTo: recipient@example.test\r\nSubject: Test\r\nMIME-Version: 1.0\r\n" + body)
				signed, err := signMessage(message, []*x509.Certificate{cert}, key)
				require.NoError(t, err)
				content, signature := verifyMessage(t, signed)
				assert.Equal(t, body, string(content))
				if _, err := exec.LookPath("openssl"); err == nil {
					// OpenSSL is an independent implementation and validates the bytes without trusting the test's self-signed issuer
					dir := t.TempDir()
					sigFile, contentFile := filepath.Join(dir, "signature.der"), filepath.Join(dir, "content.mime")
					require.NoError(t, os.WriteFile(sigFile, signature, 0600))
					require.NoError(t, os.WriteFile(contentFile, content, 0600))
					output, err := exec.Command("openssl", "cms", "-verify", "-binary", "-inform", "DER", "-in", sigFile, "-content", contentFile, "-noverify", "-out", filepath.Join(dir, "verified.mime")).CombinedOutput()
					require.NoError(t, err, string(output))
				}
			}
		})
	}
}

func TestEverySystemEmailIsSignedOnTheWire(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	recipient := "recipient@example.test"
	user := model.User{Base: model.Base{ID: "smime-recipient"}, Username: "smime-recipient", Email: &recipient, FirstName: "Müller"}
	require.NoError(t, db.Create(&user).Error)
	module, err := New(db)
	require.NoError(t, err)
	_, _, pfx := signingFixture(t, "RSA")
	require.NoError(t, module.UploadSigningCertificate(t.Context(), pfx, " password ", "sender@example.test"))
	require.NoError(t, module.SetSigningEnabled(t.Context(), true, "sender@example.test"))
	sends := []func(context.Context, *appconfig.AppConfigModel) error{
		func(ctx context.Context, cfg *appconfig.AppConfigModel) error {
			return module.SendTestEmail(ctx, cfg, user.ID)
		},
		func(ctx context.Context, cfg *appconfig.AppConfigModel) error {
			return module.SendEmailVerification(ctx, cfg, user.FullName(), recipient, "https://example.test/verify")
		},
		func(ctx context.Context, cfg *appconfig.AppConfigModel) error {
			return module.SendOneTimeAccessEmail(ctx, cfg, user.FullName(), recipient, "123456", "https://example.test/login", "https://example.test/login/123456", "15 minutes")
		},
		func(ctx context.Context, cfg *appconfig.AppConfigModel) error {
			return module.SendNewLogin(ctx, cfg, user.FullName(), recipient, "192.0.2.1", "Germany", "Berlin", "Firefox", "Passkey", time.Now())
		},
		func(ctx context.Context, cfg *appconfig.AppConfigModel) error {
			return module.SendAPIKeyExpiringSoon(ctx, cfg, user.FullName(), recipient, user.FirstName, "Test", time.Now())
		},
	}
	for i, send := range sends {
		t.Run([]string{"test", "verification", "access", "login", "api-key"}[i], func(t *testing.T) {
			server := newSMTPTestServer(t)
			cfg := newSMTPTestConfig(t, server.address())
			require.NoError(t, send(t.Context(), cfg))
			session, err := server.wait()
			require.NoError(t, err)
			verifyMessage(t, []byte(session.message))
			assert.Equal(t, "<sender@example.test>", session.mailFrom)
		})
	}
}

func TestSigningFailsBeforeSMTPForExpiredOrMismatchedCertificate(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	module, err := New(db)
	require.NoError(t, err)
	_, cert, pfx := signingFixture(t, "RSA")
	require.NoError(t, module.UploadSigningCertificate(t.Context(), pfx, " password ", "sender@example.test"))
	require.NoError(t, module.SetSigningEnabled(t.Context(), true, "sender@example.test"))
	cfg := &appconfig.AppConfigModel{SmtpHost: "127.0.0.1", SmtpPort: "1", SmtpFrom: "wrong@example.test"}
	// Port 1 would fail to connect; the returned identity error proves validation happened first
	err = module.SendEmailVerification(t.Context(), cfg, "User", "recipient@example.test", "https://example.test")
	require.ErrorContains(t, err, "must match")
	settings, err := module.loadSigningSettings(t.Context())
	require.NoError(t, err)
	_, signer, err := signerFromSettings(settings, "sender@example.test")
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		change func(*x509.Certificate)
		now    time.Time
		want   string
	}{
		{"expired", func(c *x509.Certificate) {}, cert.NotAfter.Add(time.Second), "expired"},
		{"future", func(c *x509.Certificate) {}, cert.NotBefore.Add(-time.Second), "not valid yet"},
		{"usage", func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth} }, time.Now(), "email protection"},
		{"key usage", func(c *x509.Certificate) { c.KeyUsage = x509.KeyUsageKeyEncipherment }, time.Now(), "digital signatures"},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := *cert
			test.change(&copy)
			require.ErrorContains(t, validateSigningCertificate(&copy, signer, "sender@example.test", test.now), test.want)
		})
	}
	// Simulate a previously valid certificate reaching expiry while signing remains enabled
	expiredTemplate := *cert
	expiredTemplate.NotBefore = time.Now().Add(-2 * time.Hour)
	expiredTemplate.NotAfter = time.Now().Add(-time.Hour)
	expiredDER, err := x509.CreateCertificate(rand.Reader, &expiredTemplate, &expiredTemplate, signer.Public(), signer)
	require.NoError(t, err)
	expiredChain, err := json.Marshal([][]byte{expiredDER})
	require.NoError(t, err)
	require.NoError(t, db.Model(&SigningSettings{}).Where("id = ?", 1).Update("certificate_chain", string(expiredChain)).Error)
	cfg.SmtpFrom = "sender@example.test"
	err = module.SendEmailVerification(t.Context(), cfg, "User", "recipient@example.test", "https://example.test")
	require.ErrorContains(t, err, "expired")
	expiredStatus, err := module.SigningStatus(t.Context(), "sender@example.test")
	require.NoError(t, err)
	require.Equal(t, "expired", expiredStatus.Certificate.Status)
	differentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	require.ErrorContains(t, validateSigningCertificate(cert, differentKey, "sender@example.test", time.Now()), "does not match")

	// Explicitly disabling signing restores ordinary delivery even when the certificate no longer matches
	require.NoError(t, module.SetSigningEnabled(t.Context(), false, "wrong@example.test"))
	server := newSMTPTestServer(t)
	cfg = newSMTPTestConfig(t, server.address())
	require.NoError(t, module.SendEmailVerification(t.Context(), cfg, "User", "recipient@example.test", "https://example.test"))
	session, err := server.wait()
	require.NoError(t, err)
	assert.False(t, strings.Contains(session.message, "multipart/signed"))
}

func TestSigningWithIssuerChain(t *testing.T) {
	leafKey, leaf, _ := signingFixture(t, "ECDSA")
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(100), Subject: pkix.Name{CommonName: "Test Root"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(400 * 24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, rootKey.Public(), rootKey)
	require.NoError(t, err)
	root, err := x509.ParseCertificate(rootDER)
	require.NoError(t, err)
	intermediateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	intermediateTemplate := *rootTemplate
	intermediateTemplate.Subject = pkix.Name{CommonName: "Test Intermediate"}
	intermediateTemplate.SerialNumber = big.NewInt(101)
	intermediateDER, err := x509.CreateCertificate(rand.Reader, &intermediateTemplate, root, intermediateKey.Public(), rootKey)
	require.NoError(t, err)
	intermediate, err := x509.ParseCertificate(intermediateDER)
	require.NoError(t, err)
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, intermediate, leafKey.Public(), intermediateKey)
	require.NoError(t, err)
	leaf, err = x509.ParseCertificate(leafDER)
	require.NoError(t, err)
	// PKCS#12 does not promise issuer order; upload must arrange the chain for CMS
	pfx, err := pkcs12.Modern.Encode(leafKey, leaf, []*x509.Certificate{root, intermediate}, "chain")
	require.NoError(t, err)
	module, err := New(testutils.NewDatabaseForTest(t))
	require.NoError(t, err)
	require.NoError(t, module.UploadSigningCertificate(t.Context(), pfx, "chain", "sender@example.test"))
	settings, err := module.loadSigningSettings(t.Context())
	require.NoError(t, err)
	certs, signer, err := signerFromSettings(settings, "sender@example.test")
	require.NoError(t, err)
	require.Len(t, certs, 3)
	assert.Equal(t, "Test Intermediate", certs[1].Subject.CommonName)
	message, err := signMessage([]byte("From: sender@example.test\r\nMIME-Version: 1.0\r\nContent-Type: text/plain\r\n\r\nHello\r\n"), certs, signer)
	require.NoError(t, err)
	content, signature := verifyMessage(t, message)
	signed, err := pkcs7.Parse(signature)
	require.NoError(t, err)
	signed.Content = content
	roots := x509.NewCertPool()
	roots.AddCert(root)
	require.NoError(t, signed.VerifyWithChain(roots))
}
