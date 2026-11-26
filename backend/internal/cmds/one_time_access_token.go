package cmds

import (
	"context"
	"errors"
	"fmt"
	"time"

	francishost "github.com/italypaleale/francis/host"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
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

		user, err := findUserByUsernameOrEmail(cmd.Context(), db, userArg)
		if err != nil {
			return err
		}

		instanceID, err := instanceid.Load(cmd.Context(), db)
		if err != nil {
			return err
		}

		// One-time access tokens live in the actor state store, which is reached differently depending on where the actor runtime runs
		// The CLI never runs the full actor host: with an embedded runtime it writes to Pocket ID's database directly, and with a standalone one it joins the cluster as a client for just long enough to write the token
		token, err := storeOneTimeAccessToken(cmd.Context(), pg, instanceID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to create access token: %w", err)
		}
		auditlogs.RecordCLI(cmd.Context(), db, auditlogs.EventLoginCodeCreated, user.ID, auditlogs.Data{"delivery": "link"})

		// Print the result
		fmt.Printf(`A one-time access token valid for 1 hour has been created for "%s".`+"\n", user.Username)
		fmt.Printf("Use the following URL to sign in once: %s/lc/%s\n", common.EnvConfig.AppURL, token)

		return nil
	},
}

// findUserByUsernameOrEmail refuses a value that is one user's username and another's email, because anyone can pick an email address as their username and would get the link by chance
func findUserByUsernameOrEmail(ctx context.Context, db *gorm.DB, value string) (model.User, error) {
	queryCtx, queryCancel := context.WithTimeout(ctx, 10*time.Second)
	defer queryCancel()

	var users []model.User
	err := db.
		WithContext(queryCtx).
		Where("username = ? OR email = ?", value, value).
		Limit(2).
		Find(&users).
		Error
	switch {
	case err != nil:
		return model.User{}, fmt.Errorf("failed to query for user: %w", err)
	case len(users) == 0:
		return model.User{}, errors.New("user not found")
	case len(users) > 1:
		return model.User{}, fmt.Errorf(`"%s" matches the users "%s" and "%s", pass a username or email that only one of them has`, value, users[0].Username, users[1].Username)
	}
	return users[0], nil
}

// storeOneTimeAccessToken persists a one-time access token valid for one hour, through whichever actor runtime this deployment uses, and returns the token
func storeOneTimeAccessToken(ctx context.Context, pg *pgxpool.Pool, instanceID string, userID string) (string, error) {
	// A standalone Francis runtime owns the actor state, so the token is written through a short-lived client connection to it
	if !common.EnvConfig.HasEmbeddedFrancisRuntime() {
		var token string
		err := bootstrap.WithActorClient(ctx, &common.EnvConfig, func(clientCtx context.Context, client francishost.Host) error {
			tokenCtx, tokenCancel := context.WithTimeout(clientCtx, 10*time.Second)
			defer tokenCancel()

			var rErr error
			token, rErr = onetimeaccess.StoreToken(tokenCtx, client, userID, time.Hour, nil, false, "")
			return rErr
		})
		if err != nil {
			return "", err
		}

		return token, nil
	}

	// With the embedded runtime the actor state lives in Pocket ID's own database, or francis.db on SQLite, which a minimal state store writes to without running an actor host
	actorStore, err := bootstrap.NewActorStateStore(bootstrap.NewActorsOpts{
		Postgres:   pg,
		EnvConfig:  &common.EnvConfig,
		InstanceID: instanceID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize the actor state store: %w", err)
	}

	tokenCtx, tokenCancel := context.WithTimeout(ctx, 10*time.Second)
	defer tokenCancel()

	token, err := onetimeaccess.StoreToken(tokenCtx, actorStore, userID, time.Hour, nil, false, "")
	if err != nil {
		return "", err
	}

	return token, nil
}

func init() {
	rootCmd.AddCommand(oneTimeAccessTokenCmd)
}
