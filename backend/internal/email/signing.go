package email

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/digitorus/pkcs7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	pkcs12 "software.sslmate.com/src/go-pkcs12"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// SigningSettings is a singleton whose private key is encrypted at rest and never exposed through the settings API
type SigningSettings struct {
	ID               int `gorm:"primaryKey"`
	Enabled          bool
	CertificateChain string
	PrivateKey       datatype.EncryptedString `json:"-"`
}

func (SigningSettings) TableName() string { return "email_signing_settings" }

type SigningStatus struct {
	Enabled     bool             `json:"enabled"`
	Certificate *CertificateInfo `json:"certificate"`
}

type CertificateInfo struct {
	Subject       string    `json:"subject"`
	Issuer        string    `json:"issuer"`
	Emails        []string  `json:"emails"`
	NotBefore     time.Time `json:"notBefore"`
	NotAfter      time.Time `json:"notAfter"`
	Fingerprint   string    `json:"fingerprint"`
	Status        string    `json:"status"`
	SenderMatches bool      `json:"senderMatches"`
}

func signingError(message string) error {
	return apperror.New("smime_invalid_configuration", http.StatusBadRequest, message)
}

func (m *Module) loadSigningSettings(ctx context.Context, columns ...string) (*SigningSettings, error) {
	if m.db == nil {
		return &SigningSettings{}, nil
	}
	var settings SigningSettings
	query := m.db.WithContext(ctx)
	if len(columns) > 0 {
		query = query.Select(columns)
	}
	if err := query.First(&settings, "id = ?", 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &SigningSettings{ID: 1}, nil
		}
		return nil, err
	}
	return &settings, nil
}

func parseCertificates(encoded string) ([]*x509.Certificate, error) {
	var raw [][]byte
	if err := json.Unmarshal([]byte(encoded), &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("S/MIME certificate is missing")
	}
	certs := make([]*x509.Certificate, len(raw))
	for i, der := range raw {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, err
		}
		certs[i] = cert
	}
	return certs, nil
}

func senderMatches(cert *x509.Certificate, sender string) bool {
	for _, address := range cert.EmailAddresses {
		if strings.EqualFold(address, sender) {
			return true
		}
	}
	return false
}

func validateSigningCertificate(cert *x509.Certificate, signer crypto.Signer, sender string, now time.Time) error {
	// Validate identity and permitted usage before accepting a key for system emails
	if now.Before(cert.NotBefore) {
		return signingError("The S/MIME certificate is not valid yet")
	}
	if !now.Before(cert.NotAfter) {
		return signingError("The S/MIME certificate has expired")
	}
	if cert.IsCA {
		return signingError("Upload an email signing certificate, not a CA certificate")
	}
	if cert.KeyUsage != 0 && cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return signingError("The certificate does not permit digital signatures")
	}
	emailUsage := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageEmailProtection || usage == x509.ExtKeyUsageAny {
			emailUsage = true
		}
	}
	if !emailUsage {
		return signingError("The certificate must permit email protection")
	}
	if len(cert.UnhandledCriticalExtensions) > 0 {
		return signingError("The certificate has unsupported critical extensions")
	}
	if !senderMatches(cert, sender) {
		return signingError("The certificate email address must match the saved SMTP sender address")
	}

	// Bound supported algorithms and compare the actual public keys instead of trusting the container
	switch key := signer.Public().(type) {
	case *rsa.PublicKey:
		if key.N.BitLen() < 2048 || key.N.BitLen() > 8192 {
			return signingError("S/MIME requires an RSA key between 2048 and 8192 bits")
		}
	case *ecdsa.PublicKey:
		if key.Curve.Params().BitSize < 256 {
			return signingError("S/MIME requires an elliptic curve of at least 256 bits")
		}
	default:
		return signingError("Use an RSA or ECDSA email signing certificate")
	}
	actual, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return err
	}
	expected, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return signingError("The private key does not match the certificate")
	}
	return nil
}

func (m *Module) SigningStatus(ctx context.Context, sender string) (*SigningStatus, error) {
	// Metadata and removal remain accessible even if a stored private key cannot be decrypted
	settings, err := m.loadSigningSettings(ctx, "id", "enabled", "certificate_chain")
	if err != nil {
		return nil, err
	}
	result := &SigningStatus{Enabled: settings.Enabled}
	if settings.CertificateChain == "" {
		return result, nil
	}
	certs, err := parseCertificates(settings.CertificateChain)
	if err != nil {
		return nil, err
	}
	cert := certs[0]
	now := time.Now()
	status := "valid"
	switch {
	case now.Before(cert.NotBefore):
		status = "notYetValid"
	case !now.Before(cert.NotAfter):
		status = "expired"
	case cert.NotAfter.Before(now.Add(30 * 24 * time.Hour)):
		status = "expiringSoon"
	}
	fingerprint := sha256.Sum256(cert.Raw)
	result.Certificate = &CertificateInfo{
		Subject: cert.Subject.String(), Issuer: cert.Issuer.String(), Emails: cert.EmailAddresses,
		NotBefore: cert.NotBefore, NotAfter: cert.NotAfter, Fingerprint: hex.EncodeToString(fingerprint[:]),
		Status: status, SenderMatches: senderMatches(cert, sender),
	}
	return result, nil
}

func (m *Module) UploadSigningCertificate(ctx context.Context, pfx []byte, password, sender string) error {
	// Decode the encrypted upload in memory and never persist its password
	key, cert, parents, err := pkcs12.DecodeChain(pfx, password)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) || errors.Is(err, pkcs12.ErrDecryption) {
			return signingError("Unable to decrypt the PKCS#12 file; check the password")
		}
		return signingError("Unable to read the PKCS#12 file: " + err.Error())
	}
	signer, ok := key.(crypto.Signer)
	if !ok || cert == nil {
		return signingError("The file must contain a certificate and its private key")
	}
	// PKCS#12 bag order does not identify the sender certificate, so find the certificate belonging to the private key
	publicKey, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return err
	}
	certificates := append([]*x509.Certificate{cert}, parents...)
	matchingIndex := -1
	for i, candidate := range certificates {
		candidateKey, err := x509.MarshalPKIXPublicKey(candidate.PublicKey)
		if err != nil {
			return err
		}
		if bytes.Equal(publicKey, candidateKey) {
			matchingIndex = i
			break
		}
	}
	if matchingIndex < 0 {
		return signingError("The private key does not match the certificate")
	}
	cert = certificates[matchingIndex]
	parents = certificates[:matchingIndex]
	parents = append(parents, certificates[matchingIndex+1:]...)
	if err := validateSigningCertificate(cert, signer, sender, time.Now()); err != nil {
		return err
	}

	// Order the included issuer certificates so CMS can embed a usable chain without requiring a public trust root
	ordered := []*x509.Certificate{cert}
	for len(parents) > 0 {
		current := ordered[len(ordered)-1]
		found := -1
		for i, parent := range parents {
			if bytes.Equal(current.RawIssuer, parent.RawSubject) && current.CheckSignatureFrom(parent) == nil {
				found = i
				break
			}
		}
		if found < 0 {
			return signingError("The file contains an invalid or unrelated certificate chain")
		}
		ordered = append(ordered, parents[found])
		parents = append(parents[:found], parents[found+1:]...)
	}
	// Exercise the signing algorithm at upload time so an unusable certificate cannot replace a working one
	if _, err := detachedSignature([]byte("Pocket ID S/MIME certificate check"), ordered, signer); err != nil {
		return signingError("The certificate chain or signing algorithm is not supported")
	}
	raw := make([][]byte, len(ordered))
	for i, c := range ordered {
		raw[i] = c.Raw
	}
	chain, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	// Replace both halves atomically while retaining the administrator's enabled setting
	return m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"certificate_chain", "private_key"}),
	}).Create(&SigningSettings{ID: 1, CertificateChain: string(chain), PrivateKey: datatype.EncryptedString(base64.StdEncoding.EncodeToString(der))}).Error
}

func signerFromSettings(settings *SigningSettings, sender string) ([]*x509.Certificate, crypto.Signer, error) {
	certs, err := parseCertificates(settings.CertificateChain)
	if err != nil {
		return nil, nil, signingError("Upload an S/MIME certificate before enabling signing")
	}
	der, err := base64.StdEncoding.DecodeString(settings.PrivateKey.String())
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, signingError("The S/MIME private key is not supported")
	}
	if err := validateSigningCertificate(certs[0], signer, sender, time.Now()); err != nil {
		return nil, nil, err
	}
	return certs, signer, nil
}

func (m *Module) SetSigningEnabled(ctx context.Context, enabled bool, sender string) error {
	// Disabling remains available even before a certificate has been uploaded
	if !enabled {
		return m.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.Assignments(map[string]any{"enabled": false}),
		}).Create(&SigningSettings{ID: 1}).Error
	}
	query := m.db.WithContext(ctx).Model(&SigningSettings{}).Where("id = ?", 1)
	settings, err := m.loadSigningSettings(ctx)
	if err != nil {
		return err
	}
	if _, _, err := signerFromSettings(settings, sender); err != nil {
		return err
	}
	// Refuse activation if a simultaneous upload or removal changed the certificate after validation
	query = query.Where("certificate_chain = ?", settings.CertificateChain)
	result := query.Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return signingError("The certificate changed; reload the settings and try again")
	}
	return nil
}

func (m *Module) RemoveSigningCertificate(ctx context.Context) error {
	// Removal also disables signing in the same write so no enabled configuration can lose its key
	return m.db.WithContext(ctx).Model(&SigningSettings{}).Where("id = ?", 1).Updates(map[string]any{
		"enabled": false, "certificate_chain": "", "private_key": "",
	}).Error
}

func detachedSignature(content []byte, certs []*x509.Certificate, signer crypto.Signer) ([]byte, error) {
	signed, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, err
	}
	signed.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := signed.AddSignerChain(certs[0], signer, certs[1:], pkcs7.SignerInfoConfig{}); err != nil {
		return nil, err
	}
	signed.Detach()
	return signed.Finish()
}

func signMessage(payload []byte, certs []*x509.Certificate, signer crypto.Signer) ([]byte, error) {
	// Separate transport headers from the exact MIME entity whose bytes the recipient will verify
	parts := bytes.SplitN(payload, []byte("\r\n\r\n"), 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid MIME message")
	}
	var outer, inner bytes.Buffer
	contentHeader := false
	for _, line := range bytes.Split(parts[0], []byte("\r\n")) {
		// Continuation lines belong to the preceding header and must retain their exact signed bytes
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			contentHeader = bytes.HasPrefix(bytes.ToLower(line), []byte("content-"))
		}
		if contentHeader {
			inner.Write(line)
			inner.WriteString("\r\n")
		} else {
			outer.Write(line)
			outer.WriteString("\r\n")
		}
	}
	inner.WriteString("\r\n")
	inner.Write(parts[1])
	signature, err := detachedSignature(inner.Bytes(), certs, signer)
	if err != nil {
		return nil, err
	}

	// Build multipart/signed without re-encoding the signed entity or including its delimiter CRLF in the signature
	boundary := multipart.NewWriter(&bytes.Buffer{}).Boundary()
	outer.WriteString("Content-Type: " + mime.FormatMediaType("multipart/signed", map[string]string{
		"boundary": boundary, "protocol": "application/pkcs7-signature", "micalg": "sha-256",
	}) + "\r\n\r\n")
	fmt.Fprintf(&outer, "--%s\r\n", boundary)
	outer.Write(inner.Bytes())
	fmt.Fprintf(&outer, "\r\n--%s\r\nContent-Type: application/pkcs7-signature; name=\"smime.p7s\"\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"smime.p7s\"\r\n\r\n", boundary)
	encoded := base64.StdEncoding.EncodeToString(signature)
	for len(encoded) > 76 {
		outer.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	outer.WriteString(encoded)
	fmt.Fprintf(&outer, "\r\n--%s--\r\n", boundary)
	return outer.Bytes(), nil
}
