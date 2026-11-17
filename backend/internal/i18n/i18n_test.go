package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		preference string
		want       *string
	}{
		{preference: "de-CH", want: new("de")},
		{preference: "pt_BR", want: new("pt-BR")},
		{preference: "nb-NO", want: new("no")},
		{preference: "no", want: new("no")},
		{preference: "zh-Hant-HK", want: new("zh-TW")},
		{preference: "fr, en;q=0.8", want: new("fr")},
		{preference: "xx", want: nil},
		{preference: "", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.preference, func(t *testing.T) {
			assert.Equal(t, tt.want, Match(tt.preference))
		})
	}
}
