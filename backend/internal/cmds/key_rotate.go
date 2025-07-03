package cmds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type keyRotateFlags struct {
	Alg string
	Crv string
	Yes bool
}

func init() {
	var flags keyRotateFlags

	keyRotateCmd := &cobra.Command{
		Use:   "key-rotate",
		Short: "Generates a new token signing key and replaces the current one",
		RunE: func(cmd *cobra.Command, args []string) error {
			db := bootstrap.NewDatabase()

			return keyRotate(cmd.Context(), flags, db, &common.EnvConfig)
		},
	}

	keyRotateCmd.Flags().StringVarP(&flags.Alg, "alg", "a", "RS256", "Key algorithm. Supported values: RS256, RS384, RS512, ES256, ES384, ES512, EdDSA")
	keyRotateCmd.Flags().StringVarP(&flags.Crv, "crv", "c", "", "Curve name when using EdDSA keys. Supported values: Ed25519")
	keyRotateCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Do not prompt for confirmation")

	rootCmd.AddCommand(keyRotateCmd)
}

func keyRotate(ctx context.Context, flags keyRotateFlags, db *gorm.DB, envConfig *common.EnvConfigSchema) error {
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
			return nil
		}
	}

	// Init the services we need
	appConfigService := service.NewAppConfigService(ctx, db)

	// Get the key provider
	keyProvider, err := jwkutils.GetKeyProvider(db, envConfig, appConfigService.GetDbConfig().InstanceID.Value)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	// Generate a new key
	key, err := jwkutils.GenerateKey(flags.Alg, flags.Crv)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save the key
	err = keyProvider.SaveKey(key)
	if err != nil {
		return fmt.Errorf("failed to store new key: %w", err)
	}

	fmt.Println("Key rotated successfully")
	fmt.Println("Note: if pocket-id is running, you will need to restart it for the new key to be loaded")

	return nil
}
