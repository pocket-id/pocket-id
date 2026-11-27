package appassets

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

const (
	applicationAssetsPath = "application-assets"
	// assetKeyPrefix prefixes the "kv" rows that hold the file of each asset relative to applicationAssetsPath, or an empty value once an admin removed it
	assetKeyPrefix = "application_asset_"
)

// Init copies the bundled images for assets that were never set, which the upgrade from v2 leaves without a row
// Every replica runs it on startup, and the copies of two replicas write the same file and keep the row of the first one
func Init(ctx context.Context, db *gorm.DB, fileStorage storage.FileStorage) error {
	// Load the assets that are already recorded, including the ones an admin removed
	var rows []model.KV
	err := db.WithContext(ctx).Where("key LIKE ?", assetKeyPrefix+"%").Find(&rows).Error
	if err != nil {
		return fmt.Errorf("failed to load application assets: %w", err)
	}
	recorded := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		recorded[row.Key[len(assetKeyPrefix):]] = struct{}{}
	}

	bundledFiles, err := resources.FS.ReadDir("images")
	if err != nil {
		return fmt.Errorf("failed to read the bundled images: %w", err)
	}
	for _, bundledFile := range bundledFiles {
		name, _ := utils.SplitFileName(bundledFile.Name())
		if _, ok := recorded[name]; ok {
			continue
		}

		slog.InfoContext(ctx, "Writing new application image", slog.String("name", bundledFile.Name()))
		err = copyBundledImage(ctx, fileStorage, bundledFile.Name())
		if err != nil {
			return err
		}
		err = recordAsset(ctx, db, name, bundledFile.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

// recordAsset stores the file of an asset unless another replica recorded it first
func recordAsset(ctx context.Context, db *gorm.DB, name, file string) error {
	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.KV{Key: assetKeyPrefix + name, Value: &file}).
		Error
	if err != nil {
		return fmt.Errorf("failed to record application asset '%s': %w", name, err)
	}
	return nil
}

func copyBundledImage(ctx context.Context, fileStorage storage.FileStorage, fileName string) error {
	file, err := resources.FS.Open(path.Join("images", fileName))
	if err != nil {
		return fmt.Errorf("failed to open bundled image '%s': %w", fileName, err)
	}
	defer file.Close()

	err = fileStorage.Save(ctx, path.Join(applicationAssetsPath, fileName), file)
	if err != nil {
		return fmt.Errorf("failed to store application image '%s': %w", fileName, err)
	}
	return nil
}
