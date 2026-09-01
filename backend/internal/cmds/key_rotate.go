package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type keyRotateFlags struct {
	Alg        string
	Crv        string
	SessionKey bool
	Yes        bool
}

func init() {
	var flags keyRotateFlags

	keyRotateCmd := &cobra.Command{
		Use:   "key-rotate",
		Short: "Generates a new signing key and replaces the current one",
		RunE: func(cmd *cobra.Command, args []string) error {
			// The session key is always a symmetric HS256 key, so the algorithm flags don't apply to it
			if flags.SessionKey && (cmd.Flags().Changed("alg") || cmd.Flags().Changed("crv")) {
				return errors.New("the --alg and --crv flags cannot be used together with --session-key")
			}

			db, _, err := bootstrap.NewDatabase(cmd.Context())
			if err != nil {
				return err
			}

			instanceID, err := instanceid.Load(cmd.Context(), db)
			if err != nil {
				return err
			}

			return keyRotate(cmd.Context(), flags, db, instanceID, &common.EnvConfig)
		},
	}

	keyRotateCmd.Flags().StringVarP(&flags.Alg, "alg", "a", "RS256", "Key algorithm. Supported values: RS256, RS384, RS512, ES256, ES384, ES512, EdDSA")
	keyRotateCmd.Flags().StringVarP(&flags.Crv, "crv", "c", "", "Curve name when using EdDSA keys. Supported values: Ed25519")
	keyRotateCmd.Flags().BoolVar(&flags.SessionKey, "session-key", false, "Rotate the key used to sign session tokens instead of the token signing key")
	keyRotateCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Do not prompt for confirmation")

	rootCmd.AddCommand(keyRotateCmd)
}

func keyRotate(ctx context.Context, flags keyRotateFlags, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema) error {
	// The session key is a separate key, generated with a fixed algorithm, so it's rotated on its own
	if flags.SessionKey {
		return sessionKeyRotate(ctx, flags, db, instanceID, envConfig)
	}

	// Validate the flags
	switch strings.ToUpper(flags.Alg) {
	case jwa.RS256().String(), jwa.RS384().String(), jwa.RS512().String(),
		jwa.ES256().String(), jwa.ES384().String(), jwa.ES512().String():
		// All good, but uppercase it for consistency
		flags.Alg = strings.ToUpper(flags.Alg)
	case strings.ToUpper(jwa.EdDSA().String()):
		// Ensure Crv is set and valid
		switch strings.ToUpper(flags.Crv) {
		case strings.ToUpper(jwa.Ed25519().String()):
			// All good, but ensure consistency in casing
			flags.Crv = jwa.Ed25519().String()
		case "":
			return errors.New("a curve name is required when algorithm is EdDSA")
		default:
			return errors.New("unsupported EdDSA curve; supported values: Ed25519")
		}
	case "":
		return errors.New("key algorithm is required")
	default:
		return errors.New("unsupported key algorithm; supported values: RS256, RS384, RS512, ES256, ES384, ES512, EdDSA")
	}

	if !flags.Yes {
		fmt.Println("WARNING: Rotating the private key will invalidate all existing tokens. Both pocket-id and all client applications will likely need to be restarted.")
		ok, err := utils.PromptForConfirmation("Confirm")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted")
			os.Exit(1)
		}
	}

	// Get the key provider
	keyProvider, err := jwkutils.GetKeyProvider(db, envConfig, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	// Generate a new key
	key, err := jwkutils.GenerateKey(flags.Alg, flags.Crv)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save the key
	err = keyProvider.SaveKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to store new key: %w", err)
	}

	fmt.Println("Key rotated successfully")
	fmt.Println("Note: if pocket-id is running, you will need to restart it for the new key to be loaded")

	return nil
}

func sessionKeyRotate(ctx context.Context, flags keyRotateFlags, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema) error {
	if !flags.Yes {
		fmt.Println("WARNING: Rotating the session key will invalidate all existing sessions, and all users will need to sign in again. Tokens issued to client applications are not affected.")
		ok, err := utils.PromptForConfirmation("Confirm")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted")
			os.Exit(1)
		}
	}

	// Get the key provider for the session key
	keyProvider, err := jwkutils.GetSessionKeyProvider(db, envConfig, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get session key provider: %w", err)
	}

	// Generate a new key
	key, err := jwkutils.GenerateSessionKey()
	if err != nil {
		return fmt.Errorf("failed to generate session key: %w", err)
	}

	// Save the key
	err = keyProvider.SaveKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to store new session key: %w", err)
	}

	fmt.Println("Session key rotated successfully")
	fmt.Println("Note: if pocket-id is running, you will need to restart it for the new key to be loaded")

	return nil
}
