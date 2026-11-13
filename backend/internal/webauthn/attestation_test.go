package webauthn

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/metadata/providers/memory"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestVerifyRegistrationAttestation(t *testing.T) {
	authenticator := newTestAuthenticator(t)
	certifiedL1 := []metadata.StatusReport{{Status: metadata.FidoCertifiedL1}}
	tomorrow := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name         string
		attestation  appconfig.AppConfigValue
		format       string
		listedAAGUID uuid.UUID
		// listedRoot defaults to the test authenticator's root
		listedRoot *x509.Certificate
		reports    []metadata.StatusReport
		wantCode   apperror.Code
		// wantVerified is only checked when the registration succeeds
		wantVerified bool
	}{
		{
			name:         "not required accepts a passkey without attestation",
			attestation:  "none",
			format:       "none",
			listedAAGUID: authenticator.aaguid,
			reports:      certifiedL1,
		},
		{
			name:         "accepts a listed authenticator at the required level",
			attestation:  "l1",
			format:       "packed",
			listedAAGUID: authenticator.aaguid,
			reports:      certifiedL1,
			wantVerified: true,
		},
		{
			name:         "rejects a passkey without attestation",
			attestation:  "verified",
			format:       "none",
			listedAAGUID: authenticator.aaguid,
			reports:      certifiedL1,
			wantCode:     apperror.CodePasskeyAttestationRequired,
		},
		{
			name:         "rejects an authenticator the metadata doesn't list",
			attestation:  "verified",
			format:       "packed",
			listedAAGUID: uuid.New(),
			reports:      certifiedL1,
			wantCode:     apperror.CodeAuthenticatorNotCertified,
		},
		{
			name:         "rejects an authenticator below the required level",
			attestation:  "l2",
			format:       "packed",
			listedAAGUID: authenticator.aaguid,
			reports:      certifiedL1,
			wantCode:     apperror.CodeAuthenticatorNotCertified,
		},
		{
			name:         "rejects a listed AAGUID whose certificate doesn't chain to the listed roots",
			attestation:  "verified",
			format:       "packed",
			listedAAGUID: authenticator.aaguid,
			listedRoot:   newTestAuthenticator(t).root,
			reports:      certifiedL1,
			wantCode:     apperror.CodeAuthenticatorNotCertified,
		},
		{
			name:         "rejects a revoked authenticator",
			attestation:  "verified",
			format:       "packed",
			listedAAGUID: authenticator.aaguid,
			reports:      []metadata.StatusReport{{Status: metadata.FidoCertifiedL2}, {Status: metadata.Revoked}},
			wantCode:     apperror.CodeAuthenticatorNotCertified,
		},
		{
			name:         "rejects a certification that only takes effect later",
			attestation:  "l2",
			format:       "packed",
			listedAAGUID: authenticator.aaguid,
			reports:      []metadata.StatusReport{{Status: metadata.FidoCertifiedL1}, {Status: metadata.FidoCertifiedL2, EffectiveDate: &tomorrow}},
			wantCode:     apperror.CodeAuthenticatorNotCertified,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			user := model.User{Base: model.Base{ID: "attested-user"}, Username: "attested-user"}
			require.NoError(t, db.Create(&user).Error)

			service, err := newService(Dependencies{DB: db, AppURL: "https://example.com", AuditLog: noopAuditLogger{}})
			require.NoError(t, err)

			// The metadata lists one authenticator model, by default with the test authenticator's root
			listedRoot := authenticator.root
			if tc.listedRoot != nil {
				listedRoot = tc.listedRoot
			}
			provider, err := memory.New(memory.WithMetadata(map[uuid.UUID]*metadata.Entry{
				tc.listedAAGUID: {
					AaGUID: tc.listedAAGUID,
					MetadataStatement: metadata.Statement{
						AttestationTypes:            metadata.AuthenticatorAttestationTypes{metadata.BasicFull},
						AttestationRootCertificates: []*x509.Certificate{listedRoot},
					},
					StatusReports: tc.reports,
				},
			}))
			require.NoError(t, err)
			service.fidoMetadata = &fidoMetadata{provider: provider, refreshAt: time.Now().Add(time.Hour)}

			dbConfig := &appconfig.AppConfigModel{
				AppName:                         "Pocket ID",
				WebauthnUserVerification:        "required",
				WebauthnAllowSyncedPasskeys:     "true",
				WebauthnAuthenticatorAttachment: "any",
				WebauthnAttestation:             tc.attestation,
			}
			registration, err := service.BeginRegistration(t.Context(), dbConfig, user.ID)
			require.NoError(t, err)

			response := authenticator.registrationResponse(t, registration.Response.Challenge, tc.format)
			credential, err := service.VerifyRegistration(t.Context(), dbConfig, registration.SessionID, user.ID, response, "127.0.0.1", "")

			if tc.wantCode != "" {
				require.Error(t, err)
				assert.True(t, apperror.IsCode(err, tc.wantCode), "unexpected error: %v", err)

				var count int64
				require.NoError(t, db.Model(&model.WebauthnCredential{}).Count(&count).Error)
				assert.Zero(t, count)
				return
			}
			require.NoError(t, err)

			var stored model.WebauthnCredential
			require.NoError(t, db.First(&stored, "id = ?", credential.ID).Error)
			assert.Equal(t, tc.wantVerified, stored.AttestationVerified)
		})
	}
}

func TestFidoMetadataWaitsBeforeRetryingAFailedDownload(t *testing.T) {
	var requests atomic.Int32
	m := &fidoMetadata{
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			requests.Add(1)
			return nil, errors.New("too many requests")
		})},
		url:  "https://mds.example.com",
		path: t.TempDir() + "/fido-mds.jwt",
	}

	_, err := m.get()
	require.Error(t, err)
	_, err = m.get()
	require.ErrorContains(t, err, "too many requests", "the error while waiting should still name the failed download")

	assert.Equal(t, int32(1), requests.Load())
}

func TestFidoMetadataReplacesAnUnreadableCache(t *testing.T) {
	var requests atomic.Int32
	m := &fidoMetadata{
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			requests.Add(1)
			return nil, errors.New("offline")
		})},
		url:  "https://mds.example.com",
		path: t.TempDir() + "/fido-mds.jwt",
	}
	require.NoError(t, os.WriteFile(m.path, []byte("not a jwt"), 0600))

	// The unreadable blob is removed without a download, because the provider fails on it before downloading
	_, err := m.get()
	require.Error(t, err)
	assert.NoFileExists(t, m.path)
	assert.Zero(t, requests.Load())

	// The next registration downloads a new blob instead of waiting for the retry interval
	_, err = m.get()
	require.ErrorContains(t, err, "offline")
	assert.Equal(t, int32(1), requests.Load())
}

func TestFidoMetadataRefreshesInTheBackground(t *testing.T) {
	release := make(chan struct{})
	var requests atomic.Int32
	stale, err := memory.New(memory.WithMetadata(map[uuid.UUID]*metadata.Entry{}))
	require.NoError(t, err)
	m := &fidoMetadata{
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			requests.Add(1)
			<-release
			return nil, errors.New("offline")
		})},
		url:      "https://mds.example.com",
		path:     t.TempDir() + "/fido-mds.jwt",
		provider: stale,
	}

	// Registrations get the outdated metadata without waiting while a single refresh is still downloading
	for range 2 {
		provider, err := m.get()
		require.NoError(t, err)
		assert.Same(t, stale, provider)
	}

	// The failed refresh keeps the outdated metadata
	close(release)
	require.Eventually(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return !m.refreshing
	}, time.Second, 10*time.Millisecond)
	assert.Same(t, stale, m.provider)
	assert.Equal(t, int32(1), requests.Load())
}

func TestFidoMetadataNeverDownloadsWithoutAURL(t *testing.T) {
	var requests atomic.Int32
	m := &fidoMetadata{
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			requests.Add(1)
			return nil, errors.New("offline")
		})},
		path: t.TempDir() + "/fido-mds.jwt",
	}

	_, err := m.get()
	require.ErrorContains(t, err, "downloads are disabled")
	assert.Zero(t, requests.Load())
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// noopAuditLogger only implements the audit event a registration writes
type noopAuditLogger struct{ AuditLogger }

func (noopAuditLogger) Create(context.Context, auditlogs.Event, string, string, string, auditlogs.Data, *gorm.DB) (auditlogs.AuditLog, bool) {
	return auditlogs.AuditLog{}, true
}

// testAuthenticator fabricates registration responses with a packed attestation, signed by an attestation certificate from its own root
type testAuthenticator struct {
	aaguid  uuid.UUID
	root    *x509.Certificate
	attCert []byte
	attKey  *ecdsa.PrivateKey
}

func newTestAuthenticator(t *testing.T) testAuthenticator {
	t.Helper()

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Attestation Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	require.NoError(t, err)
	root, err := x509.ParseCertificate(rootDER)
	require.NoError(t, err)

	// Packed attestation requires these subject fields on the attestation certificate
	attKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	attTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Country:            []string{"DE"},
			Organization:       []string{"Pocket ID"},
			OrganizationalUnit: []string{"Authenticator Attestation"},
			CommonName:         "Test Authenticator",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
	}
	attCert, err := x509.CreateCertificate(rand.Reader, attTemplate, root, &attKey.PublicKey, rootKey)
	require.NoError(t, err)

	return testAuthenticator{aaguid: uuid.New(), root: root, attCert: attCert, attKey: attKey}
}

// registrationResponse answers the challenge with a new credential, attested in the given format
func (a testAuthenticator) registrationResponse(t *testing.T, challenge protocol.URLEncodedBase64, format string) *protocol.ParsedCredentialCreationData {
	t.Helper()

	// Build the authenticator data with a new ES256 credential, user presence and user verification
	credentialKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKey, err := credentialKey.PublicKey.ECDH()
	require.NoError(t, err)
	point := publicKey.Bytes()
	coseKey, err := webauthncbor.Marshal(webauthncose.EC2PublicKeyData{
		PublicKeyData: webauthncose.PublicKeyData{KeyType: int64(webauthncose.EllipticKey), Algorithm: int64(webauthncose.AlgES256)},
		Curve:         int64(webauthncose.P256),
		XCoord:        point[1:33],
		YCoord:        point[33:],
	})
	require.NoError(t, err)
	credentialUUID := uuid.New()
	credentialID := credentialUUID[:]
	rpIDHash := sha256.Sum256([]byte("example.com"))
	authData := slices.Concat(rpIDHash[:], []byte{0x45, 0, 0, 0, 0}, a.aaguid[:], []byte{0, 16}, credentialID, coseKey)

	clientDataJSON, err := json.Marshal(map[string]string{
		"type":      "webauthn.create",
		"challenge": base64.RawURLEncoding.EncodeToString(challenge),
		"origin":    "https://example.com",
	})
	require.NoError(t, err)

	// Sign the authenticator data and client data hash with the attestation key
	attStmt := map[string]any{}
	if format == "packed" {
		clientDataHash := sha256.Sum256(clientDataJSON)
		digest := sha256.Sum256(slices.Concat(authData, clientDataHash[:]))
		sig, err := ecdsa.SignASN1(rand.Reader, a.attKey, digest[:])
		require.NoError(t, err)
		attStmt = map[string]any{"alg": int64(webauthncose.AlgES256), "sig": sig, "x5c": [][]byte{a.attCert}}
	}
	attestationObject, err := webauthncbor.Marshal(map[string]any{"fmt": format, "attStmt": attStmt, "authData": authData})
	require.NoError(t, err)

	encode := base64.RawURLEncoding.EncodeToString
	body, err := json.Marshal(map[string]any{
		"id":                      encode(credentialID),
		"rawId":                   encode(credentialID),
		"type":                    "public-key",
		"authenticatorAttachment": "cross-platform",
		"clientExtensionResults":  map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    encode(clientDataJSON),
			"attestationObject": encode(attestationObject),
			"transports":        []string{"usb"},
		},
	})
	require.NoError(t, err)

	parsed, err := protocol.ParseCredentialCreationResponseBytes(body)
	require.NoError(t, err)
	return parsed
}
