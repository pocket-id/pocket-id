package profilepicture

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/disintegration/imageorient"
	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/pocket-id/pocket-id/backend/resources"
)

const profilePictureSize = 300

// CreateProfilePicture resizes the profile picture to a square
func CreateProfilePicture(file io.Reader) (io.Reader, error) {
	img, _, err := imageorient.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	img = imaging.Fill(img, profilePictureSize, profilePictureSize, imaging.Center, imaging.Lanczos)

	pr, pw := io.Pipe()
	go func() {
		err = imaging.Encode(pw, img, imaging.PNG)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to encode image: %w", err))
			return
		}
		pw.Close()
	}()

	return pr, nil
}

// CreateDefaultProfilePicture creates a profile picture with the initials
func CreateDefaultProfilePicture(initials string) (*bytes.Buffer, error) {
	// Create a blank image with a white background
	img := imaging.New(profilePictureSize, profilePictureSize, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Load the font
	fontBytes, err := resources.FS.ReadFile("fonts/PlayfairDisplay-Bold.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read font file: %w", err)
	}

	// Parse the font
	fontFace, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	// Create a font.Face with a specific size
	fontSize := 160.0
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
	y := (profilePictureSize-face.Metrics().Height.Ceil())/2 + face.Metrics().Ascent.Ceil() - 10
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
