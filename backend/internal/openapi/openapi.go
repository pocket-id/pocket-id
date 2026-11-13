// Package openapi builds the OpenAPI spec of the API from the documentation that routes declare when they are registered
// Pocket ID doesn't serve the spec, the documentation site renders its API reference from the output of cmd/openapi
package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// tags lists the groups of the API reference in the order a reader meets them in the product, each with the description shown under it
var tags = openapi3.Tags{
	{Name: "Users", Description: "Users, their passkeys and profile pictures, login codes and signup tokens."},
	{Name: "User Groups", Description: "Groups and their members."},
	{Name: "OIDC", Description: "OIDC clients with their secrets, logos and allowed groups, plus the clients each user has authorized."},
	{Name: "OIDC Custom Scopes", Description: "Scopes of your own that clients can request besides the built-in ones."},
	{Name: "OIDC Claim Mapping Policies", Description: "Policies that decide which claims the tokens of a scope carry and who may request it."},
	{Name: "APIs", Description: "Your own APIs, their permissions and the clients that may request them."},
	{Name: "Custom Claims", Description: "Extra claims that Pocket ID adds to the tokens of a user or of every member of a group."},
	{Name: "API Keys", Description: "Keys for this REST API."},
	{Name: "Sessions", Description: "The sign-in sessions of the current user or of any user."},
	{Name: "Application Configuration", Description: "The settings of the Application Configuration page, the LDAP sync and the test email."},
	{Name: "Application Assets", Description: "The logo, favicon, background, font, email logo and default profile picture."},
	{Name: "Features", Description: "The optional features and whether they are on."},
	{Name: "Audit Logs", Description: "Sign-ins and other security events, of the current user or of everyone."},
	{Name: "SCIM", Description: "SCIM provisioning of an OIDC client."},
	{Name: "Device Login", Description: "The requests behind sign-in with another device."},
	{Name: "Well Known", Description: "Discovery documents and signing keys, which OIDC clients read without authentication."},
	{Name: "Version", Description: "The running version and the latest release."},
	{Name: "Health", Description: "A health check for container orchestrators and load balancers."},
	{Name: "Storage", Description: "Storage warnings for the admin UI."},
}

var ginParam = regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

// Route is a registered route with the options that document it, where a route without options stays out of the spec
type Route struct {
	Method string
	Path   string
	Docs   []fuego.RouteOption
}

// Spec returns the OpenAPI spec of the documented routes as indented JSON
func Spec(routes []Route) ([]byte, error) {
	// The customizer records the Go type of every struct, so the component schemas of request bodies can be traced back to their binding tags
	types := map[string]reflect.Type{}
	customize := func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if t.Kind() == reflect.Struct {
			types[t.Name()] = t
		}
		return customizeSchema(name, t, tag, schema)
	}

	engine := fuego.NewEngine(
		fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
			DisableLocalSave: true,
			DisableMessages:  true,
			Info:             &openapi3.Info{Title: "Pocket ID API", Description: "The REST API of Pocket ID.", Version: common.Version},
			MiddlewareConfig: fuego.MiddlewareConfig{DisableControllerSection: true, DisableMiddlewareSection: true},
		}),
		fuego.WithRequestContentType("application/json"),
		fuego.WithResponseContentType("application/json"),
		fuego.WithOpenAPIGeneratorSchemaCustomizer(customize),
	)

	// Every error has the shape of ErrorDto, so each operation documents it once as its default response
	errorResponse := option.DefaultResponse("Error", fuego.Response{Type: dto.ErrorDto{}, ContentTypes: []string{"application/json"}})

	// openapi3gen replaces a component schema whenever another type nests it, so a stable order keeps the output stable
	slices.SortFunc(routes, func(a, b Route) int {
		return strings.Compare(a.Path+" "+a.Method, b.Path+" "+b.Method)
	})
	for _, route := range routes {
		if len(route.Docs) == 0 {
			continue
		}

		// Fuego takes a handler only to name the operation, which every documented route does with its summary
		path := ginParam.ReplaceAllString(route.Path, "{$1}")
		r := fuego.Route[any, any, any]{BaseRoute: fuego.NewBaseRoute(route.Method, path, Spec, engine, append(route.Docs, errorResponse)...)}
		if err := r.RegisterOpenAPIOperation(engine.OpenAPI); err != nil {
			return nil, fmt.Errorf("failed to document %s %s: %w", route.Method, route.Path, err)
		}
	}

	// Fuego registers a placeholder schema for the untyped bodies of gin handlers, which every route replaces with its own documentation
	spec := engine.OutputOpenAPISpec()
	delete(spec.Components.Schemas, "unknown-interface")
	spec.Tags = tags

	// Fuego describes every component schema as "<name> schema", which tells a reader nothing
	for name, schema := range spec.Components.Schemas {
		if schema.Value.Description == name+" schema" {
			schema.Value.Description = ""
		}
	}

	// Fuego marks every field without omitempty as required, which holds for responses, but a request only has to send the fields its binding requires
	// This runs last because openapi3gen would replace a component schema adjusted earlier
	requestTypes := map[reflect.Type]bool{}
	for _, path := range spec.Paths.Map() {
		for _, operation := range path.Operations() {
			if operation.RequestBody == nil {
				continue
			}
			for _, media := range operation.RequestBody.Value.Content {
				if t, ok := types[strings.TrimPrefix(media.Schema.Ref, "#/components/schemas/")]; ok {
					requireBoundFields(spec.Components.Schemas, t, requestTypes)
				}
			}
		}
	}

	// openapi3gen marks nullable values with the OpenAPI 3.0 keyword, which a 3.1 document expresses as a null type instead
	visited := map[*openapi3.Schema]bool{}
	for _, schema := range spec.Components.Schemas {
		nullAsType(schema, visited)
	}
	for _, path := range spec.Paths.Map() {
		for _, operation := range path.Operations() {
			for _, parameter := range operation.Parameters {
				nullAsType(parameter.Value.Schema, visited)
			}
			if operation.RequestBody != nil {
				for _, media := range operation.RequestBody.Value.Content {
					nullAsType(media.Schema, visited)
				}
			}
			for _, response := range operation.Responses.Map() {
				for _, media := range response.Value.Content {
					nullAsType(media.Schema, visited)
				}
			}
		}
	}

	if err := spec.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}
	return json.MarshalIndent(spec, "", "  ")
}

// customizeSchema adapts the generated schemas to how Pocket ID declares its DTOs
func customizeSchema(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	// Raw JSON holds any value, while openapi3gen would describe it as a byte string
	if t == reflect.TypeFor[json.RawMessage]() {
		*schema = openapi3.Schema{}
		return nil
	}

	// DateTime and JSONDuration marshal to strings, while openapi3gen sees structs without fields and references component schemas it never defines
	// A struct's schema is customized after its properties, so they are replaced there, where the field still tells whether it can be null
	if t.Kind() == reflect.Struct {
		for _, field := range jsonFields(t) {
			fieldType, nullable := field.Type, field.Type.Kind() == reflect.Pointer
			if nullable {
				fieldType = fieldType.Elem()
			}

			var property *openapi3.Schema
			switch fieldType {
			case reflect.TypeFor[datatype.DateTime]():
				property = openapi3.NewDateTimeSchema()
			case reflect.TypeFor[utils.JSONDuration]():
				property = &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString, openapi3.TypeNumber}, Description: "A duration such as 15m or 24h, or a number of seconds"}
			default:
				continue
			}
			property.Nullable = nullable
			schema.Properties[field.Name] = property.NewRef()
		}
		return nil
	}

	// DTOs declare their validation in gin's binding tag, while fuego reads limits and enums from the validate tag
	binding, ok := tag.Lookup("binding")
	if !ok {
		return nil
	}

	// Rules before dive apply to a slice itself and the rest to its items, while openapi3gen passes the whole tag to the items too
	own, items, _ := strings.Cut(binding, ",dive")
	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return fuego.SchemaCustomizer(name, t, reflect.StructTag(fmt.Sprintf("validate:%q", own)), schema)
	}
	for rule := range strings.SplitSeq(own, ",") {
		if n, ok := strings.CutPrefix(rule, "min="); ok {
			schema.MinItems, _ = strconv.ParseUint(n, 10, 64)
		}
		if n, ok := strings.CutPrefix(rule, "max="); ok {
			maxItems, _ := strconv.ParseUint(n, 10, 64)
			schema.MaxItems = &maxItems
		}
	}
	if schema.Items == nil || schema.Items.Value == nil || strings.HasPrefix(schema.Items.Ref, "#/") {
		return nil
	}
	item := schema.Items.Value
	item.MinLength, item.MaxLength, item.Min, item.Max, item.Enum = 0, nil, nil, nil, nil
	return fuego.SchemaCustomizer(name, t.Elem(), reflect.StructTag(fmt.Sprintf("validate:%q", strings.TrimPrefix(items, ","))), item)
}

// jsonFields returns the fields of a struct as they appear in its JSON, including those of embedded structs
func jsonFields(t reflect.Type) []reflect.StructField {
	var fields []reflect.StructField
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if field.Anonymous && name == "" && field.Type.Kind() == reflect.Struct {
			fields = append(fields, jsonFields(field.Type)...)
			continue
		}
		if name == "-" || name == "" {
			continue
		}
		field.Name = name
		fields = append(fields, field)
	}
	return fields
}

// nullAsType replaces the nullable keyword with a null type throughout a schema
func nullAsType(ref *openapi3.SchemaRef, visited map[*openapi3.Schema]bool) {
	if ref == nil || ref.Value == nil || visited[ref.Value] {
		return
	}
	schema := ref.Value
	visited[schema] = true

	if schema.Nullable {
		schema.Nullable = false
		if schema.Type != nil && !schema.Type.Includes(openapi3.TypeNull) {
			types := append(schema.Type.Slice(), openapi3.TypeNull)
			schema.Type = (*openapi3.Types)(&types)
		}
	}
	for _, property := range schema.Properties {
		nullAsType(property, visited)
	}
	nullAsType(schema.Items, visited)
	nullAsType(schema.AdditionalProperties.Schema, visited)
}

// List documents the pagination, sorting and filtering query parameters that utils.PaginateFilterAndSort applies to the model M
// The sort columns and filters come from the model's sortable and filterable tags, and extraSortColumns are the columns a service sorts by itself
func List[M any](extraSortColumns ...string) fuego.RouteOption {
	fields := utils.ModelMetadata(new(M))
	sortColumns := slices.Clone(extraSortColumns)
	var filters []fuego.RouteOption
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		// The query names a field in lower camel case, which utils capitalizes again to find it
		field, key := fields[name], strings.ToLower(name[:1])+name[1:]
		if field.IsSortable {
			sortColumns = append(sortColumns, key)
		}
		if field.IsFilterable {
			kind := field.Type.Kind()
			if kind == reflect.Pointer {
				kind = field.Type.Elem().Kind()
			}
			filters = append(filters, option.QueryArray("filters["+key+"]", "Only items whose "+key+" is one of these values", kind))
		}
	}
	slices.Sort(sortColumns)

	return option.Group(append([]fuego.RouteOption{
		option.QueryInt("pagination[page]", "Page number for pagination", param.Default(1)),
		option.QueryInt("pagination[limit]", "Number of items per page", param.Default(20)),
		enumQuery("sort[column]", "Column to sort by", sortColumns, nil),
		enumQuery("sort[direction]", "Sort direction", []string{"asc", "desc"}, "asc"),
	}, filters...)...)
}

// enumQuery documents a string query parameter that accepts only the values
func enumQuery(name, description string, values []string, defaultValue any) fuego.RouteOption {
	schema := openapi3.NewStringSchema()
	schema.Default = defaultValue
	for _, value := range values {
		schema.Enum = append(schema.Enum, value)
	}
	return func(r *fuego.BaseRoute) {
		r.Operation.AddParameter(&openapi3.Parameter{Name: name, In: openapi3.ParameterInQuery, Description: description, Schema: schema.NewRef()})
	}
}

// Body documents a request body of type T, which is JSON unless other content types are given
func Body[T any](contentTypes ...string) fuego.RouteOption {
	if len(contentTypes) == 0 {
		contentTypes = []string{"application/json"}
	}
	return func(r *fuego.BaseRoute) {
		r.Operation.RequestBody = &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.NewContentWithSchemaRef(schemaRef(r, *new(T)), contentTypes))}
	}
}

// UpdateBody documents the body of an update with type T, which a PUT replaces as a whole and a PATCH merges as a JSON merge patch (RFC 7396)
func UpdateBody[T any]() fuego.RouteOption {
	return func(r *fuego.BaseRoute) {
		Body[T]()(r)
		if r.Method != http.MethodPatch {
			return
		}

		// A merge patch carries only the fields it changes, so none of them is required
		ref := r.Operation.RequestBody.Value.Content.Get("application/json").Schema
		patch := *r.OpenAPI.Description().Components.Schemas[strings.TrimPrefix(ref.Ref, "#/components/schemas/")].Value
		patch.Required, patch.Description = nil, ""
		r.Operation.RequestBody.Value.Content = openapi3.NewContentWithSchema(&patch, []string{"application/merge-patch+json", "application/json"})
	}
}

// requireBoundFields marks the fields of a request type and its nested types as required exactly when their binding requires them
func requireBoundFields(schemas openapi3.Schemas, t reflect.Type, visited map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	schema, ok := schemas[t.Name()]
	if t.Kind() != reflect.Struct || !ok || visited[t] {
		return
	}
	visited[t] = true

	schema.Value.Required = nil
	for _, field := range jsonFields(t) {
		own, _, _ := strings.Cut(field.Tag.Get("binding"), ",dive")
		rules := strings.Split(own, ",")
		required := slices.Contains(rules, "required")
		if required {
			schema.Value.Required = append(schema.Value.Required, field.Name)
		}

		// Fuego lets a slice or map be null, which the binding rejects when the field is required or needs a minimum length
		collection := field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map
		if required || collection && slices.ContainsFunc(rules, func(rule string) bool { return strings.HasPrefix(rule, "min=") && rule != "min=0" }) {
			if property := schema.Value.Properties[field.Name]; property != nil && property.Value != nil && property.Value.Type != nil {
				types := slices.DeleteFunc(property.Value.Type.Slice(), func(t string) bool { return t == openapi3.TypeNull })
				property.Value.Type = (*openapi3.Types)(&types)
			}
		}
		requireBoundFields(schemas, field.Type, visited)
	}
}

// Response documents a successful JSON response of type T, whose status becomes the operation's success status
func Response[T any](status int, description string) fuego.RouteOption {
	return func(r *fuego.BaseRoute) {
		r.DefaultStatusCode = status
		r.Operation.AddResponse(status, openapi3.NewResponse().
			WithDescription(orStatusText(description, status)).
			WithJSONSchemaRef(schemaRef(r, *new(T))))
	}
}

// Status documents a successful response without a body, such as 204 No Content
func Status(status int, description string) fuego.RouteOption {
	return func(r *fuego.BaseRoute) {
		r.DefaultStatusCode = status

		// An empty but present content keeps fuego from adding a placeholder body
		response := openapi3.NewResponse().WithDescription(orStatusText(description, status))
		response.Content = openapi3.Content{}
		r.Operation.AddResponse(status, response)
	}
}

// File documents a successful response that carries a file in one of the content types
func File(description string, contentTypes ...string) fuego.RouteOption {
	return func(r *fuego.BaseRoute) {
		schema := openapi3.NewStringSchema().WithFormat("binary")
		r.Operation.AddResponse(http.StatusOK, openapi3.NewResponse().
			WithDescription(description).
			WithContent(openapi3.NewContentWithSchema(schema, contentTypes)))
	}
}

// Upload documents a multipart request body that carries a single file in its file field
func Upload(description string) fuego.RouteOption {
	return func(r *fuego.BaseRoute) {
		file := openapi3.NewStringSchema().WithFormat("binary")
		file.Description = description
		schema := openapi3.NewObjectSchema().WithProperty("file", file)
		schema.Required = []string{"file"}
		r.Operation.RequestBody = &openapi3.RequestBodyRef{Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.NewContentWithFormDataSchema(schema))}
	}
}

// schemaRef returns the schema of a Go value, referencing the component schemas of named types
func schemaRef(r *fuego.BaseRoute, v any) *openapi3.SchemaRef {
	// Fuego registers a struct as a component schema, also as the items of a slice
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		tag := fuego.SchemaTagFromType(r.OpenAPI, v)
		return &tag.SchemaRef
	}

	// Fuego would turn other types, such as a map or a string, into a component schema of their value type, so openapi3gen describes them inline
	ref, err := r.OpenAPI.Generator().NewSchemaRefForValue(v, r.OpenAPI.Description().Components.Schemas)
	if err != nil {
		panic(fmt.Sprintf("failed to generate the schema of %T: %v", v, err))
	}
	return ref
}

func orStatusText(description string, status int) string {
	if description == "" {
		return http.StatusText(status)
	}
	return description
}
