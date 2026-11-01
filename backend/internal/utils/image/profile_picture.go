package profilepicture

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"
	"sync"

	"github.com/boxes-ltd/imaging"
	"github.com/disintegration/imageorient"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/webp"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

const profilePictureSize = 300

var ErrInvalidImage = errors.New("invalid image")

var resizeSlots = make(chan struct{}, 2)

// CreateProfilePicture resizes the profile picture to a square and encodes it as PNG
func CreateProfilePicture(file io.ReadSeeker) (io.ReadSeeker, error) {
	// Reject an oversized pixel count before decoding can allocate a pixel buffer
	validationErr := validateImageDimensions(file)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}
	if validationErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidImage, validationErr)
	}

	resizeSlots <- struct{}{}
	defer func() { <-resizeSlots }()

	// Attempt standard formats first
	img, _, err := imageorient.Decode(file)
	if err != nil {
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			return nil, fmt.Errorf("failed to seek file: %w", seekErr)
		}
		// Try WebP
		webpImg, webpErr := webp.Decode(file)
		if webpErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidImage, errors.Join(
				fmt.Errorf("standard formats: %w", err),
				fmt.Errorf("WebP: %w", webpErr),
			))
		}

		img = webpImg
	}

	// Resize to square
	img = imaging.Resize(centerSquare(img), profilePictureSize, profilePictureSize, imaging.Lanczos)

	// Encode back to PNG
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func centerSquare(img image.Image) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())

	sub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		return imaging.CropCenter(img, side, side)
	}

	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	return sub.SubImage(image.Rect(x, y, x+side, y+side))
}

var loadInitialsFont = sync.OnceValues(func() (*opentype.Font, error) {
	fontBytes, err := resources.FS.ReadFile("fonts/FunnelDisplay-SemiBold.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read font file: %w", err)
	}

	fontFace, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	return fontFace, nil
})

// Initials returns the initials shown on the default profile picture
func Initials(firstName, lastName, username string) string {
	// The font only covers Latin, so names it can't render fall back to the username, which is always ASCII
	initials := strings.ToUpper(utils.GetFirstCharacter(firstName) + utils.GetFirstCharacter(lastName))
	if initials != "" && canRenderInitials(initials) {
		return initials
	}

	return strings.ToUpper(utils.GetFirstCharacter(username))
}

// canRenderInitials reports whether the initials font has a glyph for every character of the initials
// The font only covers Latin, so other scripts would otherwise be drawn as empty boxes
func canRenderInitials(initials string) bool {
	fontFace, err := loadInitialsFont()
	if err != nil {
		return false
	}

	for _, r := range initials {
		idx, err := fontFace.GlyphIndex(nil, r)
		if err != nil || idx == 0 {
			return false
		}
	}

	return true
}

// DefaultPictureFile returns the cache file name of the default picture for the initials
// The suffix changes whenever the rendering changes, so the cleanup job deletes pictures cached by an earlier version
func DefaultPictureFile(initials string) string {
	return initials + "-v3.png"
}

// CreateDefaultProfilePicture creates a profile picture with the initials
func CreateDefaultProfilePicture(initials string) (*bytes.Buffer, error) {
	// Create a blank image with a white background
	img := imaging.New(profilePictureSize, profilePictureSize, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Load the font
	fontFace, err := loadInitialsFont()
	if err != nil {
		return nil, err
	}

	// Create a font.Face with a specific size
	// Clients often crop the picture to a circle, so the initials keep clear of its edge
	fontSize := 120.0
	face, err := opentype.NewFace(fontFace, &opentype.FaceOptions{
		Size: fontSize,
		DPI:  72,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create font face: %w", err)
	}

	// Create a drawer for the image
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 0, G: 0, B: 0, A: 255}), // Black text color
		Face: face,
	}

	// Center the initials
	x := (profilePictureSize - font.MeasureString(face, initials).Ceil()) / 2
	y := (profilePictureSize-face.Metrics().Height.Ceil())/2 + face.Metrics().Ascent.Ceil()
	drawer.Dot = fixed.P(x, y)

	// Draw the initials
	drawer.DrawString(initials)

	var buf bytes.Buffer
	err = imaging.Encode(&buf, img, imaging.PNG)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return &buf, nil
}
