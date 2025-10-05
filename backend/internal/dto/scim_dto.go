package dto

// SCIM 2.0 Core Schema DTOs based on RFC 7643

// ScimListResponse represents a SCIM list response
type ScimListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	ItemsPerPage int         `json:"itemsPerPage"`
	Resources    interface{} `json:"Resources"`
}

// ScimError represents a SCIM error response
type ScimError struct {
	Schemas []string `json:"schemas"`
	Status  string   `json:"status"`
	Detail  string   `json:"detail,omitempty"`
}

// ScimUser represents a SCIM user resource
type ScimUser struct {
	Schemas  []string         `json:"schemas"`
	ID       string           `json:"id"`
	UserName string           `json:"userName"`
	Name     *ScimUserName    `json:"name,omitempty"`
	Emails   []ScimEmail      `json:"emails,omitempty"`
	Active   bool             `json:"active"`
	Groups   []ScimGroupRef   `json:"groups,omitempty"`
	Meta     *ScimMeta        `json:"meta,omitempty"`
	Locale   string           `json:"locale,omitempty"`
}

// ScimUserName represents a SCIM user's name
type ScimUserName struct {
	Formatted  string `json:"formatted,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
}

// ScimEmail represents a SCIM email address
type ScimEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// ScimGroup represents a SCIM group resource
type ScimGroup struct {
	Schemas     []string       `json:"schemas"`
	ID          string         `json:"id"`
	DisplayName string         `json:"displayName"`
	Members     []ScimMember   `json:"members,omitempty"`
	Meta        *ScimMeta      `json:"meta,omitempty"`
}

// ScimMember represents a member of a SCIM group
type ScimMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// ScimGroupRef represents a reference to a group
type ScimGroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// ScimMeta represents SCIM resource metadata
type ScimMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

// ScimPatchOp represents a SCIM PATCH operation
type ScimPatchOp struct {
	Schemas    []string           `json:"schemas"`
	Operations []ScimPatchOpEntry `json:"Operations"`
}

// ScimPatchOpEntry represents a single PATCH operation
type ScimPatchOpEntry struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// ScimServiceProviderConfig represents the service provider configuration
type ScimServiceProviderConfig struct {
	Schemas               []string                        `json:"schemas"`
	DocumentationURI      string                          `json:"documentationUri,omitempty"`
	Patch                 ScimSupported                   `json:"patch"`
	Bulk                  ScimBulkSupported               `json:"bulk"`
	Filter                ScimFilterSupported             `json:"filter"`
	ChangePassword        ScimSupported                   `json:"changePassword"`
	Sort                  ScimSupported                   `json:"sort"`
	Etag                  ScimSupported                   `json:"etag"`
	AuthenticationSchemes []ScimAuthenticationScheme      `json:"authenticationSchemes"`
	Meta                  *ScimMeta                       `json:"meta,omitempty"`
}

// ScimSupported represents a simple supported feature
type ScimSupported struct {
	Supported bool `json:"supported"`
}

// ScimBulkSupported represents bulk operation support
type ScimBulkSupported struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations,omitempty"`
	MaxPayloadSize int  `json:"maxPayloadSize,omitempty"`
}

// ScimFilterSupported represents filter support
type ScimFilterSupported struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults,omitempty"`
}

// ScimAuthenticationScheme represents an authentication scheme
type ScimAuthenticationScheme struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	SpecURI          string `json:"specUri,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
}

// ScimSchema represents a SCIM schema definition
type ScimSchema struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Attributes  []ScimSchemaAttribute `json:"attributes"`
}

// ScimSchemaAttribute represents a SCIM schema attribute
type ScimSchemaAttribute struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MultiValued   bool   `json:"multiValued"`
	Description   string `json:"description"`
	Required      bool   `json:"required"`
	CaseExact     bool   `json:"caseExact"`
	Mutability    string `json:"mutability"`
	Returned      string `json:"returned"`
	Uniqueness    string `json:"uniqueness"`
}

// ScimResourceType represents a SCIM resource type
type ScimResourceType struct {
	Schemas             []string `json:"schemas"`
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Endpoint            string   `json:"endpoint"`
	Description         string   `json:"description"`
	Schema              string   `json:"schema"`
	SchemaExtensions    []interface{} `json:"schemaExtensions,omitempty"`
	Meta                *ScimMeta `json:"meta,omitempty"`
}

// SCIM 2.0 Schema URNs
const (
	ScimSchemaCore                 = "urn:ietf:params:scim:schemas:core:2.0:User"
	ScimSchemaGroup                = "urn:ietf:params:scim:schemas:core:2.0:Group"
	ScimSchemaError                = "urn:ietf:params:scim:api:messages:2.0:Error"
	ScimSchemaListResponse         = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	ScimSchemaPatchOp              = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	ScimSchemaServiceProviderConfig = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	ScimSchemaResourceType         = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	ScimSchemaSchema               = "urn:ietf:params:scim:schemas:core:2.0:Schema"
)
