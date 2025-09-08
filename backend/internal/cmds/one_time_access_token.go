package cmds

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		db, err := bootstrap.NewDatabase()
		if err != nil {
			return err
		}

		// Create the access token
		var oneTimeAccessToken *model.OneTimeAccessToken
		err = db.Transaction(func(tx *gorm.DB) error {
			// Load the user to retrieve the user ID (case-insensitive on username)
			var user model.User
			queryCtx, queryCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer queryCancel()

			// Try exact match first
			txErr := tx.
				WithContext(queryCtx).
				Where("username = ? OR email = ?", userArg, userArg).
				First(&user).
				Error

			// If not found, try lowercasing the username (case-insensitive behavior)
			if errors.Is(txErr, gorm.ErrRecordNotFound) {
				lc := strings.ToLower(userArg)
				txErr = tx.
					WithContext(queryCtx).
					Where("username = ? OR email = ?", lc, userArg).
					First(&user).
					Error
			}

			switch {
			case errors.Is(txErr, gorm.ErrRecordNotFound):
				return errors.New("user not found")
			case txErr != nil:
				return fmt.Errorf("failed to query for user: %w", txErr)
			case user.ID == "":
				return errors.New("invalid user loaded: ID is empty")
			}

			// Create a new access token that expires in 1 hour
			oneTimeAccessToken, txErr = service.NewOneTimeAccessToken(user.ID, time.Hour)
			if txErr != nil {
				return fmt.Errorf("failed to generate access token: %w", txErr)
			}

			queryCtx, queryCancel = context.WithTimeout(cmd.Context(), 10*time.Second)
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
	},
}

func init() {
	rootCmd.AddCommand(oneTimeAccessTokenCmd)
}
