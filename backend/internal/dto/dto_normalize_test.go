package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/unicode/norm"
)

type testDto struct {
	Name        string `unorm:"nfc"`
	Description string `unorm:"nfd"`
	Other       string
	BadForm     string `unorm:"bad"`
}

func TestNormalize(t *testing.T) {
	input := testDto{
		// Is in NFC form already
		Name: norm.NFC.String("Café"),
		// NFC form will be normalized to NFD
		Description: norm.NFC.String("vërø"),
		// Should be unchanged
		Other: "NöTag",
		// Should be unchanged
		BadForm: "BåD",
	}

	Normalize(&input)

	assert.Equal(t, norm.NFC.String("Café"), input.Name)
	assert.Equal(t, norm.NFD.String("vërø"), input.Description)
	assert.Equal(t, "NöTag", input.Other)
	assert.Equal(t, "BåD", input.BadForm)
}
