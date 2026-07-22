//go:build e2etest

package usersignup

import (
	"context"
	"fmt"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
)

// SignupTokenSeed describes a signup token to seed into the actor state, used by E2E test setup.
type SignupTokenSeed struct {
	ID           string
	Token        string
	ExpiresAt    time.Time
	UsageLimit   int
	UsageCount   int
	UserGroupIDs []string
	CreatedAt    time.Time
}

// SeedSignupTokens replaces the signup token singleton actor's state with the given tokens.
// It's intended for E2E test setup, where fixture tokens need to be created with exact values.
func SeedSignupTokens(ctx context.Context, actors *local.Host, seeds []SignupTokenSeed) error {
	tokens := make([]storedSignupToken, len(seeds))
	for i, s := range seeds {
		tokens[i] = storedSignupToken{
			ID:           s.ID,
			Token:        s.Token,
			ExpiresAt:    s.ExpiresAt,
			UsageLimit:   s.UsageLimit,
			UsageCount:   s.UsageCount,
			UserGroupIDs: s.UserGroupIDs,
			CreatedAt:    s.CreatedAt,
		}
	}

	_, err := actors.Service().Invoke(ctx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodReplace, signupTokenReplaceRequest{
		Tokens: tokens,
	})
	if err != nil {
		return fmt.Errorf("failed to seed signup tokens into actor: %w", err)
	}
	return nil
}
