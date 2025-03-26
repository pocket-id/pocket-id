package transaction

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey struct{}

// AddToContext adds a *gorm.DB to the context.
func AddToContext(ctx context.Context, conn *gorm.DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, conn)
}

// FromContext returns a *gorm.DB from the context, or the default value if not present.
func FromContext(ctx context.Context, conn *gorm.DB) *gorm.DB {
	val, ok := ctx.Value(ctxKey{}).(*gorm.DB)
	if !ok {
		return conn
	}
	return val
}
