package bootstrap

import (
	"context"
	"log/slog"
	"path"

	"github.com/pocket-id/fosite"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// moveLegacyOidcClientImages moves client logos of earlier versions, which were stored under the raw client ID, to their hashed path
// Each logo is copied before it is deleted, so an interrupted move finishes on the next start
func moveLegacyOidcClientImages(ctx context.Context, db *gorm.DB, fileStorage storage.FileStorage) error {
	files, err := fileStorage.List(ctx, "oidc-client-images")
	if storage.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	stored := make(map[string]struct{}, len(files))
	for _, f := range files {
		stored[f.Path] = struct{}{}
	}

	var clients []model.OidcClient
	err = db.WithContext(ctx).
		Select("id", "image_type", "dark_image_type").
		Where("image_type IS NOT NULL OR dark_image_type IS NOT NULL").
		Find(&clients).Error
	if err != nil {
		return err
	}

	moved := 0
	for _, client := range clients {
		for suffix, ext := range map[string]*string{"": client.ImageType, "-dark": client.DarkImageType} {
			if ext == nil {
				continue
			}
			oldPath := legacyOidcClientImagePath(client.ID, suffix, *ext)
			if _, ok := stored[oldPath]; !ok {
				continue
			}

			reader, _, err := fileStorage.Open(ctx, oldPath)
			if err != nil {
				return err
			}
			err = fileStorage.Save(ctx, service.OidcClientImagePath(client.ID, suffix, *ext), reader)
			reader.Close()
			if err != nil {
				return err
			}
			if err := fileStorage.Delete(ctx, oldPath); err != nil {
				return err
			}
			moved++
		}
	}

	if moved > 0 {
		slog.Info("Moved OIDC client logos to their hashed paths", slog.Int("count", moved))
	}
	return nil
}

// Earlier versions only allowed client IDs that are safe as file names, except for CIMD clients whose URL was hashed
func legacyOidcClientImagePath(clientID string, suffix string, extension string) string {
	storageID := clientID
	if fosite.LooksLikeCIMDURL(clientID) {
		storageID = "cimd-" + utils.CreateSha256Hash(clientID)
	}
	return path.Join("oidc-client-images", storageID+suffix+"."+extension)
}
