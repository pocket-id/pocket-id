package api

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

const builtInAPIName = "Pocket ID"

// Bootstrap inserts the built-in API that represents Pocket ID itself, or refreshes it when it already exists, so its resource follows APP_URL and its permissions follow the scope catalog
// It must run after the database migrations, since it writes into the tables they create
func Bootstrap(ctx context.Context, db *gorm.DB, appURL string) error {
	audience := authz.ResourceURI(appURL)
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A custom API created before the resource was reserved can hold it, and failing here would keep the instance from starting
		var conflicts int64
		err := tx.Model(&API{}).Where("audience = ? AND id <> ?", audience, BuiltInAPIID).Count(&conflicts).Error
		if err != nil {
			return err
		}
		if conflicts > 0 {
			slog.WarnContext(ctx, "Pocket ID's built-in API is unavailable because a custom API already uses its resource; delete that API to enable it", slog.String("resource", audience))
			return nil
		}

		// Upsert on the fixed ID so concurrent instances booting at the same time converge on a single row, leaving the CIMD opt-in to the admin
		api := API{
			Base:     model.Base{ID: BuiltInAPIID},
			Name:     builtInAPIName,
			Audience: audience,
		}
		err = tx.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "audience"}),
			}).
			Create(&api).
			Error
		if err != nil {
			return err
		}

		// Every scope an OAuth token may carry for a user becomes a permission, which the frontend labels by its key
		scopes := authz.UserScopes(true, authz.KindOAuthUser)
		permissions := make([]Permission, 0, len(scopes))
		keys := make([]string, 0, len(scopes))
		for scope := range scopes {
			permissions = append(permissions, Permission{APIID: BuiltInAPIID, Key: string(scope), Name: string(scope)})
			keys = append(keys, string(scope))
		}
		err = tx.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "api_id"}, {Name: "key"}},
				DoNothing: true,
			}).
			Create(&permissions).
			Error
		if err != nil {
			return err
		}

		// Remove permissions whose scope left the catalog, together with the grants and access they implied
		var removedIDs []string
		err = tx.
			Model(&Permission{}).
			Where("api_id = ? AND key NOT IN ?", BuiltInAPIID, keys).
			Pluck("id", &removedIDs).
			Error
		if err != nil {
			return err
		}
		return deletePermissions(ctx, tx, removedIDs)
	})
	if err != nil {
		return fmt.Errorf("failed to bootstrap the built-in Pocket ID API: %w", err)
	}

	return nil
}
