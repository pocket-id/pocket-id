package claimmappingpolicy

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

const builtInClaimMappingPolicyName = "Built-in Policy"

// builtInClaimMappings returns the standard OIDC claims released by the built-in policy
func builtInClaimMappings() OidcClaimMappings {
	return OidcClaimMappings{
		{ClaimName: "email", SourceType: MappingSourceUserField, SourceValue: string(UserFieldEmail), Scope: datatype.StringList{"email"}, IDToken: true, UserInfo: true},
		{ClaimName: "email_verified", SourceType: MappingSourceUserField, SourceValue: string(UserFieldEmailVerified), Scope: datatype.StringList{"email"}, IDToken: true, UserInfo: true},
		{ClaimName: "given_name", SourceType: MappingSourceUserField, SourceValue: string(UserFieldFirstName), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "family_name", SourceType: MappingSourceUserField, SourceValue: string(UserFieldLastName), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "name", SourceType: MappingSourceUserField, SourceValue: string(UserFieldFullName), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "display_name", SourceType: MappingSourceUserField, SourceValue: string(UserFieldDisplayName), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "preferred_username", SourceType: MappingSourceUserField, SourceValue: string(UserFieldUsername), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "locale", SourceType: MappingSourceUserField, SourceValue: string(UserFieldLocale), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "picture", SourceType: MappingSourceUserField, SourceValue: string(UserFieldPicture), Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
		{ClaimName: "groups", SourceType: MappingSourceUserField, SourceValue: string(UserFieldGroups), Scope: datatype.StringList{"groups"}, IDToken: true, UserInfo: true},
		{ClaimName: "*", SourceType: MappingSourceCustomClaim, SourceValue: "*", Scope: datatype.StringList{"profile"}, IDToken: true, UserInfo: true},
	}
}

// Bootstrap inserts the built-in claim mapping policy, or refreshes it when it already exists, so it always matches the running version
// It must run after the database migrations, since it writes into the table they create
func Bootstrap(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check whether a default policy exists, which only fails on a fresh database or when the default flag was lost
		var defaultCount int64
		err := tx.Model(&OidcClaimMappingPolicy{}).Where("is_default = ?", true).Count(&defaultCount).Error
		if err != nil {
			return err
		}
		noDefault := defaultCount == 0

		// The name and the mappings always follow the code, while the default flag belongs to the admin and is only claimed when no policy holds it
		updatedColumns := []string{"name", "claim_mappings"}
		if noDefault {
			updatedColumns = append(updatedColumns, "is_default")
		}

		// Upsert on the fixed ID so concurrent instances booting at the same time converge on a single row
		policy := OidcClaimMappingPolicy{
			Base:          model.Base{ID: BuiltInClaimMappingPolicyID},
			Name:          builtInClaimMappingPolicyName,
			IsDefault:     noDefault,
			ClaimMappings: builtInClaimMappings(),
		}
		return tx.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns(updatedColumns),
			}).
			Create(&policy).
			Error
	})
	if err != nil {
		return fmt.Errorf("failed to bootstrap the built-in claim mapping policy: %w", err)
	}

	return nil
}
