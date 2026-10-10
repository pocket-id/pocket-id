package profilepicture

import (
	"fmt"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/pocket-id/pocket-id/backend/resources"
)

// The size of the drawn initials before any downscaling is applied
// A generous base size keeps wide CJK glyph pairs crisp after fitting them into the canvas
const baseInitialsFontSize = 160.0

// The fraction of the canvas the drawn initials may occupy at most
// A margin keeps wide CJK pairs from touching the picture border
const maxInitialsWidthRatio = 0.8

// initialsFaceDPI is the resolution the initials faces are created at
// 72 DPI makes one point equal one pixel, so font sizes translate to pixel sizes directly
const initialsFaceDPI = 72.0

// initialsFonts holds the fallback chain used to render initials, in priority order
// A rune is rendered by the first font in the chain that has a glyph for it
// Latin runes keep the Gloock look while CJK runes fall back to Noto Sans SC or Noto Sans KR
var initialsFonts []*opentype.Font

var initialsFontsOnce sync.Once

var initialsFontsErr error

// loadInitialsFonts parses the bundled fonts once per process instead of on every generated picture
func loadInitialsFonts() ([]*opentype.Font, error) {
	initialsFontsOnce.Do(func() {
		for _, name := range []string{"fonts/Gloock.ttf", "fonts/NotoSansSC-Regular.otf", "fonts/NotoSansKR-Regular.otf"} {
			fontBytes, err := resources.FS.ReadFile(name)
			if err != nil {
				initialsFontsErr = fmt.Errorf("failed to read font file %s: %w", name, err)
				return
			}
			parsed, err := opentype.Parse(fontBytes)
			if err != nil {
				initialsFontsErr = fmt.Errorf("failed to parse font file %s: %w", name, err)
				return
			}
			initialsFonts = append(initialsFonts, parsed)
		}
	})
	return initialsFonts, initialsFontsErr
}

// fontHasRune reports whether the font provides a glyph other than .notdef for the rune
// Glyph index 0 is the .notdef glyph, which is what a tofu box is rendered from
func fontHasRune(f *opentype.Font, r rune) bool {
	idx, err := f.GlyphIndex(nil, r)
	return err == nil && idx != 0
}

// runeFont binds a rune to the bundled font that covers it
type runeFont struct {
	rune rune
	font *opentype.Font
}

// resolveRuneFonts assigns every rune of the initials to the first bundled font that has a glyph for it
// Runes that no bundled font covers are dropped, so a tofu box can never be drawn
func resolveRuneFonts(initials string, fonts []*opentype.Font) []runeFont {
	resolved := make([]runeFont, 0, len(initials))
	for _, r := range initials {
		for _, f := range fonts {
			if fontHasRune(f, r) {
				resolved = append(resolved, runeFont{rune: r, font: f})
				break
			}
		}
	}
	return resolved
}

// initialsRun is a maximal span of consecutive runes resolved to the same font
// Drawing a whole run through one face keeps kerning between the runes of the run, e.g. between Latin letters
type initialsRun struct {
	text string
	face font.Face
}

// buildRuns groups resolved runes into runs and creates the face for each run at the requested size
func buildRuns(resolved []runeFont, size float64) ([]initialsRun, error) {
	var runs []initialsRun
	for i := 0; i < len(resolved); {
		// Find the end of the run of consecutive runes resolved to the same font
		j := i
		for j < len(resolved) && resolved[j].font == resolved[i].font {
			j++
		}

		runes := make([]rune, 0, j-i)
		for _, rf := range resolved[i:j] {
			runes = append(runes, rf.rune)
		}

		face, err := opentype.NewFace(resolved[i].font, &opentype.FaceOptions{Size: size, DPI: initialsFaceDPI})
		if err != nil {
			return nil, fmt.Errorf("failed to create font face: %w", err)
		}

		runs = append(runs, initialsRun{text: string(runes), face: face})
		i = j
	}
	return runs, nil
}

// measureRun measures the pixel width of a single run, kerning included
func measureRun(run initialsRun) (float64, error) {
	width := 0.0
	prev := rune(-1)
	for _, r := range run.text {
		if prev != rune(-1) {
			width += float64(run.face.Kern(prev, r)) / 64.0
		}
		advance, ok := run.face.GlyphAdvance(r)
		if !ok {
			return 0, fmt.Errorf("glyph advance missing for %q", r)
		}
		width += float64(advance) / 64.0
		prev = r
	}
	return width, nil
}

// measureRuns returns the combined pixel width of all runs
// Kerning is not applied across run boundaries because each boundary switches fonts
func measureRuns(runs []initialsRun) (float64, error) {
	total := 0.0
	for _, run := range runs {
		w, err := measureRun(run)
		if err != nil {
			return 0, err
		}
		total += w
	}
	return total, nil
}

// maxRunAscent returns the largest ascent among the runs' faces, in pixels
// The shared baseline must sit below the tallest glyphs of every script in the initials
func maxRunAscent(runs []initialsRun) int {
	max := 0
	for _, run := range runs {
		if a := run.face.Metrics().Ascent.Ceil(); a > max {
			max = a
		}
	}
	return max
}

// maxRunDescent returns the largest descent among the runs' faces, in pixels
// CJK faces typically report little descent, so the Latin face usually dominates here
func maxRunDescent(runs []initialsRun) int {
	max := 0
	for _, run := range runs {
		if d := run.face.Metrics().Descent.Ceil(); d > max {
			max = d
		}
	}
	return max
}

// fittedRuns builds runs at the largest font size whose rendered width fits the canvas budget
// Wide CJK pairs overflow the 300 pixel canvas at the base size, so the size is scaled down to fit
func fittedRuns(initials string, fonts []*opentype.Font) ([]initialsRun, error) {
	resolved := resolveRuneFonts(initials, fonts)
	if len(resolved) == 0 {
		return nil, nil
	}

	budget := float64(profilePictureSize) * maxInitialsWidthRatio
	size := baseInitialsFontSize

	// Measure at the base size first so the common Latin case never iterates more than once
	runs, err := buildRuns(resolved, size)
	if err != nil {
		return nil, err
	}
	width, err := measureRuns(runs)
	if err != nil {
		return nil, err
	}

	// Scale the size linearly when the measured width exceeds the budget
	// Glyph advances scale proportionally with the face size, so one proportional step lands exactly on the budget
	if width > budget {
		size = size * budget / width
		runs, err = buildRuns(resolved, size)
		if err != nil {
			return nil, err
		}
	}

	return runs, nil
}
