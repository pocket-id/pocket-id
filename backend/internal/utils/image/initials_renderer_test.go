package profilepicture

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFontHasRuneFollowsFallbackChain pins which bundled font covers which script
func TestFontHasRuneFollowsFallbackChain(t *testing.T) {
	fonts, err := loadInitialsFonts()
	require.NoError(t, err)
	require.Len(t, fonts, 3)

	// The display font covers Latin but no CJK
	assert.True(t, fontHasRune(fonts[0], 'J'))
	assert.False(t, fontHasRune(fonts[0], '漢'))
	assert.False(t, fontHasRune(fonts[0], 'あ'))
	assert.False(t, fontHasRune(fonts[0], '한'))

	// The CJK font covers Han and Kana but not Hangul
	assert.True(t, fontHasRune(fonts[1], '漢'))
	assert.True(t, fontHasRune(fonts[1], '字'))
	assert.True(t, fontHasRune(fonts[1], 'あ'))
	assert.True(t, fontHasRune(fonts[1], 'ア'))
	assert.False(t, fontHasRune(fonts[1], '한'))

	// The Hangul font covers Hangul
	assert.True(t, fontHasRune(fonts[2], '한'))
	assert.True(t, fontHasRune(fonts[2], '글'))
	assert.False(t, fontHasRune(fonts[2], '漢'))
}

// TestResolveRuneFontsAssignsFirstCoveringFont pins the per-rune resolution order
func TestResolveRuneFontsAssignsFirstCoveringFont(t *testing.T) {
	fonts, err := loadInitialsFonts()
	require.NoError(t, err)

	// Mixed initials split across the chain in order
	resolved := resolveRuneFonts("J中한", fonts)
	require.Len(t, resolved, 3)
	assert.Equal(t, fonts[0], resolved[0].font)
	assert.Equal(t, fonts[1], resolved[1].font)
	assert.Equal(t, fonts[2], resolved[2].font)

	// Runes no font covers are dropped, so a tofu box can never be drawn
	resolved = resolveRuneFonts("J😀", fonts)
	require.Len(t, resolved, 1)
	assert.Equal(t, 'J', resolved[0].rune)
}

// TestBuildRunsGroupsConsecutiveSameFontRunes pins the run grouping that preserves intra-run kerning
func TestBuildRunsGroupsConsecutiveSameFontRunes(t *testing.T) {
	fonts, err := loadInitialsFonts()
	require.NoError(t, err)

	resolved := resolveRuneFonts("AJ中", fonts)
	require.Len(t, resolved, 3)

	runs, err := buildRuns(resolved, 100)
	require.NoError(t, err)
	require.Len(t, runs, 2, "AJ forms one Latin run and 中 forms one CJK run")
	assert.Equal(t, "AJ", runs[0].text)
	assert.Equal(t, "中", runs[1].text)
}

// TestFittedRunsScalesWidePairsIntoBudget pins the width fitting for wide CJK pairs
func TestFittedRunsScalesWidePairsIntoBudget(t *testing.T) {
	fonts, err := loadInitialsFonts()
	require.NoError(t, err)

	budget := float64(profilePictureSize) * maxInitialsWidthRatio

	// A wide CJK pair is scaled down into the budget
	runs, err := fittedRuns("漢字", fonts)
	require.NoError(t, err)
	width, err := measureRuns(runs)
	require.NoError(t, err)
	assert.LessOrEqual(t, width, budget+1, "the fitted CJK pair must not exceed the canvas budget")

	// A Latin pair fits without scaling and keeps the base size
	resolved := resolveRuneFonts("JD", fonts)
	baseRuns, err := buildRuns(resolved, baseInitialsFontSize)
	require.NoError(t, err)
	latinWidth, err := measureRuns(baseRuns)
	require.NoError(t, err)
	if latinWidth <= budget {
		// The unscaled width already fits, so fittedRuns must not shrink the size
		fitted, err := fittedRuns("JD", fonts)
		require.NoError(t, err)
		fittedWidth, err := measureRuns(fitted)
		require.NoError(t, err)
		assert.InDelta(t, latinWidth, fittedWidth, 2.0, "a fitting Latin pair must keep the base size")
	}
}

// TestFittedRunsEmptyInitialsReturnsNoRuns pins the empty-input contract
func TestFittedRunsEmptyInitialsReturnsNoRuns(t *testing.T) {
	fonts, err := loadInitialsFonts()
	require.NoError(t, err)

	runs, err := fittedRuns("", fonts)
	require.NoError(t, err)
	assert.Empty(t, runs)
}
