package profilepicture

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodePNG decodes a rendered profile picture into an image for pixel assertions
func decodePNG(t *testing.T, buf *bytes.Buffer) image.Image {
	t.Helper()
	img, err := png.Decode(buf)
	require.NoError(t, err)
	return img
}

// countInkPixels returns the number of pixels darker than the white background
func countInkPixels(img image.Image) int {
	count := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r|g|bl != 0xFFFF {
				count++
			}
		}
	}
	return count
}

// inkColumnExtent returns the horizontal range covered by ink pixels
func inkColumnExtent(img image.Image) (minX, maxX int) {
	b := img.Bounds()
	minX, maxX = -1, -1
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r|g|bl != 0xFFFF {
				if minX == -1 {
					minX = x
				}
				maxX = x
				break
			}
		}
	}
	return minX, maxX
}

// renderPNGBytes renders initials and returns the encoded PNG bytes
func renderPNGBytes(t *testing.T, initials string) []byte {
	t.Helper()
	buf, err := CreateDefaultProfilePicture(initials)
	require.NoError(t, err)
	return buf.Bytes()
}

// TestCreateDefaultProfilePictureRendersDistinctCJKGlyphs pins the fix for issue #1828
// The bundled display font covers Latin only, so on main every CJK rune maps to the same .notdef glyph
// Two initials built from the same runes in swapped order therefore render byte-identical pictures on main
// With the font fallback each rune renders its own glyph, so the swapped orders must produce different pictures
func TestCreateDefaultProfilePictureRendersDistinctCJKGlyphs(t *testing.T) {
	for _, runes := range [][2]string{
		{"漢字", "字漢"}, // Han
		{"あい", "いあ"}, // Hiragana
		{"한글", "글한"}, // Hangul
	} {
		forward := renderPNGBytes(t, runes[0])
		swapped := renderPNGBytes(t, runes[1])
		assert.NotEqual(t, forward, swapped,
			"initials %q and %q rendered identical pictures, so the runes are drawn as the same missing-glyph box", runes[0], runes[1])
	}
}

// TestCreateDefaultProfilePictureCJKInkIsRealGlyphs guards against a fallback that still draws boxes
// A tofu box outline concentrates its ink in long straight vertical runs, so comparing two different CJK initials must yield different pictures
func TestCreateDefaultProfilePictureCJKInkIsRealGlyphs(t *testing.T) {
	for _, initials := range []string{"漢字", "あいう", "한글"} {
		img := decodePNG(t, bytes.NewBuffer(renderPNGBytes(t, initials)))
		ink := countInkPixels(img)
		assert.Greater(t, ink, 1000, "initials %q produced too little ink", initials)

		minX, maxX := inkColumnExtent(img)
		assert.GreaterOrEqual(t, minX, 0, "initials %q produced no ink", initials)
		assert.LessOrEqual(t, maxX-minX, 300, "initials %q overflow the canvas width", initials)
	}
}

// TestCreateDefaultProfilePictureLatinUnchanged pins that Latin initials keep the bundled display font look
func TestCreateDefaultProfilePictureLatinUnchanged(t *testing.T) {
	img := decodePNG(t, bytes.NewBuffer(renderPNGBytes(t, "JD")))
	// A centered Latin pair fills a large share of the canvas, as before the fix
	assert.Greater(t, countInkPixels(img), 2000)
	minX, maxX := inkColumnExtent(img)
	assert.LessOrEqual(t, maxX-minX, 300, "Latin initials overflow the canvas")
}

// TestCreateDefaultProfilePictureMixedScript covers initials that span fonts
// A Latin rune renders through the display font while a CJK rune renders through the fallback, in one picture
func TestCreateDefaultProfilePictureMixedScript(t *testing.T) {
	forward := renderPNGBytes(t, "J中")
	swapped := renderPNGBytes(t, "中J")
	require.NotEqual(t, forward, swapped, "the Latin and CJK runes rendered as the same glyph")

	img := decodePNG(t, bytes.NewBuffer(forward))
	assert.Greater(t, countInkPixels(img), 1000)
}

// TestCreateDefaultProfilePictureEmptyStaysBlank pins the empty-initials behavior
func TestCreateDefaultProfilePictureEmptyStaysBlank(t *testing.T) {
	img := decodePNG(t, bytes.NewBuffer(renderPNGBytes(t, "")))
	assert.Equal(t, 0, countInkPixels(img))
}
