//go:build e2etest

package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"path"
	"time"

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
	appLockService   *AppLockService
	externalIdPKey   jwk.Key
}

func NewTestService(db *gorm.DB, appConfigService *AppConfigService, jwtService *JwtService, ldapService *LdapService, appLockService *AppLockService, fileStorage storage.FileStorage) (*TestService, error) {
	s := &TestService{
		db:               db,
		appConfigService: appConfigService,
		jwtService:       jwtService,
		ldapService:      ldapService,
		appLockService:   appLockService,
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

		publicKeyPasskey1, _ := base64.StdEncoding.DecodeString("pQMmIAEhWCDBw6jkpXXr0pHrtAQetxiR5cTcILG/YGDCdKrhVhNDHCJYIIu12YrF6B7Frwl3AUqEpdrYEwj3Fo3XkGgvrBIJEUmGAQI=")
		publicKeyPasskey2, _ := base64.StdEncoding.DecodeString("pSJYIPmc+FlEB0neERqqscxKckGF8yq1AYrANiloshAUAouHAQIDJiABIVggj4qA0PrZzg8Co1C27nyUbzrp8Ewjr7eOlGI2LfrzmbI=")
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
			CredentialParams: model.CredentialParameters{
				{Type: "public-key", Algorithm: -7},
				{Type: "public-key", Algorithm: -257},
			},
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
			UserID:    users[0].ID,
			ExpiresAt: datatype.DateTime(time.Now().Add(30 * 24 * time.Hour)),
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

		keyValues := []model.KV{
			{
				Key:   jwkutils.PrivateKeyDBKey,
				Value: utils.Ptr("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"),
			},
		}

		for _, kv := range keyValues {
			if err := tx.Create(&kv).Error; err != nil {
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

	// Manually set instance ID
	err = s.appConfigService.UpdateAppConfigValues(ctx, "instanceId", "test-instance-id")
	if err != nil {
		return err
	}

	// Reload the app config from the database after resetting the values
	return s.appConfigService.LoadDbConfig(ctx)
}

func (s *TestService) ResetLock(ctx context.Context) error {
	_, err := s.appLockService.Acquire(ctx, true)
	return err
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
			"ldapAdminGroupName":                 "admin_group",
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
