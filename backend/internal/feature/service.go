package feature

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
)

// Feature is an optional part of Pocket ID whose pages and tabs the admin UI only shows once it is on
type Feature string

const (
	LDAP                 Feature = "ldap"
	SCIM                 Feature = "scim"
	CustomClaims         Feature = "customClaims"
	APIs                 Feature = "apis"
	APIKeys              Feature = "apiKeys"
	CIMD                 Feature = "cimd"
	FederatedCredentials Feature = "federatedCredentials"
)

var features = []Feature{LDAP, SCIM, CustomClaims, APIs, APIKeys, CIMD, FederatedCredentials}

type Status struct {
	Enabled bool `json:"enabled"`
	InUse   bool `json:"inUse"`
}

type appConfigStore interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
	UpdateAppConfigValues(ctx context.Context, keysAndValues ...string) error
}

type Service struct {
	db        *gorm.DB
	appConfig appConfigStore
}

// List reports every feature as enabled when an admin turned it on or when something uses it
// Something created through the API while its feature was off stays visible, and a feature in use cannot be turned off
func (s *Service) List(ctx context.Context) (map[Feature]Status, error) {
	cfg, err := s.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	usage, err := s.usage(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var enabled []Feature
	if cfg.EnabledFeatures != "" {
		if err := json.Unmarshal([]byte(cfg.EnabledFeatures), &enabled); err != nil {
			return nil, fmt.Errorf("failed to parse the enabled features: %w", err)
		}
	} else if !common.EnvConfig.UiConfigDisabled {
		// Instances from before features turn on what they use once, so those features stay on after their last object is deleted
		// This runs on the first read instead of at startup because the configuration actor is bootstrapped after the host is ready
		enabled = []Feature{}
		for _, feature := range features {
			if usage[feature] > 0 {
				enabled = append(enabled, feature)
			}
		}
		value, err := json.Marshal(enabled)
		if err != nil {
			return nil, err
		}
		if err := s.appConfig.UpdateAppConfigValues(ctx, "enabledFeatures", string(value)); err != nil {
			return nil, fmt.Errorf("failed to save the enabled features: %w", err)
		}
	}

	statuses := map[Feature]Status{}
	for _, feature := range features {
		statuses[feature] = Status{Enabled: usage[feature] > 0, InUse: usage[feature] > 0}
	}
	for _, feature := range enabled {
		if status, ok := statuses[feature]; ok {
			status.Enabled = true
			statuses[feature] = status
		}
	}

	return statuses, nil
}

func (s *Service) usage(ctx context.Context, cfg *appconfig.AppConfigModel) (map[Feature]int64, error) {
	usage := map[Feature]int64{}
	if cfg.LdapEnabled.IsTrue() {
		usage[LDAP] = 1
	}

	var signupClaims []json.RawMessage
	if err := json.Unmarshal([]byte(cfg.SignupDefaultCustomClaims), &signupClaims); err != nil {
		return nil, fmt.Errorf("failed to parse the sign-up default custom claims: %w", err)
	}
	usage[CustomClaims] = int64(len(signupClaims))

	var cimdURLs []string
	if err := json.Unmarshal([]byte(cfg.CIMDURLAllowlist), &cimdURLs); err != nil {
		return nil, fmt.Errorf("failed to parse the CIMD URL allowlist: %w", err)
	}
	usage[CIMD] = int64(len(cimdURLs))

	type query struct {
		feature Feature
		db      *gorm.DB
	}
	db := s.db.WithContext(ctx)
	queries := []query{
		{SCIM, db.Model(&scimsync.ServiceProvider{})},
		{CustomClaims, db.Model(&model.CustomClaim{})},
		{APIs, db.Model(&api.API{})},
		{APIKeys, db.Model(&apikey.ApiKey{})},
		// An API that admits metadata document clients keeps its CIMD access tab even while no such client exists
		{CIMD, db.Model(&model.OidcClient{}).Where("client_type = ?", model.OidcClientTypeCIMD)},
		{CIMD, db.Model(&api.API{}).Where("allow_cimd_clients = ?", true)},
	}
	for _, q := range queries {
		var count int64
		if err := q.db.Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count the usage of %s: %w", q.feature, err)
		}
		usage[q.feature] += count
	}

	// Federated identities live in each client's credentials document, which differs between SQLite and Postgres, so they are counted here
	// A metadata document client's identities come from its document, so only the ones an admin set count
	var clients []model.OidcClient
	err := db.Select("credentials").Where("client_type <> ?", model.OidcClientTypeCIMD).Find(&clients).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load the client credentials: %w", err)
	}
	for _, c := range clients {
		if len(c.Credentials.FederatedIdentities) > 0 {
			usage[FederatedCredentials]++
		}
	}

	return usage, nil
}
