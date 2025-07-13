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

func TestNormalizeSlice(t *testing.T) {
	obj1 := testDto{
		Name:        norm.NFC.String("Café1"),
		Description: norm.NFC.String("vërø1"),
		Other:       "NöTag1",
		BadForm:     "BåD1",
	}
	obj2 := testDto{
		Name:        norm.NFD.String("Résumé2"),
		Description: norm.NFD.String("accéléré2"),
		Other:       "NöTag2",
		BadForm:     "BåD2",
	}

	t.Run("slice of structs", func(t *testing.T) {
		slice := []testDto{obj1, obj2}
		Normalize(&slice)

		// Verify first element
		assert.Equal(t, norm.NFC.String("Café1"), slice[0].Name)
		assert.Equal(t, norm.NFD.String("vërø1"), slice[0].Description)
		assert.Equal(t, "NöTag1", slice[0].Other)
		assert.Equal(t, "BåD1", slice[0].BadForm)

		// Verify second element
		assert.Equal(t, norm.NFC.String("Résumé2"), slice[1].Name)
		assert.Equal(t, norm.NFD.String("accéléré2"), slice[1].Description)
		assert.Equal(t, "NöTag2", slice[1].Other)
		assert.Equal(t, "BåD2", slice[1].BadForm)
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		slice := []*testDto{&obj1, &obj2}
		Normalize(&slice)

		// Verify first element
		assert.Equal(t, norm.NFC.String("Café1"), slice[0].Name)
		assert.Equal(t, norm.NFD.String("vërø1"), slice[0].Description)
		assert.Equal(t, "NöTag1", slice[0].Other)
		assert.Equal(t, "BåD1", slice[0].BadForm)

		// Verify second element
		assert.Equal(t, norm.NFC.String("Résumé2"), slice[1].Name)
		assert.Equal(t, norm.NFD.String("accéléré2"), slice[1].Description)
		assert.Equal(t, "NöTag2", slice[1].Other)
		assert.Equal(t, "BåD2", slice[1].BadForm)
	})
}
