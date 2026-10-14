//go:build unit

package email

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisabledSigningDoesNotDecryptUnusedPrivateKey(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	module, err := New(db)
	require.NoError(t, err)
	_, _, pfx := signingFixture(t, "RSA")
	require.NoError(t, module.UploadSigningCertificate(t.Context(), pfx, " password ", "sender@example.test"))
	require.NoError(t, db.Exec("UPDATE email_signing_settings SET private_key = ? WHERE id = 1", "corrupt ciphertext").Error)
	cfg := &appconfig.AppConfigModel{SmtpHost: "localhost", SmtpFrom: "sender@example.test"}
	_, err = module.getEmailer(t.Context(), cfg)
	require.NoError(t, err, "disabled signing must not prevent unsigned delivery because of an unused key")
	status, err := module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	require.NotNil(t, status.Certificate)
	require.Error(t, module.SetSigningEnabled(t.Context(), true, cfg.SmtpFrom.String()))
	// An already-enabled configuration fails closed, but an administrator can still inspect and disable it
	require.NoError(t, db.Exec("UPDATE email_signing_settings SET enabled = TRUE WHERE id = 1").Error)
	_, err = module.getEmailer(t.Context(), cfg)
	require.Error(t, err)
	status, err = module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	require.NoError(t, module.SetSigningEnabled(t.Context(), false, cfg.SmtpFrom.String()))
	_, err = module.getEmailer(t.Context(), cfg)
	require.NoError(t, err)
	require.NoError(t, module.RemoveSigningCertificate(t.Context()))
	status, err = module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	assert.Nil(t, status.Certificate)
}

func TestSigningCertificateUsageValidation(t *testing.T) {
	key, original, _ := signingFixture(t, "RSA")
	_, other, _ := signingFixture(t, "RSA")
	tests := []struct {
		name   string
		change func(*x509.Certificate)
		want   string
	}{
		{"future", func(c *x509.Certificate) { c.NotBefore = time.Now().Add(time.Hour) }, "not valid yet"},
		{"expired", func(c *x509.Certificate) { c.NotAfter = time.Now().Add(-time.Hour) }, "expired"},
		{"CA", func(c *x509.Certificate) { c.IsCA = true }, "not a CA"},
		{"key usage", func(c *x509.Certificate) { c.KeyUsage = x509.KeyUsageKeyEncipherment }, "digital signatures"},
		{"EKU", func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth} }, "email protection"},
		{"email", func(c *x509.Certificate) { c.EmailAddresses = []string{"other@example.test"} }, "match the saved SMTP"},
		{"key mismatch", func(c *x509.Certificate) { c.PublicKey = other.PublicKey }, "private key does not match"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cert := *original
			test.change(&cert)
			require.ErrorContains(t, validateSigningCertificate(&cert, key, "sender@example.test", time.Now()), test.want)
		})
	}
	require.NoError(t, validateSigningCertificate(original, key, "SENDER@example.test", time.Now()))
}

func TestSigningPostgresLifecycle(t *testing.T) {
	db := testutils.NewPostgresDatabaseForTest(t)
	module, err := New(db)
	require.NoError(t, err)
	cfg := &appconfig.AppConfigModel{SmtpHost: "localhost", SmtpFrom: "sender@example.test"}
	_, _, first := signingFixture(t, "RSA")
	require.NoError(t, module.UploadSigningCertificate(t.Context(), indefiniteSigningEnvelope(t, first), " password ", cfg.SmtpFrom.String()))
	require.NoError(t, module.SetSigningEnabled(t.Context(), true, cfg.SmtpFrom.String()))
	_, _, replacement := signingFixture(t, "ECDSA")
	require.NoError(t, module.UploadSigningCertificate(t.Context(), replacement, " password ", cfg.SmtpFrom.String()))
	status, err := module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	assert.True(t, status.Enabled, "replacement must preserve signing activation")
	_, err = module.getEmailer(t.Context(), cfg)
	require.NoError(t, err)
	// Metadata queries and the conditional key projection must work with Postgres as well as SQLite
	require.NoError(t, db.Exec("UPDATE email_signing_settings SET private_key = ? WHERE id = 1", "corrupt ciphertext").Error)
	_, err = module.getEmailer(t.Context(), cfg)
	require.Error(t, err)
	_, err = module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	require.NoError(t, module.SetSigningEnabled(t.Context(), false, cfg.SmtpFrom.String()))
	_, err = module.getEmailer(t.Context(), cfg)
	require.NoError(t, err)
	require.NoError(t, module.RemoveSigningCertificate(t.Context()))
	status, err = module.SigningStatus(t.Context(), cfg.SmtpFrom.String())
	require.NoError(t, err)
	assert.False(t, status.Enabled)
	assert.Nil(t, status.Certificate)
}
