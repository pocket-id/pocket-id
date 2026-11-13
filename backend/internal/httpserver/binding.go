package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

// BindJSON binds and normalizes a JSON request while distinguishing invalid input from internal failures
func BindJSON(c *gin.Context, value any) error {
	if err := requireJSONContentType(c); err != nil {
		return err
	}

	err := classifyBindingError(c.ShouldBindJSON(value))
	if err != nil {
		return err
	}

	dto.Normalize(value)
	return nil
}

// acceptedPatchTypes lists the PATCH formats BindUpdateJSON applies, where plain JSON is read as a merge patch too
const acceptedPatchTypes = "application/merge-patch+json, application/json"

// BindUpdateJSON binds the body of a PUT or PATCH update into the zero-valued update DTO
// A PATCH is a JSON merge patch (RFC 7396) on the current resource, so the fields its body leaves out keep their value and null clears a field
func BindUpdateJSON(c *gin.Context, value any, current func() (any, error)) error {
	// Advertise the PATCH formats on every update response, as RFC 5789 suggests, so clients can discover them from a PUT or a rejected PATCH
	c.Header("Accept-Patch", acceptedPatchTypes)
	if c.Request.Method != http.MethodPatch {
		return BindJSON(c, value)
	}

	// Reject other patch formats such as JSON Patch, which would otherwise be misread as a merge patch
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || (mediaType != "application/merge-patch+json" && mediaType != "application/json") {
		return apperror.UnsupportedPatchType(acceptedPatchTypes)
	}

	// Parse the patch before loading the resource, so a malformed body fails as invalid input
	var patch any
	if err := decodeJSON(c.Request.Body, &patch); err != nil {
		return apperror.InvalidRequestBody(err)
	}

	// Serialize the current resource in the shape of the update DTO
	resource, err := current()
	if err != nil {
		return err
	}
	if err := dto.MapStruct(resource, value); err != nil {
		return err
	}
	currentJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var target any
	if err := decodeJSON(bytes.NewReader(currentJSON), &target); err != nil {
		return err
	}

	// Decode the merged document into an empty DTO, because decoding onto the current one would let replaced array elements keep the fields their replacements leave out
	merged, err := json.Marshal(mergePatch(target, patch))
	if err != nil {
		return err
	}
	reflect.ValueOf(value).Elem().SetZero()
	if err := classifyBindingError(binding.JSON.BindBody(merged, value)); err != nil {
		return err
	}

	dto.Normalize(value)
	return nil
}

// mergePatch applies a JSON merge patch to a decoded JSON document: objects merge recursively, null removes a member and anything else replaces it
func mergePatch(target, patch any) any {
	patchObject, ok := patch.(map[string]any)
	if !ok {
		return patch
	}
	targetObject, ok := target.(map[string]any)
	if !ok {
		targetObject = map[string]any{}
	}

	for key, value := range patchObject {
		// Match members case-insensitively like encoding/json does, so "FirstName" replaces the current "firstName"
		for existing := range targetObject {
			if existing != key && strings.EqualFold(existing, key) {
				targetObject[key] = targetObject[existing]
				delete(targetObject, existing)
			}
		}

		if value == nil {
			delete(targetObject, key)
		} else {
			targetObject[key] = mergePatch(targetObject[key], value)
		}
	}
	return targetObject
}

// decodeJSON keeps numbers as json.Number, so large integers survive the round trip through the merge
func decodeJSON(r io.Reader, value any) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(value)
}

// Validate validates and normalizes input that was not bound from a request, such as MCP tool arguments, with the same rules as BindJSON
func Validate(value any) error {
	if err := binding.Validator.ValidateStruct(value); err != nil {
		return err
	}

	dto.Normalize(value)
	return nil
}

// BindOptionalJSON accepts an empty body while normalizing valid input and classifying malformed JSON as invalid input
func BindOptionalJSON(c *gin.Context, value any) error {
	err := c.ShouldBindJSON(value)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if contentTypeErr := requireJSONContentType(c); contentTypeErr != nil {
		return contentTypeErr
	}
	if err = classifyBindingError(err); err != nil {
		return err
	}

	dto.Normalize(value)
	return nil
}

// FormFile returns an uploaded file while classifying a missing field as request validation
func FormFile(c *gin.Context, field string) (*multipart.FileHeader, error) {
	file, err := c.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, apperror.MissingField(field)
	}
	if err != nil {
		return nil, apperror.InvalidRequestBody(err)
	}

	return file, nil
}

func requireJSONContentType(c *gin.Context) error {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil {
		return apperror.InvalidRequestBody(err)
	}
	if !isJSONMediaType(mediaType) {
		return apperror.InvalidRequestBody(errors.New("request Content-Type is not JSON"))
	}

	return nil
}

func isJSONMediaType(mediaType string) bool {
	topLevelType, subtype, ok := strings.Cut(mediaType, "/")
	if !ok || topLevelType != "application" {
		return false
	}
	if subtype == "json" {
		return true
	}

	baseSubtype, ok := strings.CutSuffix(subtype, "+json")
	return ok && baseSubtype != ""
}

func classifyBindingError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return err
	}

	if _, ok := errors.AsType[binding.SliceValidationError](err); ok {
		return err
	}

	return apperror.InvalidRequestBody(err)
}
