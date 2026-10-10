package profilepicture

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/boxes-ltd/imaging"
	"github.com/disintegration/imageorient"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/webp"
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

// CreateDefaultProfilePicture creates a profile picture with the initials
// Each rune is rendered by the first bundled font that covers it, so non-Latin initials do not degrade into tofu boxes
// See issue #1828
func CreateDefaultProfilePicture(initials string) (*bytes.Buffer, error) {
	// Create a blank image with a white background
	img := imaging.New(profilePictureSize, profilePictureSize, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	fonts, err := loadInitialsFonts()
	if err != nil {
		return nil, err
	}

	// Resolve runes to fonts and fit the text into the canvas, shrinking wide CJK pairs as needed
	runs, err := fittedRuns(initials, fonts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if len(runs) == 0 {
		// Nothing renderable in the initials, e.g. an empty string, so return the blank canvas
		err = imaging.Encode(&buf, img, imaging.PNG)
		if err != nil {
			return nil, fmt.Errorf("failed to encode image: %w", err)
		}
		return &buf, nil
	}

	// Create a drawer for the image
	drawer := &font.Drawer{
		Dst: img,
		Src: image.NewUniform(color.RGBA{R: 0, G: 0, B: 0, A: 255}), // Black text color
	}

	// Center the whole run sequence horizontally and place the shared baseline vertically
	width := 0.0
	for _, run := range runs {
		w, err := measureRun(run)
		if err != nil {
			return nil, err
		}
		width += w
	}
	ascent := maxRunAscent(runs)
	descent := maxRunDescent(runs)
	x := (float64(profilePictureSize) - width) / 2
	y := (float64(profilePictureSize)-float64(ascent+descent))/2 + float64(ascent)
	drawer.Dot = fixed.P(int(x), int(y))

	// Draw each run with its own face, advancing the dot run by run
	for _, run := range runs {
		drawer.Face = run.face
		drawer.DrawString(run.text)
	}

	err = imaging.Encode(&buf, img, imaging.PNG)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return &buf, nil
}
