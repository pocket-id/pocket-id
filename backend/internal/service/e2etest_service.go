//go:build e2etest

package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	"github.com/pocket-id/pocket-id/backend/resources"
)

type TestService struct {
	db               *gorm.DB
	jwtService       *JwtService
	appConfigService *AppConfigService
	ldapService      *LdapService
	fileStorage      storage.FileStorage
	externalIdPKey   jwk.Key
}

func NewTestService(db *gorm.DB, appConfigService *AppConfigService, jwtService *JwtService, ldapService *LdapService, fileStorage storage.FileStorage) (*TestService, error) {
	s := &TestService{
		db:               db,
		appConfigService: appConfigService,
		jwtService:       jwtService,
		ldapService:      ldapService,
		fileStorage:      fileStorage,
	}
	err := s.initExternalIdP()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize external IdP: %w", err)
	}
	return s, nil
}

// Initializes the "external IdP"
// This creates a new "issuing authority" containing a public JWKS
// It also stores the private key internally that will be used to issue JWTs
func (s *TestService) initExternalIdP() error {
	// Generate a new ECDSA key
	rawKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	s.externalIdPKey, err = jwkutils.ImportRawKey(rawKey, jwa.ES256().String(), "")
	if err != nil {
		return fmt.Errorf("failed to import private key: %w", err)
	}

	return nil
}

//nolint:gocognit
func (s *TestService) SeedDatabase(baseURL string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		users := []model.User{
			{
				Base: model.Base{
					ID: "f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e",
				},
				Username:    "tim",
				Email:       utils.Ptr("tim.cook@test.com"),
				FirstName:   "Tim",
				LastName:    "Cook",
				DisplayName: "Tim Cook",
				IsAdmin:     true,
			},
			{
				Base: model.Base{
					ID: "1cd19686-f9a6-43f4-a41f-14a0bf5b4036",
				},
				Username:    "craig",
				Email:       utils.Ptr("craig.federighi@test.com"),
				FirstName:   "Craig",
				LastName:    "Federighi",
				DisplayName: "Craig Federighi",
				IsAdmin:     false,
			},
		}
		for _, user := range users {
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		}

		oneTimeAccessTokens := []model.OneTimeAccessToken{{
			Base: model.Base{
				ID: "bf877753-4ea4-4c9c-bbbd-e198bb201cb8",
			},
			Token:     "HPe6k6uiDRRVuAQV",
			ExpiresAt: datatype.DateTime(time.Now().Add(1 * time.Hour)),
			UserID:    users[0].ID,
		},
			{
				Base: model.Base{
					ID: "d3afae24-fe2d-4a98-abec-cf0b8525096a",
				},
				Token:     "YCGDtftvsvYWiXd0",
				ExpiresAt: datatype.DateTime(time.Now().Add(-1 * time.Second)), // expired
				UserID:    users[0].ID,
			},
		}
		for _, token := range oneTimeAccessTokens {
			if err := tx.Create(&token).Error; err != nil {
				return err
			}
		}

		userGroups := []model.UserGroup{
			{
				Base: model.Base{
					ID: "c7ae7c01-28a3-4f3c-9572-1ee734ea8368",
				},
				Name:         "developers",
				FriendlyName: "Developers",
				Users:        []model.User{users[0], users[1]},
			},
			{
				Base: model.Base{
					ID: "adab18bf-f89d-4087-9ee1-70ff15b48211",
				},
				Name:         "designers",
				FriendlyName: "Designers",
				Users:        []model.User{users[0]},
			},
		}
		for _, group := range userGroups {
			if err := tx.Create(&group).Error; err != nil {
				return err
			}
		}

		oidcClients := []model.OidcClient{
			{
				Base: model.Base{
					ID: "3654a746-35d4-4321-ac61-0bdcff2b4055",
				},
				Name:               "Nextcloud",
				LaunchURL:          utils.Ptr("https://nextcloud.local"),
				Secret:             "$2a$10$9dypwot8nGuCjT6wQWWpJOckZfRprhe2EkwpKizxS/fpVHrOLEJHC", // w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY
				CallbackURLs:       model.UrlList{"http://nextcloud/auth/callback"},
				LogoutCallbackURLs: model.UrlList{"http://nextcloud/auth/logout/callback"},
				ImageType:          utils.StringPointer("png"),
				CreatedByID:        utils.Ptr(users[0].ID),
			},
			{
				Base: model.Base{
					ID: "606c7782-f2b1-49e5-8ea9-26eb1b06d018",
				},
				Name:         "Immich",
				Secret:       "$2a$10$Ak.FP8riD1ssy2AGGbG.gOpnp/rBpymd74j0nxNMtW0GG1Lb4gzxe", // PYjrE9u4v9GVqXKi52eur0eb2Ci4kc0x
				CallbackURLs: model.UrlList{"http://immich/auth/callback"},
				CreatedByID:  utils.Ptr(users[1].ID),
				AllowedUserGroups: []model.UserGroup{
					userGroups[1],
				},
			},
			{
				Base: model.Base{
					ID: "7c21a609-96b5-4011-9900-272b8d31a9d1",
				},
				Name:               "Tailscale",
				Secret:             "$2a$10$xcRReBsvkI1XI6FG8xu/pOgzeF00bH5Wy4d/NThwcdi3ZBpVq/B9a", // n4VfQeXlTzA6yKpWbR9uJcMdSx2qH0Lo
				CallbackURLs:       model.UrlList{"http://tailscale/auth/callback"},
				LogoutCallbackURLs: model.UrlList{"http://tailscale/auth/logout/callback"},
				CreatedByID:        utils.Ptr(users[0].ID),
			},
			{
				Base: model.Base{
					ID: "c48232ff-ff65-45ed-ae96-7afa8a9b443b",
				},
				Name:              "Federated",
				Secret:            "$2a$10$Ak.FP8riD1ssy2AGGbG.gOpnp/rBpymd74j0nxNMtW0GG1Lb4gzxe", // PYjrE9u4v9GVqXKi52eur0eb2Ci4kc0x
				CallbackURLs:      model.UrlList{"http://federated/auth/callback"},
				CreatedByID:       utils.Ptr(users[1].ID),
				AllowedUserGroups: []model.UserGroup{},
				Credentials: model.OidcClientCredentials{
					FederatedIdentities: []model.OidcClientFederatedIdentity{
						{
							Issuer:   "https://external-idp.local",
							Audience: "api://PocketID",
							Subject:  "c48232ff-ff65-45ed-ae96-7afa8a9b443b",
							JWKS:     baseURL + "/api/externalidp/jwks.json",
						},
					},
				},
			},
		}
		for _, client := range oidcClients {
			if err := tx.Create(&client).Error; err != nil {
				return err
			}
		}

		authCodes := []model.OidcAuthorizationCode{
			{
				Code:      "auth-code",
				Scope:     "openid profile",
				Nonce:     "nonce",
				ExpiresAt: datatype.DateTime(time.Now().Add(1 * time.Hour)),
				UserID:    users[0].ID,
				ClientID:  oidcClients[0].ID,
			},
			{
				Code:      "federated",
				Scope:     "openid profile",
				Nonce:     "nonce",
				ExpiresAt: datatype.DateTime(time.Now().Add(1 * time.Hour)),
				UserID:    users[1].ID,
				ClientID:  oidcClients[2].ID,
			},
		}
		for _, authCode := range authCodes {
			if err := tx.Create(&authCode).Error; err != nil {
				return err
			}
		}

		refreshToken := model.OidcRefreshToken{
			Token:     utils.CreateSha256Hash("ou87UDg249r1StBLYkMEqy9TXDbV5HmGuDpMcZDo"),
			ExpiresAt: datatype.DateTime(time.Now().Add(24 * time.Hour)),
			Scope:     "openid profile email",
			UserID:    users[0].ID,
			ClientID:  oidcClients[0].ID,
		}
		if err := tx.Create(&refreshToken).Error; err != nil {
			return err
		}

		accessToken := model.OneTimeAccessToken{
			Token:     "one-time-token",
			ExpiresAt: datatype.DateTime(time.Now().Add(1 * time.Hour)),
			UserID:    users[0].ID,
		}
		if err := tx.Create(&accessToken).Error; err != nil {
			return err
		}

		userAuthorizedClients := []model.UserAuthorizedOidcClient{
			{
				Scope:      "openid profile email",
				UserID:     users[0].ID,
				ClientID:   oidcClients[0].ID,
				LastUsedAt: datatype.DateTime(time.Date(2025, 8, 1, 13, 0, 0, 0, time.UTC)),
			},
			{
				Scope:      "openid profile email",
				UserID:     users[0].ID,
				ClientID:   oidcClients[2].ID,
				LastUsedAt: datatype.DateTime(time.Date(2025, 8, 10, 14, 0, 0, 0, time.UTC)),
			},
			{
				Scope:      "openid profile email",
				UserID:     users[1].ID,
				ClientID:   oidcClients[3].ID,
				LastUsedAt: datatype.DateTime(time.Date(2025, 8, 12, 12, 0, 0, 0, time.UTC)),
			},
		}
		for _, userAuthorizedClient := range userAuthorizedClients {
			if err := tx.Create(&userAuthorizedClient).Error; err != nil {
				return err
			}
		}

		// To generate a new key pair, run the following command:
		// openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 | \
		// openssl pkcs8 -topk8 -nocrypt | tee >(openssl pkey -pubout)

		publicKeyPasskey1, _ := s.getCborPublicKey("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwcOo5KV169KR67QEHrcYkeXE3CCxv2BgwnSq4VYTQxyLtdmKxegexa8JdwFKhKXa2BMI9xaN15BoL6wSCRFJhg==")
		publicKeyPasskey2, _ := s.getCborPublicKey("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEj4qA0PrZzg8Co1C27nyUbzrp8Ewjr7eOlGI2LfrzmbL5nPhZRAdJ3hEaqrHMSnJBhfMqtQGKwDYpaLIQFAKLhw==")
		webauthnCredentials := []model.WebauthnCredential{
			{
				Name:            "Passkey 1",
				CredentialID:    []byte("test-credential-tim"),
				PublicKey:       publicKeyPasskey1,
				AttestationType: "none",
				Transport:       model.AuthenticatorTransportList{protocol.Internal},
				UserID:          users[0].ID,
			},
			{
				Name:            "Passkey 2",
				CredentialID:    []byte("test-credential-craig"),
				PublicKey:       publicKeyPasskey2,
				AttestationType: "none",
				Transport:       model.AuthenticatorTransportList{protocol.Internal},
				UserID:          users[1].ID,
			},
		}
		for _, credential := range webauthnCredentials {
			if err := tx.Create(&credential).Error; err != nil {
				return err
			}
		}

		webauthnSession := model.WebauthnSession{
			Challenge:        "challenge",
			ExpiresAt:        datatype.DateTime(time.Now().Add(1 * time.Hour)),
			UserVerification: "preferred",
		}
		if err := tx.Create(&webauthnSession).Error; err != nil {
			return err
		}

		apiKey := model.ApiKey{
			Base: model.Base{
				ID: "5f1fa856-c164-4295-961e-175a0d22d725",
			},
			Name:      "Test API Key",
			Key:       "6c34966f57ef2bb7857649aff0e7ab3ad67af93c846342ced3f5a07be8706c20",
			ExpiresAt: datatype.DateTime(time.Now().Add(24 * time.Hour)),
			UserID:    users[0].ID,
		}
		if err := tx.Create(&apiKey).Error; err != nil {
			return err
		}

		signupTokens := []model.SignupToken{
			{
				Base: model.Base{
					ID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				},
				Token:      "VALID1234567890A",
				ExpiresAt:  datatype.DateTime(time.Now().Add(24 * time.Hour)),
				UsageLimit: 1,
				UsageCount: 0,
			},
			{
				Base: model.Base{
					ID: "dc3c9c96-714e-48eb-926e-2d7c7858e6cf",
				},
				Token:      "PARTIAL567890ABC",
				ExpiresAt:  datatype.DateTime(time.Now().Add(7 * 24 * time.Hour)),
				UsageLimit: 5,
				UsageCount: 2,
			},
			{
				Base: model.Base{
					ID: "44de1863-ffa5-4db1-9507-4887cd7a1e3f",
				},
				Token:      "EXPIRED34567890B",
				ExpiresAt:  datatype.DateTime(time.Now().Add(-24 * time.Hour)), // Expired
				UsageLimit: 3,
				UsageCount: 1,
			},
			{
				Base: model.Base{
					ID: "f1b1678b-7720-4d8b-8f91-1dbff1e2d02b",
				},
				Token:      "FULLYUSED567890C",
				ExpiresAt:  datatype.DateTime(time.Now().Add(24 * time.Hour)),
				UsageLimit: 1,
				UsageCount: 1, // Usage limit reached
			},
		}
		for _, token := range signupTokens {
			if err := tx.Create(&token).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *TestService) ResetDatabase() error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var tables []string

		switch common.EnvConfig.DbProvider {
		case common.DbProviderSqlite:
			// Query to get all tables for SQLite
			if err := tx.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations';").Scan(&tables).Error; err != nil {
				return err
			}
		case common.DbProviderPostgres:
			// Query to get all tables for PostgreSQL
			if err := tx.Raw(`
                SELECT tablename
                FROM pg_tables
                WHERE schemaname = 'public' AND tablename != 'schema_migrations';
            `).Scan(&tables).Error; err != nil {
				return err
			}
		case common.DbProviderMysql:
			// Query to get all tables for MySQL
			if err := tx.Raw(`
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = DATABASE()
                AND table_name != 'schema_migrations';
            `).Scan(&tables).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
		}

		// Delete all rows from all tables
		for _, table := range tables {
			if err := tx.Exec(fmt.Sprintf("DELETE FROM %s;", table)).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (s *TestService) ResetApplicationImages(ctx context.Context) error {
	if err := s.fileStorage.DeleteAll(ctx, "/"); err != nil {
		slog.ErrorContext(ctx, "Error removing uploads", slog.Any("error", err))
		return err
	}

	files, err := resources.FS.ReadDir("images")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		srcFilePath := path.Join("images", file.Name())
		srcFile, err := resources.FS.Open(srcFilePath)
		if err != nil {
			return err
		}
		if err := s.fileStorage.Save(ctx, path.Join("application-images", file.Name()), srcFile); err != nil {
			srcFile.Close()
			return err
		}
		srcFile.Close()
	}

	return nil
}

func (s *TestService) ResetAppConfig(ctx context.Context) error {
	// Reset all app config variables to their default values in the database
	err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&model.AppConfigVariable{}).Update("value", "").Error
	if err != nil {
		return err
	}

	// Reload the app config from the database after resetting the values
	return s.appConfigService.LoadDbConfig(ctx)
}

func (s *TestService) SetJWTKeys() {
	const privateKeyString = `{"alg":"RS256","d":"mvMDWSdPPvcum0c0iEHE2gbqtV2NKMmLwrl9E6K7g8lTV95SePLnW_bwyMPV7EGp7PQk3l17I5XRhFjze7GqTnFIOgKzMianPs7jv2ELtBMGK0xOPATgu1iGb70xZ6vcvuEfRyY3dJ0zr4jpUdVuXwKmx9rK4IdZn2dFCKfvSuspqIpz11RhF1ALrqDLkxGVv7ZwNh0_VhJZU9hcjG5l6xc7rQEKpPRkZp0IdjkGS8Z0FskoVaiRIWAbZuiVFB9WCW8k1czC4HQTPLpII01bUQx2ludbm0UlXRgVU9ptUUbU7GAImQqTOW8LfPGklEvcgzlIlR_oqw4P9yBxLi-yMQ","dp":"pvNCSnnhbo8Igw9psPR-DicxFnkXlu_ix4gpy6efTrxA-z1VDFDioJ814vKQNioYDzpyAP1gfMPhRkvG_q0hRZsJah3Sb9dfA-WkhSWY7lURQP4yIBTMU0PF_rEATuS7lRciYk1SOx5fqXZd3m_LP0vpBC4Ujlq6NAq6CIjCnms","dq":"TtUVGCCkPNgfOLmkYXu7dxxUCV5kB01-xAEK2OY0n0pG8vfDophH4_D_ZC7nvJ8J9uDhs_3JStexq1lIvaWtG99RNTChIEDzpdn6GH9yaVcb_eB4uJjrNm64FhF8PGCCwxA-xMCZMaARKwhMB2_IOMkxUbWboL3gnhJ2rDO_QO0","e":"AQAB","kid":"8uHDw3M6rf8","kty":"RSA","n":"yaeEL0VKoPBXIAaWXsUgmu05lAvEIIdJn0FX9lHh4JE5UY9B83C5sCNdhs9iSWzpeP11EVjWp8i3Yv2CF7c7u50BXnVBGtxpZpFC-585UXacoJ0chUmarL9GRFJcM1nPHBTFu68aRrn1rIKNHUkNaaxFo0NFGl_4EDDTO8HwawTjwkPoQlRzeByhlvGPVvwgB3Fn93B8QJ_cZhXKxJvjjrC_8Pk76heC_ntEMru71Ix77BoC3j2TuyiN7m9RNBW8BU5q6lKoIdvIeZfTFLzi37iufyfvMrJTixp9zhNB1NxlLCeOZl2MXegtiGqd2H3cbAyqoOiv9ihUWTfXj7SxJw","p":"_Yylc9e07CKdqNRD2EosMC2mrhrEa9j5oY_l00Qyy4-jmCA59Q9viyqvveRo0U7cRvFA5BWgWN6GGLh1DG3X-QBqVr0dnk3uzbobb55RYUXyPLuBZI2q6w2oasbiDwPdY7KpkVv_H-bpITQlyDvO8hhucA6rUV7F6KTQVz8M3Ms","q":"y5p3hch-7jJ21TkAhp_Vk1fLCAuD4tbErwQs2of9ja8sB4iJOs5Wn6HD3P7Mc8Plye7qaLHvzc8I5g0tPKWvC0DPd_FLPXiWwMVAzee3NUX_oGeJNOQp11y1w_KqdO9qZqHSEPZ3NcFL_SZMFgggxhM1uzRiPzsVN0lnD_6prZU","qi":"2Grt6uXHm61ji3xSdkBWNtUnj19vS1-7rFJp5SoYztVQVThf_W52BAiXKBdYZDRVoItC_VS2NvAOjeJjhYO_xQ_q3hK7MdtuXfEPpLnyXKkmWo3lrJ26wbeF6l05LexCkI7ShsOuSt-dsyaTJTszuKDIA6YOfWvfo3aVZmlWRaI","use":"sig"}`

	privateKey, _ := jwk.ParseKey([]byte(privateKeyString))
	_ = s.jwtService.SetKey(privateKey)
}

// getCborPublicKey decodes a Base64 encoded public key and returns the CBOR encoded COSE key
func (s *TestService) getCborPublicKey(base64PublicKey string) ([]byte, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(base64PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}

	coseKey := map[int]interface{}{
		1:  2,                     // Key type: EC2
		3:  -7,                    // Algorithm: ECDSA with SHA-256
		-1: 1,                     // Curve: P-256
		-2: ecdsaPubKey.X.Bytes(), // X coordinate
		-3: ecdsaPubKey.Y.Bytes(), // Y coordinate
	}

	cborPublicKey, err := cbor.Marshal(coseKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal COSE key: %w", err)
	}

	return cborPublicKey, nil
}

// SyncLdap triggers an LDAP synchronization
func (s *TestService) SyncLdap(ctx context.Context) error {
	return s.ldapService.SyncAll(ctx)
}

// SetLdapTestConfig writes the test LDAP config variables directly to the database.
func (s *TestService) SetLdapTestConfig(ctx context.Context) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ldapConfigs := map[string]string{
			"ldapUrl":                            "ldap://lldap:3890",
			"ldapBindDn":                         "uid=admin,ou=people,dc=pocket-id,dc=org",
			"ldapBindPassword":                   "admin_password",
			"ldapBase":                           "dc=pocket-id,dc=org",
			"ldapUserSearchFilter":               "(objectClass=person)",
			"ldapUserGroupSearchFilter":          "(objectClass=groupOfNames)",
			"ldapSkipCertVerify":                 "true",
			"ldapAttributeUserUniqueIdentifier":  "uuid",
			"ldapAttributeUserUsername":          "uid",
			"ldapAttributeUserEmail":             "mail",
			"ldapAttributeUserFirstName":         "givenName",
			"ldapAttributeUserLastName":          "sn",
			"ldapAttributeGroupUniqueIdentifier": "uuid",
			"ldapAttributeGroupName":             "uid",
			"ldapAttributeGroupMember":           "member",
			"ldapAttributeAdminGroup":            "admin_group",
			"ldapSoftDeleteUsers":                "true",
			"ldapEnabled":                        "true",
		}

		for key, value := range ldapConfigs {
			configVar := model.AppConfigVariable{Key: key, Value: value}
			if err := tx.Create(&configVar).Error; err != nil {
				return fmt.Errorf("failed to create config variable '%s': %w", key, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to set LDAP test config: %w", err)
	}

	if err := s.appConfigService.LoadDbConfig(ctx); err != nil {
		return fmt.Errorf("failed to load app config: %w", err)
	}

	return nil
}

func (s *TestService) SignRefreshToken(userID, clientID, refreshToken string) (string, error) {
	return s.jwtService.GenerateOAuthRefreshToken(userID, clientID, refreshToken)
}

// GetExternalIdPJWKS returns the JWKS for the "external IdP".
func (s *TestService) GetExternalIdPJWKS() (jwk.Set, error) {
	pubKey, err := s.externalIdPKey.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	set := jwk.NewSet()
	err = set.AddKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to add public key to set: %w", err)
	}

	return set, nil
}

func (s *TestService) SignExternalIdPToken(iss, sub, aud string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(sub).
		Expiration(now.Add(time.Hour)).
		IssuedAt(now).
		Issuer(iss).
		Audience([]string{aud}).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	alg, _ := s.externalIdPKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.externalIdPKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}
