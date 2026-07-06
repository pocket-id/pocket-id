package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func newAuthorizationService(db *gorm.DB, interactionSessionService *interactionSessionService, claimsService *ClaimsService, reauth ReauthenticationTokenConsumer, auditLog AuditLogger, apiAccess APIAccessProvider) *authorizationService {
	return &authorizationService{
		db:                        db,
		interactionSessionService: interactionSessionService,
		claimsService:             claimsService,
		reauth:                    reauth,
		auditLog:                  auditLog,
		apiAccess:                 apiAccess,
	}
}

type authorizationService struct {
	db                        *gorm.DB
	interactionSessionService *interactionSessionService
	claimsService             *ClaimsService
	reauth                    ReauthenticationTokenConsumer
	auditLog                  AuditLogger
	apiAccess                 APIAccessProvider
}

// resolveGrant resolves the RFC 8707 resource of a request into the token audience, the scopes that may actually be granted, and the audience-qualified keys used to record and check consent
// It always resolves against the client's user-delegated grants because every flow that passes through here acts on behalf of a user
func (s *authorizationService) resolveGrant(ctx context.Context, clientID, resource string, requestedScopes []string) (audience string, grantedScopes []string, consentKeys []string, err error) {
	audience, grantedScopes, err = resolveResource(ctx, s.apiAccess, clientID, resource, requestedScopes, SubjectTypeUser)
	if err != nil {
		return "", nil, nil, err
	}
	return audience, grantedScopes, consentScopeKeys(audience, grantedScopes), nil
}

type requestMeta struct {
	IPAddress string
	UserAgent string
}

type authorizationResult struct {
	RequiresInteraction bool
	InteractionID       string
	Session             *Session
}

type promptValues []string

func newPromptValues(prompt string) promptValues {
	return strings.Fields(prompt)
}

func (p promptValues) has(value string) bool {
	return slices.Contains(p, value)
}

// consentRequired reports whether the user has to be shown the consent screen
// A client can be configured to skip consent so trusted first-party apps are not prompted on every new authorization, but an explicit prompt=consent always forces the screen regardless of that setting
func consentRequired(hasAlreadyAuthorizedClient, clientSkipsConsent bool, prompt promptValues) bool {
	if prompt.has("consent") {
		return true
	}
	return !hasAlreadyAuthorizedClient && !clientSkipsConsent
}

// authorizeInput is the authorization request as provided by the handler.
type authorizeInput struct {
	userID                        string
	authenticationMethod          string
	authenticationTime            time.Time
	requester                     fosite.AuthorizeRequester
	hasPushedAuthorizationRequest bool
	reauthenticationToken         string
	interactionID                 string
	requestParams                 map[string]string
	meta                          requestMeta
}

// authorizeRequest is the input enriched with everything the service derives from it.
type authorizeRequest struct {
	authorizeInput

	client             Client
	prompt             promptValues
	interactionSession *InteractionSession
	now                time.Time
}

func (s *authorizationService) authorize(ctx context.Context, input authorizeInput) (authorizationResult, error) {
	client := input.requester.GetClient().(Client)
	prompt := newPromptValues(input.requester.GetRequestForm().Get("prompt"))

	err := validateClientPKCERequirement(client, input.requester)
	if err != nil {
		return authorizationResult{}, err
	}

	interactionSession, err := s.boundInteractionSession(ctx, input.interactionID, input.userID, client, input.requester)
	if err != nil {
		return authorizationResult{}, err
	}

	// Reject authorization requests that require PAR when the request is not a resumed interaction and doesn't have a valid PAR
	if client.RequiresPushedAuthorizationRequests && !input.hasPushedAuthorizationRequest && interactionSession == nil {
		return authorizationResult{}, &common.OidcPARRequiredError{}
	}

	resource, err := input.requester.GetResource()
	if err != nil {
		return authorizationResult{}, err
	}

	// Validate the requested scopes against the targeted API up front, before the user authenticates or reaches the consent screen
	// This rejects a custom permission requested without, or with the wrong, resource at the authorize endpoint itself
	_, _, _, err = s.resolveGrant(ctx, client.GetID(), resource, input.requester.GetRequestedScopes())
	if err != nil {
		// resolveGrant distinguishes an unknown API, an API this client is not granted, and a scope not allowed for the API
		// This validation runs before authentication, so returning those distinct errors would let anyone holding a public client_id diff the responses to enumerate which API audiences exist and which ones the client may request
		// Collapse every resource-targeted failure into one generic invalid_request so the pre-auth response reveals no backend state, while keeping the underlying reason in the server log for operators
		// A request that names no resource cannot leak API topology, so its scope error is returned unchanged to help legitimate integrations
		if resource != "" {
			slog.DebugContext(ctx, "Rejected authorize request with an invalid or unauthorized resource or scope", "client_id", client.GetID(), "error", err.Error())
			return authorizationResult{}, fosite.ErrInvalidRequest.WithHint("The 'resource' or 'scope' parameter is invalid.")
		}

		return authorizationResult{}, err
	}

	if input.userID == "" {
		if prompt.has("none") {
			return authorizationResult{}, fosite.ErrLoginRequired
		}

		interactionSession, err := s.createInteractionSession(ctx, input.requester, input.requestParams, "", interactionRequirements{
			AuthenticationRequired:   true,
			ReauthenticationRequired: prompt.has("login") || client.RequiresReauthentication,
			AccountSelectionRequired: prompt.has("select_account"),
			ConsentRequired:          prompt.has("consent"),
		})
		if err != nil {
			return authorizationResult{}, err
		}

		return authorizationResult{RequiresInteraction: true, InteractionID: interactionSession.ID}, nil
	}

	req := authorizeRequest{
		authorizeInput:     input,
		client:             client,
		prompt:             prompt,
		interactionSession: interactionSession,
		now:                time.Now().UTC(),
	}

	codeChallenge := input.requester.GetRequestForm().Get("code_challenge")

	var result authorizationResult
	err = withTx(ctx, s.db, func(ctx context.Context) error {
		var txErr error
		result, txErr = s.authorizeAuthenticated(ctx, req)
		if txErr != nil {
			return txErr
		}

		if codeChallenge != "" && !client.PkceEnabled && !client.PkceSupported {
			tx := dbFromContext(ctx, s.db)
			_ = flagPkceSupportedClient(ctx, client.GetID(), tx)
		}

		return nil
	})
	if err != nil {
		return authorizationResult{}, err
	}

	if result.Session == nil {
		return result, nil
	}

	err = s.claimsService.applyIDTokenClaims(ctx, result.Session, input.requester.GetGrantedScopes())
	if err != nil {
		return authorizationResult{}, err
	}

	return result, nil
}

// authorizeAuthenticated either reports the interaction the user still has to complete or grants the request.
func (s *authorizationService) authorizeAuthenticated(ctx context.Context, req authorizeRequest) (authorizationResult, error) {
	var user model.User
	err := dbFromContext(ctx, s.db).
		Preload("UserGroups").
		First(&user, "id = ?", req.userID).
		Error
	if err != nil {
		return authorizationResult{}, err
	}

	if !IsUserGroupAllowedToAuthorize(user, req.client.OidcClient) {
		return authorizationResult{}, fosite.ErrAccessDenied.WithHint("You are not allowed to access this service.")
	}

	interactionSession := req.interactionSession
	if interactionSession != nil && interactionSession.UserID != nil && *interactionSession.UserID != req.userID {
		if err := s.switchInteractionSessionUser(ctx, interactionSession, req.userID, req.authenticationTime); err != nil {
			return authorizationResult{}, err
		}
		if err := s.interactionSessionService.update(ctx, *interactionSession); err != nil {
			return authorizationResult{}, err
		}
	}

	requirements, authenticationTime, err := s.resolveRequirements(ctx, req, interactionSession)
	if err != nil {
		return authorizationResult{}, err
	}

	if requirements.any() {
		if interactionSession != nil {
			return authorizationResult{RequiresInteraction: true, InteractionID: interactionSession.ID}, nil
		}

		created, err := s.createInteractionSession(ctx, req.requester, req.requestParams, req.userID, requirements)
		if err != nil {
			return authorizationResult{}, err
		}

		return authorizationResult{RequiresInteraction: true, InteractionID: created.ID}, nil
	}

	if interactionSession != nil {
		err := s.interactionSessionService.delete(ctx, interactionSession.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return authorizationResult{}, fosite.ErrInvalidRequest.WithHint("The interaction session has already been used.")
		}
		if err != nil {
			return authorizationResult{}, err
		}
	}

	resource, err := req.requester.GetResource()
	if err != nil {
		return authorizationResult{}, err
	}
	audience, grantedScopes, consentKeys, err := s.resolveGrant(ctx, req.client.GetID(), resource, req.requester.GetRequestedScopes())
	if err != nil {
		return authorizationResult{}, err
	}

	hasAlreadyAuthorizedClient, err := s.consent(ctx, req.userID, req.client.GetID(), consentKeys)
	if err != nil {
		return authorizationResult{}, err
	}

	session := s.buildAuthorizedSession(req, interactionSession, authenticationTime)

	grantResourceIndicator(req.requester, audience, grantedScopes)

	authorizationEvent := model.AuditLogEventClientAuthorization
	if !hasAlreadyAuthorizedClient {
		authorizationEvent = model.AuditLogEventNewClientAuthorization
	}
	if s.auditLog != nil {
		s.auditLog.Create(ctx, authorizationEvent, req.meta.IPAddress, req.meta.UserAgent, req.userID, model.AuditLogData{"clientName": req.client.Name}, dbFromContext(ctx, s.db))
	}

	return authorizationResult{Session: session}, nil
}

func flagPkceSupportedClient(ctx context.Context, clientID string, tx *gorm.DB) error {
	err := tx.
		WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ?", clientID).
		Update("pkce_supported", true).
		Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

// resolveRequirements determines the interaction steps still required; the requirements
// of a resumed interaction session win over the ones derived from the request.
func (s *authorizationService) resolveRequirements(ctx context.Context, req authorizeRequest, interactionSession *InteractionSession) (interactionRequirements, time.Time, error) {
	authenticationTime := req.authenticationTime

	resource, err := req.requester.GetResource()
	if err != nil {
		return interactionRequirements{}, authenticationTime, err
	}
	_, _, consentKeys, err := s.resolveGrant(ctx, req.client.GetID(), resource, req.requester.GetRequestedScopes())
	if err != nil {
		return interactionRequirements{}, authenticationTime, err
	}
	hasAlreadyAuthorizedClient, err := s.hasAuthorizedClient(ctx, req.client.GetID(), req.userID, consentKeys)
	if err != nil {
		return interactionRequirements{}, authenticationTime, err
	}

	maxAgeReauthenticationRequired, err := requiresReauthenticationForMaxAge(req.requester.GetRequestForm().Get("max_age"), authenticationTime, req.now)
	if err != nil {
		return interactionRequirements{}, authenticationTime, err
	}

	requirements := interactionRequirements{
		ConsentRequired:          consentRequired(hasAlreadyAuthorizedClient, req.client.SkipConsent, req.prompt),
		ReauthenticationRequired: req.prompt.has("login") || req.client.RequiresReauthentication || maxAgeReauthenticationRequired,
		AccountSelectionRequired: req.prompt.has("select_account"),
		AuthenticationRequired:   false,
	}

	if interactionSession != nil {
		requirements = interactionRequirements{
			ConsentRequired:          interactionSession.ConsentRequired,
			ReauthenticationRequired: interactionSession.ReauthenticationRequired,
			AccountSelectionRequired: interactionSession.AccountSelectionRequired,
			AuthenticationRequired:   interactionSession.AuthenticationRequired,
		}
		if interactionSession.ReauthenticatedAt != nil {
			authenticationTime = interactionSession.ReauthenticatedAt.UTC()
		}
	}

	if req.prompt.has("none") && requirements.ConsentRequired {
		return interactionRequirements{}, authenticationTime, fosite.ErrConsentRequired
	}
	if req.prompt.has("none") && requirements.ReauthenticationRequired {
		return interactionRequirements{}, authenticationTime, fosite.ErrLoginRequired
	}

	if requirements.ReauthenticationRequired && req.reauthenticationToken != "" && s.reauth != nil {
		reauthenticatedAt, err := s.reauth.ConsumeReauthenticationToken(ctx, dbFromContext(ctx, s.db), req.reauthenticationToken, req.userID)
		if err == nil {
			requirements.ReauthenticationRequired = false
			authenticationTime = reauthenticatedAt
		}
	}

	return requirements, authenticationTime, nil
}

func (s *authorizationService) buildAuthorizedSession(req authorizeRequest, interactionSession *InteractionSession, authenticationTime time.Time) *Session {
	if authenticationTime.IsZero() {
		authenticationTime = req.now
	}
	requestedAt := req.requester.GetRequestedAt()
	if interactionSession != nil && !interactionSession.RequestedAt.ToTime().IsZero() {
		requestedAt = interactionSession.RequestedAt.UTC()
	}
	if requestedAt.IsZero() {
		requestedAt = req.now
	}

	return NewAuthenticatedSession(req.userID, req.authenticationMethod, authenticationTime, requestedAt)
}

// interactionRequestQuery returns the authorize parameters stored for the interaction
// session, so the handler can rebuild the re-entry request server-side.
func (s *authorizationService) interactionRequestQuery(ctx context.Context, interactionID string) (query url.Values, err error) {
	interactionSession, err := s.interactionSessionService.get(ctx, interactionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fosite.ErrInvalidRequest.WithHint("The interaction session is invalid or has expired.")
	}
	if err != nil {
		return nil, err
	}

	query = url.Values{}
	for key, value := range interactionSession.Parameters {
		if value != "" {
			query.Set(key, value)
		}
	}
	query.Set("interaction", interactionSession.ID)

	return query, nil
}

// boundInteractionSession loads the referenced interaction session and validates the request against it.
func (s *authorizationService) boundInteractionSession(ctx context.Context, interactionID string, userID string, client Client, requester fosite.AuthorizeRequester) (*InteractionSession, error) {
	if interactionID == "" {
		return nil, nil
	}

	session, err := s.interactionSessionService.get(ctx, interactionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fosite.ErrInvalidRequest.WithHint("The interaction session is invalid or has expired.")
	}
	if err != nil {
		return nil, err
	}

	if err := validateInteractionSessionBinding(session, userID, client, requester); err != nil {
		return nil, err
	}

	return &session, nil
}

// validateInteractionSessionBinding ensures a request matches the interaction session
// it resumes: the requirement flags and the PAR exemption come from the session, so a
// tampered request (extra scopes, another client's interaction) must not inherit them.
func validateInteractionSessionBinding(interactionSession InteractionSession, userID string, client Client, requester fosite.AuthorizeRequester) error {
	if interactionSession.ClientID != client.GetID() {
		return fosite.ErrInvalidRequest.WithHint("The interaction session does not belong to this client.")
	}

	if interactionSession.UserID != nil {
		if userID == "" {
			return fosite.ErrLoginRequired
		}
	}

	for _, scope := range requester.GetRequestedScopes() {
		if !slices.Contains(interactionSession.Scopes, scope) {
			return fosite.ErrInvalidRequest.WithHint("The requested scopes exceed the scopes of the interaction session.")
		}
	}

	return nil
}

type interactionRequirements struct {
	ConsentRequired          bool
	ReauthenticationRequired bool
	AuthenticationRequired   bool
	AccountSelectionRequired bool
}

func (r interactionRequirements) any() bool {
	return r.ConsentRequired || r.ReauthenticationRequired || r.AuthenticationRequired || r.AccountSelectionRequired
}

func (s *authorizationService) createInteractionSession(ctx context.Context, requester fosite.AuthorizeRequester, requestParams map[string]string, userID string, requirements interactionRequirements) (InteractionSession, error) {
	parameters := make(map[string]string, len(requestParams))
	for key, value := range requestParams {
		parameters[key] = value
	}

	return s.interactionSessionService.create(ctx, InteractionSession{
		Base: model.Base{
			ID: requester.GetID(),
		},
		Scopes:                   datatype.StringList(requester.GetRequestedScopes()),
		ClientID:                 requester.GetClient().GetID(),
		UserID:                   utils.PtrOrNil(userID),
		ConsentRequired:          requirements.ConsentRequired,
		ReauthenticationRequired: requirements.ReauthenticationRequired,
		AuthenticationRequired:   requirements.AuthenticationRequired,
		AccountSelectionRequired: requirements.AccountSelectionRequired,
		RequestedAt:              datatype.DateTime(requester.GetRequestedAt()),
		Parameters:               parameters,
	})
}

func bindInteractionSessionUser(interactionSession *InteractionSession, userID string) error {
	if userID == "" {
		return fosite.ErrLoginRequired
	}
	if interactionSession.UserID != nil && *interactionSession.UserID != userID {
		return fosite.ErrInvalidRequest.WithHint("The interaction session belongs to another user.")
	}
	if interactionSession.UserID == nil {
		interactionSession.UserID = &userID
	}

	return nil
}

func (s *authorizationService) switchInteractionSessionUser(ctx context.Context, interactionSession *InteractionSession, userID string, authenticationTime time.Time) error {
	if userID == "" {
		return fosite.ErrLoginRequired
	}
	interactionSession.UserID = &userID

	requirements, err := s.interactionRequirementsForUser(ctx, userID, interactionSession, authenticationTime)
	if err != nil {
		return err
	}

	interactionSession.AuthenticationRequired = false
	interactionSession.ConsentRequired = requirements.ConsentRequired
	interactionSession.ReauthenticationRequired = requirements.ReauthenticationRequired
	interactionSession.ReauthenticatedAt = nil

	return nil
}

func (s *authorizationService) getInteractionSession(ctx context.Context, interactionSessionID string) (interactionSessionForUser, error) {
	interactionSession, err := s.interactionSessionService.get(ctx, interactionSessionID)
	if err != nil {
		return interactionSessionForUser{}, err
	}

	return s.buildInteractionForUser(ctx, interactionSession)
}

// buildInteractionForUser builds the consent-screen DTO and enriches it with display information for the requested custom-API permissions
func (s *authorizationService) buildInteractionForUser(ctx context.Context, interactionSession InteractionSession) (interactionSessionForUser, error) {
	result, err := newInteractionSessionForUser(interactionSession)
	if err != nil {
		return interactionSessionForUser{}, err
	}

	scopeInfo, err := s.resolveScopeInfo(ctx, interactionSession)
	if err != nil {
		return interactionSessionForUser{}, err
	}
	// Always serialize a possibly empty array rather than null
	if scopeInfo == nil {
		scopeInfo = []scopeInfoDto{}
	}
	result.ScopeInfo = scopeInfo

	return result, nil
}

// resolveScopeInfo resolves display names and descriptions for the requested non-standard scopes, looked up against the API targeted by the request's RFC 8707 resource
// Standard identity scopes are rendered by the client
func (s *authorizationService) resolveScopeInfo(ctx context.Context, interactionSession InteractionSession) ([]scopeInfoDto, error) {
	return s.resolveScopeInfoForRequest(ctx, interactionSession.Parameters["resource"], interactionSession.Scopes)
}

// resolveScopeInfoForRequest resolves display names and descriptions for the requested non-standard scopes against the API identified by resource
// The browser and device consent flows share it so both show friendly permission names instead of raw scope keys
func (s *authorizationService) resolveScopeInfoForRequest(ctx context.Context, resource string, scopes []string) ([]scopeInfoDto, error) {
	if s.apiAccess == nil {
		return nil, nil
	}

	if resource == "" {
		return nil, nil
	}

	customKeys := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if !isStandardScope(scope) {
			customKeys = append(customKeys, scope)
		}
	}
	if len(customKeys) == 0 {
		return nil, nil
	}

	infos, err := s.apiAccess.DescribePermissions(ctx, resource, customKeys)
	if err != nil {
		return nil, err
	}

	scopeInfo := make([]scopeInfoDto, len(infos))
	for i, info := range infos {
		scopeInfo[i] = scopeInfoDto(info)
	}

	return scopeInfo, nil
}

func (s *authorizationService) completeInteractionStep(ctx context.Context, interactionSessionID, userID string, step interactionStep, reauthenticationToken string, authenticationTime time.Time, meta requestMeta) (completeInteractionResponse, error) {
	var interactionSession InteractionSession
	var response completeInteractionResponse
	err := withTx(ctx, s.db, func(ctx context.Context) error {
		var err error
		interactionSession, err = s.interactionSessionService.get(ctx, interactionSessionID)
		if err != nil {
			return err
		}

		if interactionSession.UserID != nil && *interactionSession.UserID != userID {
			if err := s.switchInteractionSessionUser(ctx, &interactionSession, userID, authenticationTime); err != nil {
				return err
			}
		}

		if userID == "" {
			return fosite.ErrLoginRequired
		}

		requiredSteps := requiredInteractionSteps(interactionSession)
		if len(requiredSteps) == 0 {
			response = completeInteractionResponse{RedirectURL: authorizeRedirectURL(interactionSession.ID)}
			return nil
		}

		if requiredSteps[0] != step {
			return &common.ValidationError{Message: "expected interaction step " + string(requiredSteps[0]) + " but got " + string(step)}
		}

		if err := s.applyInteractionStep(ctx, &interactionSession, userID, step, reauthenticationToken, authenticationTime, meta); err != nil {
			return err
		}

		return s.interactionSessionService.update(ctx, interactionSession)
	})
	if err != nil {
		return completeInteractionResponse{}, err
	}
	if response.RedirectURL != "" {
		return response, nil
	}

	if !hasRemainingInteractionSteps(interactionSession) {
		return completeInteractionResponse{RedirectURL: authorizeRedirectURL(interactionSession.ID)}, nil
	}

	interaction, err := s.buildInteractionForUser(ctx, interactionSession)
	if err != nil {
		return completeInteractionResponse{}, err
	}

	return completeInteractionResponse{Interaction: &interaction}, nil
}

func (s *authorizationService) applyInteractionStep(ctx context.Context, interactionSession *InteractionSession, userID string, step interactionStep, reauthenticationToken string, authenticationTime time.Time, meta requestMeta) error {
	switch step {
	case interactionStepAuthenticate:
		if err := bindInteractionSessionUser(interactionSession, userID); err != nil {
			return err
		}
		interactionSession.AuthenticationRequired = false
		return s.populatePostAuthenticationRequirements(ctx, userID, interactionSession, authenticationTime)
	case interactionStepSelectAccount:
		if err := s.switchInteractionSessionUser(ctx, interactionSession, userID, authenticationTime); err != nil {
			return err
		}
		interactionSession.AccountSelectionRequired = false
		return nil
	case interactionStepReauthenticate:
		return s.completeReauthenticationStep(ctx, interactionSession, userID, reauthenticationToken)
	case interactionStepConsent:
		return s.completeConsentStep(ctx, interactionSession, userID, meta)
	default:
		return &common.ValidationError{Message: "unknown interaction step " + string(step)}
	}
}

func (s *authorizationService) completeReauthenticationStep(ctx context.Context, interactionSession *InteractionSession, userID, reauthenticationToken string) error {
	if err := bindInteractionSessionUser(interactionSession, userID); err != nil {
		return err
	}
	if reauthenticationToken == "" {
		return &common.ValidationError{Message: "reauthentication token is required"}
	}
	reauthenticatedAt, err := s.reauth.ConsumeReauthenticationToken(ctx, dbFromContext(ctx, s.db), reauthenticationToken, userID)
	if err != nil {
		return err
	}

	interactionSession.ReauthenticationRequired = false
	interactionSession.ReauthenticatedAt = new(datatype.DateTime(reauthenticatedAt))
	return nil
}

func (s *authorizationService) completeConsentStep(ctx context.Context, interactionSession *InteractionSession, userID string, meta requestMeta) error {
	if err := bindInteractionSessionUser(interactionSession, userID); err != nil {
		return err
	}
	resource := interactionSession.Parameters["resource"]
	_, _, consentKeys, err := s.resolveGrant(ctx, interactionSession.ClientID, resource, interactionSession.Scopes)
	if err != nil {
		return err
	}
	hasAlreadyAuthorizedClient, err := s.consent(ctx, userID, interactionSession.ClientID, consentKeys)
	if err != nil {
		return err
	}
	if !hasAlreadyAuthorizedClient && s.auditLog != nil {
		s.auditLog.Create(ctx, model.AuditLogEventNewClientAuthorization, meta.IPAddress, meta.UserAgent, userID, model.AuditLogData{"clientName": interactionSession.Client.Name}, dbFromContext(ctx, s.db))
	}
	interactionSession.ConsentRequired = false
	return nil
}

func (s *authorizationService) populatePostAuthenticationRequirements(ctx context.Context, userID string, interactionSession *InteractionSession, authenticationTime time.Time) error {
	if userID == "" {
		return errors.New("user is required to complete authentication")
	}

	requirements, err := s.interactionRequirementsForUser(ctx, userID, interactionSession, authenticationTime)
	if err != nil {
		return err
	}

	interactionSession.ConsentRequired = interactionSession.ConsentRequired || requirements.ConsentRequired
	interactionSession.AccountSelectionRequired = interactionSession.AccountSelectionRequired || requirements.AccountSelectionRequired
	interactionSession.ReauthenticationRequired = interactionSession.ReauthenticationRequired || requirements.ReauthenticationRequired

	return nil
}

func (s *authorizationService) interactionRequirementsForUser(ctx context.Context, userID string, interactionSession *InteractionSession, authenticationTime time.Time) (interactionRequirements, error) {
	prompt := newPromptValues(interactionSession.Parameters["prompt"])
	resource := interactionSession.Parameters["resource"]
	_, _, consentKeys, err := s.resolveGrant(ctx, interactionSession.ClientID, resource, interactionSession.Scopes)
	if err != nil {
		return interactionRequirements{}, err
	}
	hasAlreadyAuthorizedClient, err := s.hasAuthorizedClient(ctx, interactionSession.ClientID, userID, consentKeys)
	if err != nil {
		return interactionRequirements{}, err
	}

	maxAgeReauthenticationRequired, err := requiresReauthenticationForMaxAge(interactionSession.Parameters["max_age"], authenticationTime, time.Now().UTC())
	if err != nil {
		return interactionRequirements{}, err
	}

	return interactionRequirements{
		ConsentRequired:          consentRequired(hasAlreadyAuthorizedClient, interactionSession.Client.SkipConsent, prompt),
		ReauthenticationRequired: prompt.has("login") || interactionSession.Client.RequiresReauthentication || maxAgeReauthenticationRequired,
		AccountSelectionRequired: prompt.has("select_account"),
		AuthenticationRequired:   false,
	}, nil
}

// authorizeRedirectURL only references the interaction session; the authorize endpoint
// restores the request parameters from it server-side.
func authorizeRedirectURL(interactionSessionID string) string {
	return "/authorize?interaction=" + url.QueryEscape(interactionSessionID)
}

// validateClientPKCERequirement enforces the per-client PkceEnabled flag. fosite only
// enforces PKCE for public clients (EnforcePKCEForPublicClients).
func validateClientPKCERequirement(client Client, requester fosite.AuthorizeRequester) error {
	if !client.PkceEnabled {
		return nil
	}
	if requester.GetRequestForm().Get("code_challenge") == "" {
		return fosite.ErrInvalidRequest.WithHint("This client requires PKCE, but the 'code_challenge' parameter is missing.")
	}
	return nil
}

func requiresReauthenticationForMaxAge(maxAgeRaw string, authenticationTime time.Time, now time.Time) (bool, error) {
	if maxAgeRaw == "" {
		return false, nil
	}

	maxAge, err := strconv.ParseInt(maxAgeRaw, 10, 64)
	if err != nil || maxAge < 0 {
		return false, fosite.ErrInvalidRequest.WithHint("Parameter 'max_age' must be a non-negative integer.")
	}

	if authenticationTime.IsZero() {
		return true, nil
	}

	return !now.Before(authenticationTime.UTC().Add(time.Duration(maxAge) * time.Second)), nil
}

func (s *authorizationService) consent(ctx context.Context, userID string, clientID string, scope []string) (hasAlreadyAuthorizedClient bool, err error) {
	db := dbFromContext(ctx, s.db)

	hasAlreadyAuthorizedClient, err = s.hasAuthorizedClient(ctx, clientID, userID, scope)
	if err != nil {
		return false, err
	}

	if hasAlreadyAuthorizedClient {
		err = db.
			Model(&model.UserAuthorizedOidcClient{}).
			Where("user_id = ? AND client_id = ?", userID, clientID).
			Update("last_used_at", datatype.DateTime(time.Now())).
			Error

		if err != nil {
			return hasAlreadyAuthorizedClient, err
		}

		return hasAlreadyAuthorizedClient, nil
	}

	userAuthorizedClient := model.UserAuthorizedOidcClient{
		UserID:     userID,
		ClientID:   clientID,
		Scope:      scope,
		LastUsedAt: datatype.DateTime(time.Now()),
	}

	err = db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "client_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"scope"}),
		}).
		Create(&userAuthorizedClient).
		Error

	return hasAlreadyAuthorizedClient, err
}

func (s *authorizationService) hasAuthorizedClient(ctx context.Context, clientID, userID string, scope []string) (bool, error) {
	var userAuthorizedOidcClient model.UserAuthorizedOidcClient
	err := dbFromContext(ctx, s.db).
		First(&userAuthorizedOidcClient, "client_id = ? AND user_id = ?", clientID, userID).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	authorizedScopes := userAuthorizedOidcClient.Scope
	for _, requestedScope := range scope {
		if !slices.Contains(authorizedScopes, requestedScope) {
			return false, nil
		}
	}

	return true, nil
}

// IsUserGroupAllowedToAuthorize reports whether the user may use the group-restricted client.
func IsUserGroupAllowedToAuthorize(user model.User, client model.OidcClient) bool {
	if !client.IsGroupRestricted {
		return true
	}

	for _, userGroup := range client.AllowedUserGroups {
		for _, userGroupUser := range user.UserGroups {
			if userGroup.ID == userGroupUser.ID {
				return true
			}
		}
	}

	return false
}
