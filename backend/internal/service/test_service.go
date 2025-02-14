package service

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fxamacker/cbor/v2"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/resources"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type TestService struct {
	db               *gorm.DB
	jwtService       *JwtService
	appConfigService *AppConfigService
}

func NewTestService(db *gorm.DB, appConfigService *AppConfigService, jwtService *JwtService) *TestService {
	return &TestService{db: db, appConfigService: appConfigService, jwtService: jwtService}
}

func (s *TestService) SeedDatabase() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		users := []model.User{
			{
				Base: model.Base{
					ID: "f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e",
				},
				Username:  "tim",
				Email:     "tim.cook@test.com",
				FirstName: "Tim",
				LastName:  "Cook",
				IsAdmin:   true,
			},
			{
				Base: model.Base{
					ID: "1cd19686-f9a6-43f4-a41f-14a0bf5b4036",
				},
				Username:  "craig",
				Email:     "craig.federighi@test.com",
				FirstName: "Craig",
				LastName:  "Federighi",
				IsAdmin:   false,
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
				Secret:             "$2a$10$9dypwot8nGuCjT6wQWWpJOckZfRprhe2EkwpKizxS/fpVHrOLEJHC", // w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY
				CallbackURLs:       model.UrlList{"http://nextcloud/auth/callback"},
				LogoutCallbackURLs: model.UrlList{"http://nextcloud/auth/logout/callback"},
				ImageType:          utils.StringPointer("png"),
				CreatedByID:        users[0].ID,
			},
			{
				Base: model.Base{
					ID: "606c7782-f2b1-49e5-8ea9-26eb1b06d018",
				},
				Name:         "Immich",
				Secret:       "$2a$10$Ak.FP8riD1ssy2AGGbG.gOpnp/rBpymd74j0nxNMtW0GG1Lb4gzxe", // PYjrE9u4v9GVqXKi52eur0eb2Ci4kc0x
				CallbackURLs: model.UrlList{"http://immich/auth/callback"},
				CreatedByID:  users[1].ID,
				AllowedUserGroups: []model.UserGroup{
					userGroups[1],
				},
			},
		}
		for _, client := range oidcClients {
			if err := tx.Create(&client).Error; err != nil {
				return err
			}
		}

		authCode := model.OidcAuthorizationCode{
			Code:      "auth-code",
			Scope:     "openid profile",
			Nonce:     "nonce",
			ExpiresAt: datatype.DateTime(time.Now().Add(1 * time.Hour)),
			UserID:    users[0].ID,
			ClientID:  oidcClients[0].ID,
		}
		if err := tx.Create(&authCode).Error; err != nil {
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

		userAuthorizedClient := model.UserAuthorizedOidcClient{
			Scope:    "openid profile email",
			UserID:   users[0].ID,
			ClientID: oidcClients[0].ID,
		}
		if err := tx.Create(&userAuthorizedClient).Error; err != nil {
			return err
		}

		// To generate a new key pair, run the following command:
		// openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 | \
		// openssl pkcs8 -topk8 -nocrypt | tee >(openssl pkey -pubout)

		publicKeyPasskey1, err := s.getCborPublicKey("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwcOo5KV169KR67QEHrcYkeXE3CCxv2BgwnSq4VYTQxyLtdmKxegexa8JdwFKhKXa2BMI9xaN15BoL6wSCRFJhg==")
		publicKeyPasskey2, err := s.getCborPublicKey("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEj4qA0PrZzg8Co1C27nyUbzrp8Ewjr7eOlGI2LfrzmbL5nPhZRAdJ3hEaqrHMSnJBhfMqtQGKwDYpaLIQFAKLhw==")
		if err != nil {
			return err
		}
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

		return nil
	})
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

func (s *TestService) ResetApplicationImages() error {
	if err := os.RemoveAll(common.EnvConfig.UploadPath); err != nil {
		log.Printf("Error removing directory: %v", err)
		return err
	}

	files, err := resources.FS.ReadDir("images")
	if err != nil {
		return err
	}

	for _, file := range files {
		srcFilePath := filepath.Join("images", file.Name())
		destFilePath := filepath.Join(common.EnvConfig.UploadPath, "application-images", file.Name())

		err := utils.CopyEmbeddedFileToDisk(srcFilePath, destFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TestService) ResetAppConfig() error {
	// Reseed the config variables
	if err := s.appConfigService.InitDbConfig(); err != nil {
		return err
	}

	// Reset all app config variables to their default values
	if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&model.AppConfigVariable{}).Update("value", "").Error; err != nil {
		return err
	}

	// Reload the app config from the database after resetting the values
	return s.appConfigService.LoadDbConfigFromDb()
}

func (s *TestService) SetJWTKeys() {
	privateKeyString := `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAyaeEL0VKoPBXIAaWXsUgmu05lAvEIIdJn0FX9lHh4JE5UY9B
83C5sCNdhs9iSWzpeP11EVjWp8i3Yv2CF7c7u50BXnVBGtxpZpFC+585UXacoJ0c
hUmarL9GRFJcM1nPHBTFu68aRrn1rIKNHUkNaaxFo0NFGl/4EDDTO8HwawTjwkPo
QlRzeByhlvGPVvwgB3Fn93B8QJ/cZhXKxJvjjrC/8Pk76heC/ntEMru71Ix77BoC
3j2TuyiN7m9RNBW8BU5q6lKoIdvIeZfTFLzi37iufyfvMrJTixp9zhNB1NxlLCeO
Zl2MXegtiGqd2H3cbAyqoOiv9ihUWTfXj7SxJwIDAQABAoIBAQCa8wNZJ08+9y6b
RzSIQcTaBuq1XY0oyYvCuX0ToruDyVNX3lJ48udb9vDIw9XsQans9CTeXXsjldGE
WPN7sapOcUg6ArMyJqc+zuO/YQu0EwYrTE48BOC7WIZvvTFnq9y+4R9HJjd0nTOv
iOlR1W5fAqbH2srgh1mfZ0UIp+9K6ymoinPXVGEXUAuuoMuTEZW/tnA2HT9WEllT
2FyMbmXrFzutAQqk9GRmnQh2OQZLxnQWyShVqJEhYBtm6JUUH1YJbyTVzMLgdBM8
ukgjTVtRDHaW51ubRSVdGBVT2m1RRtTsYAiZCpM5bwt88aSUS9yDOUiVH+irDg/3
IHEuL7IxAoGBAP2MpXPXtOwinajUQ9hKLDAtpq4axGvY+aGP5dNEMsuPo5ggOfUP
b4sqr73kaNFO3EbxQOQVoFjehhi4dQxt1/kAala9HZ5N7s26G2+eUWFF8jy7gWSN
qusNqGrG4g8D3WOyqZFb/x/m6SE0Jcg7zvIYbnAOq1Fexeik0Fc/DNzLAoGBAMua
d4XIfu4ydtU5AIaf1ZNXywgLg+LWxK8ELNqH/Y2vLAeIiTrOVp+hw9z+zHPD5cnu
6mix783PCOYNLTylrwtAz3fxSz14lsDFQM3ntzVF/6BniTTkKddctcPyqnTvamah
0hD2dzXBS/0mTBYIIMYTNbs0Yj87FTdJZw/+qa2VAoGBAKbzQkp54W6PCIMPabD0
fg4nMRZ5F5bv4seIKcunn068QPs9VQxQ4qCfNeLykDYqGA86cgD9YHzD4UZLxv6t
IUWbCWod0m/XXwPlpIUlmO5VEUD+MiAUzFNDxf6xAE7ku5UXImJNUjseX6l2Xd5v
yz9L6QQuFI5aujQKugiIwp5rAoGATtUVGCCkPNgfOLmkYXu7dxxUCV5kB01+xAEK
2OY0n0pG8vfDophH4/D/ZC7nvJ8J9uDhs/3JStexq1lIvaWtG99RNTChIEDzpdn6
GH9yaVcb/eB4uJjrNm64FhF8PGCCwxA+xMCZMaARKwhMB2/IOMkxUbWboL3gnhJ2
rDO/QO0CgYEA2Grt6uXHm61ji3xSdkBWNtUnj19vS1+7rFJp5SoYztVQVThf/W52
BAiXKBdYZDRVoItC/VS2NvAOjeJjhYO/xQ/q3hK7MdtuXfEPpLnyXKkmWo3lrJ26
wbeF6l05LexCkI7ShsOuSt+dsyaTJTszuKDIA6YOfWvfo3aVZmlWRaI=
-----END RSA PRIVATE KEY-----
`

	block, _ := pem.Decode([]byte(privateKeyString))
	privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	s.jwtService.PrivateKey = privateKey
	s.jwtService.PublicKey = &privateKey.PublicKey
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
