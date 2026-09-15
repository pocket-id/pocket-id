package profilepicture

import (
	"errors"
	"fmt"
	"image"
	"io"
)

const maxImagePixels = 16_000_000 // e.g. 4000x4000 pixels

var errImageDimensionsTooLarge = errors.New("image dimensions exceed the allowed limit")

// validateImageDimensions checks if the image dimensions exceed the maximum allowed pixel count.
func validateImageDimensions(r io.Reader) error {
	config, _, err := image.DecodeConfig(r)
	if err != nil {
		return err
	}

	if int64(config.Width)*int64(config.Height) > maxImagePixels {
		return fmt.Errorf("%w: got %dx%d, maximum pixel count is %d", errImageDimensionsTooLarge, config.Width, config.Height, maxImagePixels)
	}

	return nil
}
