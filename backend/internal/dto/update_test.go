package dto

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestMergePatchAppliesToCurrentState(t *testing.T) {
	type mapping struct {
		Claim   string `json:"claim"`
		IDToken bool   `json:"idToken"`
	}
	type credentials struct {
		Issuer   string `json:"issuer"`
		Audience string `json:"audience"`
	}
	type input struct {
		Name        string      `json:"name" binding:"required"`
		Description string      `json:"description"`
		Email       *string     `json:"email"`
		Mappings    []mapping   `json:"mappings"`
		Credentials credentials `json:"credentials"`
	}
	email := "current@example.com"
	current := input{
		Name:        "current",
		Description: "current",
		Email:       &email,
		Mappings:    []mapping{{Claim: "a", IDToken: true}},
		Credentials: credentials{Issuer: "current", Audience: "current"},
	}

	for _, test := range []struct {
		name  string
		patch string
		want  input
	}{
		{
			name:  "missing members keep their value",
			patch: `{"name":"changed"}`,
			want:  input{Name: "changed", Description: "current", Email: &email, Mappings: current.Mappings, Credentials: current.Credentials},
		},
		{
			name:  "null clears pointer and plain fields",
			patch: `{"email":null,"description":null}`,
			want:  input{Name: "current", Mappings: current.Mappings, Credentials: current.Credentials},
		},
		{
			name:  "nested objects merge",
			patch: `{"credentials":{"issuer":"changed"}}`,
			want:  input{Name: "current", Description: "current", Email: &email, Mappings: current.Mappings, Credentials: credentials{Issuer: "changed", Audience: "current"}},
		},
		{
			name:  "arrays replace their elements without keeping old fields",
			patch: `{"mappings":[{"claim":"b"}]}`,
			want:  input{Name: "current", Description: "current", Email: &email, Mappings: []mapping{{Claim: "b"}}, Credentials: current.Credentials},
		},
		{
			name:  "members match regardless of case",
			patch: `{"Name":"changed","Email":null,"Credentials":{"Issuer":"changed"}}`,
			want:  input{Name: "changed", Description: "current", Mappings: current.Mappings, Credentials: credentials{Issuer: "changed", Audience: "current"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var patch any
			require.NoError(t, DecodeJSON(strings.NewReader(test.patch), &patch))

			got, err := MergePatch[input](patch)(current)

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestMergePatchValidatesMergedResult(t *testing.T) {
	var patch any
	require.NoError(t, DecodeJSON(strings.NewReader(`{"name":null}`), &patch))

	_, err := MergePatch[struct {
		Name string `json:"name" binding:"required"`
	}](patch)(struct{ Name string }{Name: "current"})

	var validationErrors validator.ValidationErrors
	require.ErrorAs(t, err, &validationErrors)
}
