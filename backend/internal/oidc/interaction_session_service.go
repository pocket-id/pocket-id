package oidc

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// interactionSessionLifetime is how long a pending interaction stays valid before the
// user has to restart the authorization flow.
const interactionSessionLifetime = time.Hour

type interactionSessionService struct {
	db *gorm.DB
}

func newInteractionSessionService(db *gorm.DB) *interactionSessionService {
	return &interactionSessionService{
		db: db,
	}
}

func (s *interactionSessionService) create(ctx context.Context, interactionSession InteractionSession) (InteractionSession, error) {
	err := dbFromContext(ctx, s.db).
		Create(&interactionSession).
		Error
	if err != nil {
		return InteractionSession{}, err
	}

	return interactionSession, nil
}

func (s *interactionSessionService) get(ctx context.Context, id string) (InteractionSession, error) {
	var interactionSession InteractionSession
	err := dbFromContext(ctx, s.db).
		Preload("Client").
		First(&interactionSession, "id = ?", id).
		Error
	if err != nil {
		return InteractionSession{}, err
	}

	if time.Since(interactionSession.CreatedAt.ToTime()) > interactionSessionLifetime {
		return InteractionSession{}, gorm.ErrRecordNotFound
	}

	return interactionSession, nil
}

func (s *interactionSessionService) update(ctx context.Context, interactionSession InteractionSession) error {
	return dbFromContext(ctx, s.db).
		Model(&InteractionSession{}).
		Where("id = ?", interactionSession.ID).
		Updates(map[string]any{
			"authentication_required":    interactionSession.AuthenticationRequired,
			"account_selection_required": interactionSession.AccountSelectionRequired,
			"reauthentication_required":  interactionSession.ReauthenticationRequired,
			"consent_required":           interactionSession.ConsentRequired,
			"user_id":                    interactionSession.UserID,
			"reauthenticated_at":         interactionSession.ReauthenticatedAt,
			"parameters":                 interactionSession.Parameters,
		}).
		Error
}

func (s *interactionSessionService) delete(ctx context.Context, id string) error {
	tx := dbFromContext(ctx, s.db).Where("id = ?", id).Delete(&InteractionSession{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func hasRemainingInteractionSteps(interactionSession InteractionSession) bool {
	return interactionSession.AuthenticationRequired ||
		interactionSession.AccountSelectionRequired ||
		interactionSession.ReauthenticationRequired ||
		interactionSession.ConsentRequired
}
