package humautils

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

// Register adds a typed operation while preserving Pocket ID error and body-reading behavior
func Register[I, O any](api huma.API, operation huma.Operation, handler func(context.Context, *I) (*O, error)) {
	if operation.MaxBodyBytes == 0 {
		operation.MaxBodyBytes = -1
	}
	if operation.BodyReadTimeout == 0 {
		operation.BodyReadTimeout = -1
	}
	huma.Register(api, operation, func(ctx context.Context, input *I) (*O, error) {
		output, err := handler(ctx, input)
		if err != nil {
			return nil, mapError(ctx, err)
		}
		return output, nil
	})
}
