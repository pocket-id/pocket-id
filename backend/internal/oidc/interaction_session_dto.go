package oidc

import (
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
	ID            string                    `json:"id"`
	Scopes        []string                  `json:"scopes"`
	ScopeInfo     []dto.ScopeInfoDto        `json:"scopeInfo"`
	Client        dto.OidcClientMetaDataDto `json:"client"`
	CurrentStep   interactionStep           `json:"currentStep,omitempty"`
	RequiredSteps []interactionStep         `json:"requiredSteps"`
}

type completeInteractionRequest struct {
	Step interactionStep `json:"step"`
}

type completeInteractionResponse struct {
	Interaction *interactionSessionForUser `json:"interaction,omitempty"`
	RedirectURL string                     `json:"redirectUrl,omitempty"`
}

func newInteractionSessionForUser(interactionSession InteractionSession) (interactionSessionForUser, error) {
	var client dto.OidcClientMetaDataDto
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
