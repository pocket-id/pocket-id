package cmds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

var oneTimeAccessTokenCmd = &cobra.Command{
	Use:   "one-time-access-token [username or email]",
	Short: "Generates a one-time access token for the given user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the username or email of the user
		userArg := args[0]

		// Connect to the database
		db, pg, err := bootstrap.NewDatabase(cmd.Context())
		if err != nil {
			return err
		}

		// Load the user to retrieve the user ID
		var user model.User
		queryCtx, queryCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer queryCancel()
		err = db.
			WithContext(queryCtx).
			Where("username = ? OR email = ?", userArg, userArg).
			First(&user).
			Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return errors.New("user not found")
		case err != nil:
			return fmt.Errorf("failed to query for user: %w", err)
		case user.ID == "":
			return errors.New("invalid user loaded: ID is empty")
		}

		// One-time access tokens are stored in the actor state store.
		// The CLI doesn't run the full actor host, so it uses a minimal state store to persist the token directly.
		actorStore, err := bootstrap.NewActorStateStore(db, pg)
		if err != nil {
			return fmt.Errorf("failed to initialize the actor state store: %w", err)
		}

		// Create a new access token that expires in 1 hour
		tokenCtx, tokenCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer tokenCancel()
		token, _, err := service.StoreOneTimeAccessToken(tokenCtx, actorStore, user.ID, time.Hour, false)
		if err != nil {
			return fmt.Errorf("failed to create access token: %w", err)
		}

		// Print the result
		fmt.Printf(`A one-time access token valid for 1 hour has been created for "%s".`+"\n", userArg)
		fmt.Printf("Use the following URL to sign in once: %s/lc/%s\n", common.EnvConfig.AppURL, token)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(oneTimeAccessTokenCmd)
}
