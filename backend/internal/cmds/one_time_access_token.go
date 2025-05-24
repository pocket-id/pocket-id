package cmds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/signals"
)

// OneTimeAccessToken creates a one-time access token for the given user
// Args must contain the username or email of the user
func OneTimeAccessToken(args []string) error {
	// Get a context that is canceled when the application is stopping
	ctx := signals.SignalContext(context.Background())

	// Get the username or email of the user
	// Note length is 2 because the first argument is always the command (one-time-access-token)
	if len(args) != 2 {
		return errors.New("missing username or email of user; usage: one-time-access-token <username or email>")
	}
	userArg := args[1]

	// Connect to the database
	db := bootstrap.NewDatabase()

	// Create the access token
	var oneTimeAccessToken *model.OneTimeAccessToken
	err := db.Transaction(func(tx *gorm.DB) error {
		// Load the user to retrieve the user ID
		var user model.User
		queryCtx, queryCancel := context.WithTimeout(ctx, 10*time.Second)
		defer queryCancel()
		txErr := tx.
			WithContext(queryCtx).
			Where("username = ? OR email = ?", userArg, userArg).
			First(&user).
			Error
		switch {
		case errors.Is(txErr, gorm.ErrRecordNotFound):
			return errors.New("user not found")
		case txErr != nil:
			return fmt.Errorf("failed to query for user: %w", txErr)
		case user.ID == "":
			return errors.New("invalid user loaded: ID is empty")
		}

		// Create a new access token that expires in 1 hour
		oneTimeAccessToken, txErr = service.NewOneTimeAccessToken(user.ID, time.Now().Add(time.Hour))
		if txErr != nil {
			return fmt.Errorf("failed to generate access token: %w", txErr)
		}

		queryCtx, queryCancel = context.WithTimeout(ctx, 10*time.Second)
		defer queryCancel()
		txErr = tx.
			WithContext(queryCtx).
			Create(oneTimeAccessToken).
			Error
		if txErr != nil {
			return fmt.Errorf("failed to save access token: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Print the result
	fmt.Printf(`A one-time access token valid for 1 hour has been created for "%s".`+"\n", userArg)
	fmt.Printf("Use the following URL to sign in once: %s/lc/%s\n", common.EnvConfig.AppURL, oneTimeAccessToken.Token)

	return nil
}
