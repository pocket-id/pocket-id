package oidc

import (
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

type interactionStep string

const (
	interactionStepAuthenticate   interactionStep = "authenticate"
	interactionStepSelectAccount  interactionStep = "select_account"
	interactionStepReauthenticate interactionStep = "reauthenticate"
	interactionStepConsent        interactionStep = "consent"
)

type interactionSessionForUser struct {
	ID            string             `json:"id"`
	Scopes        []string           `json:"scopes"`
	ScopeInfo     []dto.ScopeInfoDto `json:"scopeInfo"`
	Client        interactionClient  `json:"client"`
	CurrentStep   interactionStep    `json:"currentStep,omitempty"`
	RequiredSteps []interactionStep  `json:"requiredSteps"`
}

type interactionClient struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HasLogo     bool   `json:"hasLogo"`
	HasDarkLogo bool   `json:"hasDarkLogo"`
	ClientType  string `json:"clientType"`
}

type completeInteractionRequest struct {
	Step interactionStep `json:"step"`
	// Scopes are the scopes the user approved on the consent screen
	// Omitting them approves every requested scope
	Scopes []string `json:"scopes"`
}

type completeInteractionResponse struct {
	Interaction *interactionSessionForUser `json:"interaction,omitempty"`
	RedirectURL string                     `json:"redirectUrl,omitempty"`
}

type verifyDeviceCodeRequest struct {
	// Scopes are the scopes the user approved on the consent screen
	// Omitting them approves every requested scope
	Scopes []string `json:"scopes"`
}

type endSessionRequestDto struct {
	ClientName string `json:"clientName"`
}

type endSessionConfirmationDto struct {
	// RedirectURL is the client's post-logout URL, or empty when the client registered none
	RedirectURL string `json:"redirectUrl"`
}

type claimMappingsPreviewInput struct {
	UserID        string                                   `json:"userId" binding:"required"`
	Scopes        []string                                 `json:"scopes"`
	ClaimMappings []claimmappingpolicy.OidcClaimMappingDto `json:"claimMappings" binding:"dive"`
}

func newInteractionSessionForUser(interactionSession InteractionSession) (interactionSessionForUser, error) {
	var client interactionClient
	if err := dto.MapStruct(interactionSession.Client, &client); err != nil {
		return interactionSessionForUser{}, err
	}

	requiredSteps := requiredInteractionSteps(interactionSession)
	var currentStep interactionStep
	if len(requiredSteps) > 0 {
		currentStep = requiredSteps[0]
	}

	scopes := interactionSession.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	if requiredSteps == nil {
		requiredSteps = []interactionStep{}
	}

	return interactionSessionForUser{
		ID:            interactionSession.ID,
		Scopes:        scopes,
		Client:        client,
		CurrentStep:   currentStep,
		RequiredSteps: requiredSteps,
	}, nil
}

func requiredInteractionSteps(interactionSession InteractionSession) []interactionStep {
	steps := make([]interactionStep, 0, 4)
	if interactionSession.AuthenticationRequired {
		steps = append(steps, interactionStepAuthenticate)
	}
	if interactionSession.AccountSelectionRequired {
		steps = append(steps, interactionStepSelectAccount)
	}
	if interactionSession.ReauthenticationRequired {
		steps = append(steps, interactionStepReauthenticate)
	}
	if interactionSession.ConsentRequired {
		steps = append(steps, interactionStepConsent)
	}

	return steps
}
