package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type encryptionKeyRotateFlags struct {
	NewKey string
	Yes    bool
}

func init() {
	var flags encryptionKeyRotateFlags

	encryptionKeyRotateCmd := &cobra.Command{
		Use:   "encryption-key-rotate",
		Short: "Re-encrypts data using a new encryption key",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := bootstrap.NewDatabase()
			if err != nil {
				return err
			}

			return encryptionKeyRotate(cmd.Context(), flags, db, &common.EnvConfig)
		},
	}

	encryptionKeyRotateCmd.Flags().StringVar(&flags.NewKey, "new-key", "", "New encryption key to re-encrypt data with")
	encryptionKeyRotateCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Do not prompt for confirmation")

	rootCmd.AddCommand(encryptionKeyRotateCmd)
}

func encryptionKeyRotate(ctx context.Context, flags encryptionKeyRotateFlags, db *gorm.DB, envConfig *common.EnvConfigSchema) error {
	oldKey := envConfig.EncryptionKey
	newKey := []byte(flags.NewKey)
	if len(newKey) == 0 {
		return errors.New("new encryption key is required (--new-key)")
	}
	if len(newKey) < 16 {
		return errors.New("new encryption key must be at least 16 bytes long")
	}

	if !flags.Yes {
		fmt.Println("WARNING: Rotating the encryption key will re-encrypt secrets in the database. Pocket-ID must be restarted with the new ENCRYPTION_KEY after rotation is complete.")
		ok, err := utils.PromptForConfirmation("Continue")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted")
			os.Exit(1)
		}
	}

	appConfigService, err := service.NewAppConfigService(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to create app config service: %w", err)
	}
	instanceID := appConfigService.GetDbConfig().InstanceID.Value

	// Derive the encryption keys used for the JWK encryption
	oldKek, err := jwkutils.LoadKeyEncryptionKey(&common.EnvConfigSchema{EncryptionKey: oldKey}, instanceID)
	if err != nil {
		return fmt.Errorf("failed to derive old key encryption key: %w", err)
	}
	newKek, err := jwkutils.LoadKeyEncryptionKey(&common.EnvConfigSchema{EncryptionKey: newKey}, instanceID)
	if err != nil {
		return fmt.Errorf("failed to derive new key encryption key: %w", err)
	}

	// Derive the encryption keys used for EncryptedString fields
	oldEncKey, err := datatype.DeriveEncryptedStringKey(oldKey)
	if err != nil {
		return fmt.Errorf("failed to derive old encrypted string key: %w", err)
	}
	newEncKey, err := datatype.DeriveEncryptedStringKey(newKey)
	if err != nil {
		return fmt.Errorf("failed to derive new encrypted string key: %w", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		err = rotateSigningKeyEncryption(ctx, tx, oldKek, newKek)
		if err != nil {
			return err
		}

		err = rotateScimTokens(tx, oldEncKey, newEncKey)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("Encryption key rotation completed successfully.")
	fmt.Println("Restart pocket-id with the new ENCRYPTION_KEY to use the rotated data.")

	return nil
}

func rotateSigningKeyEncryption(ctx context.Context, db *gorm.DB, oldKek []byte, newKek []byte) error {
	oldProvider := &jwkutils.KeyProviderDatabase{}
	err := oldProvider.Init(jwkutils.KeyProviderOpts{
		DB:  db,
		Kek: oldKek,
	})
	if err != nil {
		return fmt.Errorf("failed to init key provider with old encryption key: %w", err)
	}

	key, err := oldProvider.LoadKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load signing key using old encryption key: %w", err)
	}
	if key == nil {
		return nil
	}

	newProvider := &jwkutils.KeyProviderDatabase{}
	err = newProvider.Init(jwkutils.KeyProviderOpts{
		DB:  db,
		Kek: newKek,
	})
	if err != nil {
		return fmt.Errorf("failed to init key provider with new encryption key: %w", err)
	}

	if err := newProvider.SaveKey(ctx, key); err != nil {
		return fmt.Errorf("failed to store signing key with new encryption key: %w", err)
	}

	return nil
}

type scimTokenRow struct {
	ID    string
	Token string
}

func rotateScimTokens(db *gorm.DB, oldEncKey []byte, newEncKey []byte) error {
	var rows []scimTokenRow
	err := db.Model(&model.ScimServiceProvider{}).Select("id, token").Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("failed to list SCIM service providers: %w", err)
	}

	for _, row := range rows {
		if row.Token == "" {
			continue
		}

		decBytes, err := datatype.DecryptEncryptedStringWithKey(oldEncKey, row.Token)
		if err != nil {
			return fmt.Errorf("failed to decrypt SCIM token for provider %s: %w", row.ID, err)
		}

		encValue, err := datatype.EncryptEncryptedStringWithKey(newEncKey, decBytes)
		if err != nil {
			return fmt.Errorf("failed to encrypt SCIM token for provider %s: %w", row.ID, err)
		}

		err = db.Model(&model.ScimServiceProvider{}).Where("id = ?", row.ID).Update("token", encValue).Error
		if err != nil {
			return fmt.Errorf("failed to update SCIM token for provider %s: %w", row.ID, err)
		}
	}

	return nil
}
