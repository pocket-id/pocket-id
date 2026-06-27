package oidc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"gorm.io/gorm"
)

type fakeAuditLogger struct {
	events []model.AuditLogEvent
	data   []model.AuditLogData
}

func (f *fakeAuditLogger) Create(_ context.Context, event model.AuditLogEvent, _, _, _ string, data model.AuditLogData, _ *gorm.DB) (model.AuditLog, bool) {
	f.events = append(f.events, event)
	f.data = append(f.data, data)
	return model.AuditLog{}, true
}

func TestAuthorizationServiceAuthorizeLogsClientAuthorization(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	auditLogger := &fakeAuditLogger{}
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, auditLogger, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          newTestAuthorizeRequester("audit-request", clientID, ""),
		meta:               requestMeta{IPAddress: "203.0.113.1", UserAgent: "test-agent"},
	})
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)

	require.Equal(t, []model.AuditLogEvent{model.AuditLogEventClientAuthorization}, auditLogger.events)
	require.Equal(t, model.AuditLogData{"clientName": "Test Client"}, auditLogger.data[0])
}

func TestAuthorizationServiceRejectsCustomScopeWithoutResource(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client"}).Error)

	// A custom permission requested with no resource must be rejected at the
	// authorize endpoint, not displayed on the consent screen.
	requester := newTestAuthorizeRequesterWithForm("bad-scope-request", clientID, url.Values{})
	requester.(*fosite.AuthorizeRequest).RequestedScope = fosite.Arguments{"openid", "users:read"}

	_, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          requester,
		meta:               requestMeta{},
	})
	require.Error(t, err)

	var rfcErr *fosite.RFC6749Error
	require.ErrorAs(t, err, &rfcErr)
	require.Equal(t, "invalid_scope", rfcErr.ErrorField)
}

func TestAuthorizationServiceConsentStepLogsNewClientAuthorization(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	auditLogger := &fakeAuditLogger{}
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, auditLogger, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:            model.Base{ID: interactionID},
		Scopes:          datatype.StringList{"openid"},
		ClientID:        clientID,
		ConsentRequired: true,
		RequestedAt:     datatype.DateTime(time.Now().UTC()),
		Parameters:      map[string]string{},
	}).Error)

	response, err := service.completeInteractionStep(t.Context(), interactionID, userID, interactionStepConsent, "", time.Now().UTC(), requestMeta{})
	require.NoError(t, err)
	require.NotEmpty(t, response.RedirectURL)

	require.Equal(t, []model.AuditLogEvent{model.AuditLogEventNewClientAuthorization}, auditLogger.events)
	require.Equal(t, model.AuditLogData{"clientName": "Test Client"}, auditLogger.data[0])
}

func TestAuthorizationServiceAuthorizeConsumesInteractionSession(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters:  map[string]string{},
	}).Error)

	authorize := func() (authorizationResult, error) {
		return service.authorize(t.Context(), authorizeInput{
			userID:             userID,
			authenticationTime: time.Now().UTC(),
			requester:          newTestAuthorizeRequester("consume-request", clientID, ""),
			interactionID:      interactionID,
		})
	}

	authorization, err := authorize()
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)

	// The interaction session is consumed together with the grant and must be single-use
	var count int64
	require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", interactionID).Count(&count).Error)
	require.Zero(t, count)

	_, err = authorize()
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

func TestInteractionSessionServiceGetRejectsExpiredSession(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newInteractionSessionService(db)

	const (
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters:  map[string]string{},
	}).Error)

	_, err := service.get(t.Context(), interactionID)
	require.NoError(t, err)

	// Backdate the session past its lifetime; BeforeCreate stamps CreatedAt, so update directly
	expiredCreatedAt := datatype.DateTime(time.Now().Add(-interactionSessionLifetime - time.Minute))
	require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", interactionID).Update("created_at", expiredCreatedAt).Error)

	_, err = service.get(t.Context(), interactionID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAuthorizationServiceAuthorizeBindsScopesToInteractionSession(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)

	newInteractionSession := func(id string) {
		require.NoError(t, db.Create(&InteractionSession{
			Base:        model.Base{ID: id},
			Scopes:      datatype.StringList{"openid"},
			ClientID:    clientID,
			RequestedAt: datatype.DateTime(time.Now().UTC()),
			Parameters:  map[string]string{},
		}).Error)
	}

	authorize := func(interactionID string, scopes ...string) (authorizationResult, error) {
		requester := newTestAuthorizeRequester("scope-binding-request", clientID, "")
		requester.(*fosite.AuthorizeRequest).RequestedScope = fosite.Arguments(scopes)
		return service.authorize(t.Context(), authorizeInput{
			userID:             userID,
			authenticationTime: time.Now().UTC(),
			requester:          requester,
			interactionID:      interactionID,
		})
	}

	// Scopes matching the consented interaction session are granted
	newInteractionSession("matching-interaction")
	authorization, err := authorize("matching-interaction", "openid")
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)

	// Scopes added to the URL after consent must not be granted silently
	newInteractionSession("escalated-interaction")
	_, err = authorize("escalated-interaction", "openid", "profile", "email", "groups")
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)

	// The user must not have been recorded as having authorized the escalated scopes
	var authorizedClient model.UserAuthorizedOidcClient
	require.NoError(t, db.First(&authorizedClient, "user_id = ? AND client_id = ?", userID, clientID).Error)
	require.Equal(t, datatype.StringList{"openid"}, authorizedClient.Scope)
}

func TestAuthorizationServiceAuthorizeRejectsInteractionSessionOfOtherClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		otherClientID = "other-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: otherClientID},
		Name: "Other Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters:  map[string]string{},
	}).Error)

	_, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          newTestAuthorizeRequester("other-client-request", otherClientID, ""),
		interactionID:      interactionID,
	})
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

func TestAuthorizationServiceAuthorizeSwitchesUserAndResetsRequirements(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		otherUserID   = "other-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base:     model.Base{ID: userID},
		Username: "test-user",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Base:     model.Base{ID: otherUserID},
		Username: "other-user",
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:                     model.Base{ID: interactionID},
		Scopes:                   datatype.StringList{"openid"},
		ClientID:                 clientID,
		UserID:                   stringPointer(userID),
		ReauthenticatedAt:        new(datatype.DateTime(time.Now().Add(-time.Minute).UTC())),
		ReauthenticationRequired: false,
		RequestedAt:              datatype.DateTime(time.Now().UTC()),
		Parameters: map[string]string{
			"prompt": "login",
		},
	}).Error)

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             otherUserID,
		authenticationTime: time.Now().UTC(),
		requester:          newTestAuthorizeRequester("other-user-request", clientID, ""),
		interactionID:      interactionID,
	})
	require.NoError(t, err)
	require.True(t, authorization.RequiresInteraction)
	require.Equal(t, interactionID, authorization.InteractionID)

	var interactionSession InteractionSession
	require.NoError(t, db.First(&interactionSession, "id = ?", interactionID).Error)
	require.NotNil(t, interactionSession.UserID)
	require.Equal(t, otherUserID, *interactionSession.UserID)
	require.True(t, interactionSession.ConsentRequired)
	require.True(t, interactionSession.ReauthenticationRequired)
	require.Nil(t, interactionSession.ReauthenticatedAt)
}

func TestAuthorizationServiceAuthorizeRequiresLoginForUserBoundInteraction(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		UserID:      stringPointer(userID),
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters:  map[string]string{},
	}).Error)

	_, err := service.authorize(t.Context(), authorizeInput{
		requester:     newTestAuthorizeRequester("unauthenticated-bound-request", clientID, ""),
		interactionID: interactionID,
	})
	require.ErrorIs(t, err, fosite.ErrLoginRequired)
}

func TestAuthorizationServiceCompleteInteractionBindsUserToSession(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:            model.Base{ID: interactionID},
		Scopes:          datatype.StringList{"openid"},
		ClientID:        clientID,
		ConsentRequired: true,
		RequestedAt:     datatype.DateTime(time.Now().UTC()),
		Parameters:      map[string]string{},
	}).Error)

	response, err := service.completeInteractionStep(t.Context(), interactionID, userID, interactionStepConsent, "", time.Now().UTC(), requestMeta{})
	require.NoError(t, err)
	require.NotEmpty(t, response.RedirectURL)

	var interactionSession InteractionSession
	require.NoError(t, db.First(&interactionSession, "id = ?", interactionID).Error)
	require.NotNil(t, interactionSession.UserID)
	require.Equal(t, userID, *interactionSession.UserID)
}

func TestAuthorizationServiceCompleteInteractionSwitchesUserAndResetsRequirements(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		otherUserID   = "other-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base:     model.Base{ID: userID},
		Username: "test-user",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Base:     model.Base{ID: otherUserID},
		Username: "other-user",
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:                     model.Base{ID: interactionID},
		Scopes:                   datatype.StringList{"openid"},
		ClientID:                 clientID,
		UserID:                   stringPointer(userID),
		AccountSelectionRequired: true,
		RequestedAt:              datatype.DateTime(time.Now().UTC()),
		Parameters: map[string]string{
			"prompt": "select_account",
		},
	}).Error)

	response, err := service.completeInteractionStep(t.Context(), interactionID, otherUserID, interactionStepSelectAccount, "", time.Now().UTC(), requestMeta{})
	require.NoError(t, err)
	require.Empty(t, response.RedirectURL)
	require.NotNil(t, response.Interaction)
	require.Equal(t, interactionStepConsent, response.Interaction.CurrentStep)

	var interactionSession InteractionSession
	require.NoError(t, db.First(&interactionSession, "id = ?", interactionID).Error)
	require.NotNil(t, interactionSession.UserID)
	require.Equal(t, otherUserID, *interactionSession.UserID)
	require.False(t, interactionSession.AccountSelectionRequired)
	require.True(t, interactionSession.ConsentRequired)
}

// TestAuthorizationServiceSelectAccountRecomputesConsentForSelectedUser ensures consent is
// evaluated for the FINAL selected user. When an already-consented user starts
// prompt=select_account and a different, not-yet-consented user is selected, consent must
// still be required for that user rather than being inherited from the initiator.
func TestAuthorizationServiceSelectAccountRecomputesConsentForSelectedUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		initiatorID = "initiator-user"
		switchedID  = "switched-user"
		clientID    = "test-client"
	)

	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: initiatorID}, Username: "initiator"}).Error)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: switchedID}, Username: "switched"}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client"}).Error)

	// The initiator has previously consented to the client; the switched-to user has not.
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   initiatorID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             initiatorID,
		authenticationTime: time.Now().UTC(),
		requester:          newTestAuthorizeRequester("select-account-request", clientID, "select_account"),
		requestParams:      map[string]string{"prompt": "select_account"},
	})
	require.NoError(t, err)
	require.True(t, authorization.RequiresInteraction)

	response, err := service.completeInteractionStep(t.Context(), authorization.InteractionID, switchedID, interactionStepSelectAccount, "", time.Now().UTC(), requestMeta{})
	require.NoError(t, err)

	// The flow must not be granted yet: consent is still pending for the switched-to user.
	require.Empty(t, response.RedirectURL)
	require.NotNil(t, response.Interaction)
	require.Equal(t, interactionStepConsent, response.Interaction.CurrentStep)

	// No authorization record may have been silently created for the switched-to user.
	var count int64
	require.NoError(t, db.Model(&model.UserAuthorizedOidcClient{}).
		Where("user_id = ? AND client_id = ?", switchedID, clientID).
		Count(&count).Error)
	require.Zero(t, count)
}

func TestValidateClientPKCERequirement(t *testing.T) {
	pkceClient := Client{OidcClient: model.OidcClient{Base: model.Base{ID: "c"}, PkceEnabled: true}}
	plainClient := Client{OidcClient: model.OidcClient{Base: model.Base{ID: "c"}}}

	withChallenge := newTestAuthorizeRequesterWithForm("with-challenge", "c", url.Values{"code_challenge": {"abc123"}})
	withoutChallenge := newTestAuthorizeRequesterWithForm("without-challenge", "c", url.Values{})

	require.NoError(t, validateClientPKCERequirement(plainClient, withoutChallenge))
	require.NoError(t, validateClientPKCERequirement(pkceClient, withChallenge))

	err := validateClientPKCERequirement(pkceClient, withoutChallenge)
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

// TestAuthorizationServiceAuthorizeEnforcesPerClientPKCE proves the per-client PkceEnabled
// flag is honored for confidential clients (fosite only enforces PKCE for public clients).
func TestAuthorizationServiceAuthorizeEnforcesPerClientPKCE(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID   = "test-user"
		clientID = "pkce-client"
	)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "test-user"}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "PKCE Client", PkceEnabled: true}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: userID, ClientID: clientID, Scope: datatype.StringList{"openid"}}).Error)

	pkceClient := Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}, Name: "PKCE Client", PkceEnabled: true}}

	// Without a code_challenge the request is rejected before any interaction.
	missing := newTestAuthorizeRequesterWithForm("authz-no-pkce", clientID, url.Values{})
	missing.(*fosite.AuthorizeRequest).Client = pkceClient
	_, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          missing,
	})
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)

	// With a code_challenge the PKCE gate passes and the (already-consented) request is granted.
	withChallenge := newTestAuthorizeRequesterWithForm("authz-pkce", clientID, url.Values{"code_challenge": {"abc123"}})
	withChallenge.(*fosite.AuthorizeRequest).Client = pkceClient
	result, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          withChallenge,
	})
	require.NoError(t, err)
	require.False(t, result.RequiresInteraction)
}

func TestAuthorizationServiceAuthorizePARRequiredClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:                                model.Base{ID: clientID},
		Name:                                "Test Client",
		RequiresPushedAuthorizationRequests: true,
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)

	authorize := func(interactionID string, hasPushedAuthorizationRequest bool) (authorizationResult, error) {
		requester := newTestAuthorizeRequester("par-request", clientID, "")
		requester.(*fosite.AuthorizeRequest).Client = Client{
			OidcClient: model.OidcClient{
				Base:                                model.Base{ID: clientID},
				Name:                                "Test Client",
				RequiresPushedAuthorizationRequests: true,
			},
		}
		return service.authorize(t.Context(), authorizeInput{
			userID:                        userID,
			authenticationTime:            time.Now().UTC(),
			requester:                     requester,
			hasPushedAuthorizationRequest: hasPushedAuthorizationRequest,
			interactionID:                 interactionID,
		})
	}

	// Without a pushed authorization request the client must be rejected
	_, err := authorize("", false)
	var parRequiredError *common.OidcPARRequiredError
	require.ErrorAs(t, err, &parRequiredError)

	// Re-entry after a completed interaction carries no request_uri, but the bound
	// interaction session proves the original request was PAR-validated
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters:  map[string]string{},
	}).Error)

	authorization, err := authorize(interactionID, false)
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)

	// An unknown interaction ID must not satisfy the PAR requirement
	_, err = authorize("nonexistent", false)
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

func TestAuthorizationServiceInteractionRequestQuery(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:        model.Base{ID: interactionID},
		Scopes:      datatype.StringList{"openid"},
		ClientID:    clientID,
		RequestedAt: datatype.DateTime(time.Now().UTC()),
		Parameters: map[string]string{
			"client_id":     clientID,
			"response_type": "code",
			"scope":         "openid",
			"redirect_uri":  "https://client.example/callback",
			"state":         "test-state",
		},
	}).Error)

	query, err := service.interactionRequestQuery(t.Context(), interactionID)
	require.NoError(t, err)
	require.Equal(t, clientID, query.Get("client_id"))
	require.Equal(t, "code", query.Get("response_type"))
	require.Equal(t, "openid", query.Get("scope"))
	require.Equal(t, "https://client.example/callback", query.Get("redirect_uri"))
	require.Equal(t, "test-state", query.Get("state"))
	require.Equal(t, interactionID, query.Get("interaction"))

	_, err = service.interactionRequestQuery(t.Context(), "nonexistent")
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

func TestAuthorizationServiceAuthorizeUsesLoginAuthenticationTime(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	loginTime := time.Now().Add(-10 * time.Minute).UTC().Truncate(time.Second)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)

	firstAuthorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: loginTime,
		requester:          newTestAuthorizeRequester("first-request", clientID, ""),
	})
	require.NoError(t, err)
	require.False(t, firstAuthorization.RequiresInteraction)
	require.Equal(t, loginTime, firstAuthorization.Session.IDTokenClaims().AuthTime)

	secondAuthorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: loginTime,
		requester:          newTestAuthorizeRequester("second-request", clientID, "none"),
	})
	require.NoError(t, err)
	require.False(t, secondAuthorization.RequiresInteraction)
	require.Equal(t, loginTime, secondAuthorization.Session.IDTokenClaims().AuthTime)
}

func TestAuthorizationServiceAuthorizeRequiresReauthenticationWhenMaxAgeExceeded(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	loginTime := time.Now().Add(-2 * time.Minute).UTC()

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: loginTime,
		requester:          newTestAuthorizeRequesterWithForm("max-age-request", clientID, url.Values{"max_age": {"1"}}),
		requestParams:      map[string]string{"max_age": "1"},
	})
	require.NoError(t, err)
	require.True(t, authorization.RequiresInteraction)

	interaction, err := service.getInteractionSession(t.Context(), authorization.InteractionID)
	require.NoError(t, err)
	require.Equal(t, interactionStepReauthenticate, interaction.CurrentStep)
}

func TestAuthorizationServiceAuthorizeUsesCompletedReauthenticationTime(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)
	loginTime := time.Now().Add(-2 * time.Minute).UTC()
	reauthenticatedAt := time.Now().Add(-1 * time.Second).UTC().Truncate(time.Second)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:                     model.Base{ID: interactionID},
		Scopes:                   datatype.StringList{"openid"},
		ClientID:                 clientID,
		ReauthenticationRequired: false,
		ReauthenticatedAt:        new(datatype.DateTime(reauthenticatedAt)),
		Parameters: map[string]string{
			"max_age": "1",
		},
	}).Error)

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: loginTime,
		requester:          newTestAuthorizeRequesterWithForm("final-request", clientID, url.Values{"max_age": {"1"}}),
		interactionID:      interactionID,
		requestParams:      map[string]string{"max_age": "1"},
	})
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)
	require.Equal(t, reauthenticatedAt, authorization.Session.IDTokenClaims().AuthTime)
}

func TestAuthorizationServiceAuthorizeUsesOriginalInteractionRequestTime(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil, nil)

	const (
		userID        = "test-user"
		clientID      = "test-client"
		interactionID = "test-interaction"
	)
	originalRequestedAt := time.Now().Add(-10 * time.Second).UTC().Truncate(time.Second)
	reauthenticatedAt := originalRequestedAt.Add(5 * time.Second)
	continuationRequestedAt := reauthenticatedAt.Add(5 * time.Second)

	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    datatype.StringList{"openid"},
	}).Error)
	require.NoError(t, db.Create(&InteractionSession{
		Base:                     model.Base{ID: interactionID},
		Scopes:                   datatype.StringList{"openid"},
		ClientID:                 clientID,
		ReauthenticationRequired: false,
		RequestedAt:              datatype.DateTime(originalRequestedAt),
		ReauthenticatedAt:        new(datatype.DateTime(reauthenticatedAt)),
		Parameters: map[string]string{
			"prompt": "login",
		},
	}).Error)

	requester := newTestAuthorizeRequesterWithForm(
		"final-request",
		clientID,
		url.Values{"prompt": {"login"}},
	)
	requester.(*fosite.AuthorizeRequest).RequestedAt = continuationRequestedAt

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: originalRequestedAt.Add(-time.Minute),
		requester:          requester,
		interactionID:      interactionID,
		requestParams:      map[string]string{"prompt": "login"},
	})
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)
	require.Equal(t, originalRequestedAt, authorization.Session.IDTokenClaims().RequestedAt)
	require.Equal(t, reauthenticatedAt, authorization.Session.IDTokenClaims().AuthTime)
	require.False(t, authorization.Session.IDTokenClaims().AuthTime.Before(authorization.Session.IDTokenClaims().RequestedAt))
}

func TestInteractionSessionServiceSavePersistsParameters(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newInteractionSessionService(db)

	const (
		clientID      = "test-client"
		interactionID = "test-interaction"
	)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)

	interactionSession, err := service.create(t.Context(), InteractionSession{
		Base:                     model.Base{ID: interactionID},
		Scopes:                   datatype.StringList{"openid"},
		ClientID:                 clientID,
		ReauthenticationRequired: true,
		RequestedAt:              datatype.DateTime(time.Now().UTC()),
		Parameters: map[string]string{
			"max_age": "1",
		},
	})
	require.NoError(t, err)

	reauthenticatedAt := datatype.DateTime(time.Now().UTC().Truncate(time.Second))
	interactionSession.ReauthenticationRequired = false
	interactionSession.ReauthenticatedAt = &reauthenticatedAt
	require.NoError(t, service.update(t.Context(), interactionSession))

	storedInteractionSession, err := service.get(t.Context(), interactionID)
	require.NoError(t, err)
	require.False(t, storedInteractionSession.ReauthenticationRequired)
	require.Equal(t, "1", storedInteractionSession.Parameters["max_age"])
	require.NotNil(t, storedInteractionSession.ReauthenticatedAt)
	require.Equal(t, reauthenticatedAt.UTC(), storedInteractionSession.ReauthenticatedAt.UTC())
}

func newTestAuthorizeRequester(requestID, clientID, prompt string) fosite.AuthorizeRequester {
	form := url.Values{}
	if prompt != "" {
		form.Set("prompt", prompt)
	}
	return newTestAuthorizeRequesterWithForm(requestID, clientID, form)
}

func newTestAuthorizeRequesterWithForm(requestID, clientID string, form url.Values) fosite.AuthorizeRequester {
	requester := fosite.NewAuthorizeRequest()
	requester.ID = requestID
	requester.RequestedAt = time.Now().UTC()
	requester.Client = Client{
		OidcClient: model.OidcClient{
			Base: model.Base{ID: clientID},
			Name: "Test Client",
		},
	}
	requester.RequestedScope = fosite.Arguments{"openid"}
	requester.ResponseTypes = fosite.Arguments{"code"}
	requester.RedirectURI = &url.URL{Scheme: "https", Host: "client.example", Path: "/callback"}
	requester.Form = form
	return requester
}

func stringPointer(value string) *string {
	return &value
}

func TestConsentRequired(t *testing.T) {
	tests := []struct {
		name                 string
		hasAlreadyAuthorized bool
		skipConsent          bool
		prompt               string
		want                 bool
	}{
		{"new client without skip requires consent", false, false, "", true},
		{"returning client without skip skips consent", true, false, "", false},
		{"new client with skip skips consent", false, true, "", false},
		{"returning client with skip skips consent", true, true, "", false},
		{"prompt=consent forces consent despite skip", false, true, "consent", true},
		{"prompt=consent forces consent for returning client", true, false, "consent", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := consentRequired(tt.hasAlreadyAuthorized, tt.skipConsent, newPromptValues(tt.prompt))
			require.Equal(t, tt.want, got)
		})
	}
}

// A client with SkipConsent must be granted without a consent interaction even on the first authorization, while consent is still recorded so the user can later see and revoke the client
func TestAuthorizationServiceSkipConsentGrantsWithoutInteraction(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	auditLogger := &fakeAuditLogger{}
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, auditLogger)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client", SkipConsent: true}).Error)

	// No prior UserAuthorizedOidcClient record exists, so a client without skip-consent would require the consent screen here
	requester := newTestAuthorizeRequester("skip-consent-request", clientID, "")
	requester.(*fosite.AuthorizeRequest).Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client", SkipConsent: true}}

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          requester,
		meta:               requestMeta{IPAddress: "203.0.113.1", UserAgent: "test-agent"},
	})
	require.NoError(t, err)
	require.False(t, authorization.RequiresInteraction)
	require.NotNil(t, authorization.Session)

	var count int64
	require.NoError(t, db.Model(&model.UserAuthorizedOidcClient{}).Where("user_id = ? AND client_id = ?", userID, clientID).Count(&count).Error)
	require.Equal(t, int64(1), count)

	require.Equal(t, []model.AuditLogEvent{model.AuditLogEventNewClientAuthorization}, auditLogger.events)
}

// A client with SkipConsent must still show the consent screen when the request explicitly asks for it with prompt=consent
func TestAuthorizationServiceSkipConsentHonorsPromptConsent(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "", nil), nil, nil)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client", SkipConsent: true}).Error)

	requester := newTestAuthorizeRequester("skip-consent-prompt-request", clientID, "consent")
	requester.(*fosite.AuthorizeRequest).Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client", SkipConsent: true}}

	authorization, err := service.authorize(t.Context(), authorizeInput{
		userID:             userID,
		authenticationTime: time.Now().UTC(),
		requester:          requester,
		meta:               requestMeta{},
	})
	require.NoError(t, err)
	require.True(t, authorization.RequiresInteraction)

	interactionSession, err := newInteractionSessionService(db).get(t.Context(), authorization.InteractionID)
	require.NoError(t, err)
	require.True(t, interactionSession.ConsentRequired)
}
