package openapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/require"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type widgetInput struct {
	Name      string             `json:"name" binding:"required"`
	Note      string             `json:"note"`
	Tags      []string           `json:"tags" binding:"required,min=1,dive,max=10"`
	Labels    []string           `json:"labels" binding:"min=1"`
	ExpiresAt *datatype.DateTime `json:"expiresAt"`
}

type widgetPage struct {
	Data []widgetInput `json:"data"`
}

func TestSpecDocumentsRequestsByTheirBinding(t *testing.T) {
	// The page route sorts after the others, so it would replace the input's component schema if the required fields were set while registering
	docs := []fuego.RouteOption{Body[widgetInput](), Response[widgetInput](http.StatusOK, "")}
	raw, err := Spec([]Route{
		{Method: http.MethodPut, Path: "/widgets/:id", Docs: append(docs, UpdateBody[widgetInput]())},
		{Method: http.MethodPatch, Path: "/widgets/:id", Docs: append(docs, UpdateBody[widgetInput]())},
		{Method: http.MethodGet, Path: "/widgets/page", Docs: []fuego.RouteOption{Response[widgetPage](http.StatusOK, "")}},
	})
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"nullable"`)

	var spec struct {
		Paths      map[string]map[string]map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Required   []string                  `json:"required"`
				Properties map[string]map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(raw, &spec))
	input := spec.Components.Schemas["widgetInput"]

	// Only fields whose binding requires them are required, and a slice that is required or needs a minimum length can't be null
	require.ElementsMatch(t, []string{"name", "tags"}, input.Required)
	require.Equal(t, "array", input.Properties["tags"]["type"])
	require.Equal(t, "array", input.Properties["labels"]["type"])

	// Rules before dive limit the slice and the rest limit its items
	require.InDelta(t, 1, input.Properties["tags"]["minItems"], 0)
	require.Equal(t, map[string]any{"type": "string", "maxLength": float64(10)}, input.Properties["tags"]["items"])

	// A pointer to a DateTime is a nullable date-time
	require.Equal(t, map[string]any{"type": []any{"string", "null"}, "format": "date-time"}, input.Properties["expiresAt"])

	// A PATCH is a merge patch that requires nothing
	patch := spec.Paths["/widgets/{id}"]["patch"]["requestBody"].(map[string]any)["content"].(map[string]any)
	require.Contains(t, patch, "application/merge-patch+json")
	require.NotContains(t, patch["application/json"].(map[string]any)["schema"], "required")
}

type gadget struct {
	Name      string `sortable:"case-insensitive"`
	Archived  *bool  `sortable:"true" filterable:"true"`
	Kind      string `filterable:"true"`
	Unindexed string
}

func TestListDocumentsTheModelsSortColumnsAndFilters(t *testing.T) {
	raw, err := Spec([]Route{{Method: http.MethodGet, Path: "/gadgets", Docs: []fuego.RouteOption{List[gadget]("ownerCount"), Response[[]gadget](http.StatusOK, "")}}})
	require.NoError(t, err)

	var spec struct {
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name   string         `json:"name"`
				Schema map[string]any `json:"schema"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(raw, &spec))
	parameters := map[string]map[string]any{}
	for _, parameter := range spec.Paths["/gadgets"]["get"].Parameters {
		parameters[parameter.Name] = parameter.Schema
	}

	// Sortable fields and the columns the service sorts by itself are the only sort columns
	require.Equal(t, []any{"archived", "name", "ownerCount"}, parameters["sort[column]"]["enum"])

	// Each filterable field is a repeatable filter typed by its field, also through a pointer
	require.Equal(t, map[string]any{"type": "boolean"}, parameters["filters[archived]"]["items"])
	require.Equal(t, map[string]any{"type": "string"}, parameters["filters[kind]"]["items"])
	require.NotContains(t, parameters, "filters[name]")
	require.NotContains(t, parameters, "filters[unindexed]")
}
