package profilepicture

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/pocket-id/pocket-id/backend/resources"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const profilePictureSize = 300

// CreateProfilePicture resizes the profile picture to a square
func CreateProfilePicture(file io.Reader) (*bytes.Buffer, error) {
	// Read all data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Check if this is a JPEG
	isJPEG := len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8

	// Decode image
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Extract orientation for JPEGs
	if isJPEG {
		// Simplified EXIF orientation extraction - you may want a more robust implementation
		orientation := getOrientationFromEXIF(data)

		// Apply orientation
		switch orientation {
		case 3:
			img = imaging.Rotate180(img)
		case 6:
			img = imaging.Rotate270(img)
		case 8:
			img = imaging.Rotate90(img)
		}
	}

	// Continue with resizing as before
	img = imaging.Fill(img, profilePictureSize, profilePictureSize, imaging.Center, imaging.Lanczos)

	var buf bytes.Buffer
	err = imaging.Encode(&buf, img, imaging.PNG)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}

	return &buf, nil
}

// Simple EXIF orientation extraction
func getOrientationFromEXIF(jpegData []byte) int {
	// Search for the EXIF APP1 marker
	// This is a simplified version and might not work for all JPEGs
	for i := 0; i < len(jpegData)-10; i++ {
		if jpegData[i] == 0xFF && jpegData[i+1] == 0xE1 {
			// Look for "Exif" string
			if string(jpegData[i+4:i+8]) == "Exif" {
				// Very basic EXIF parsing - this would need more robust implementation
				// in a production environment
				for j := i + 10; j < len(jpegData)-2; j++ {
					if jpegData[j] == 0x12 && jpegData[j+1] == 0x01 {
						// Found orientation tag
						return int(jpegData[j+9])
					}
				}
			}
		}
	}
	return 1 // Default orientation
}

// CreateDefaultProfilePicture creates a profile picture with the initials
func CreateDefaultProfilePicture(firstName, lastName string) (*bytes.Buffer, error) {
	// Get the initials
	initials := ""
	if len(firstName) > 0 {
		initials += string(firstName[0])
	}
	if len(lastName) > 0 {
		initials += string(lastName[0])
	}
	initials = strings.ToUpper(initials)

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
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}

	return &buf, nil
}
