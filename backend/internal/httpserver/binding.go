package httpserver

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
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

// acceptedPatchTypes lists the PATCH formats BindUpdate accepts, where plain JSON is read as a merge patch too
const acceptedPatchTypes = "application/merge-patch+json, application/json"

// BindUpdate binds the body of a PUT or PATCH update into the update the service applies once it has locked the resource
// A PUT replaces the resource with the validated body, and a PATCH is a JSON merge patch (RFC 7396) that the service merges onto the locked state
func BindUpdate[T any](c *gin.Context) (dto.Update[T], error) {
	// Advertise the PATCH formats on every update response, as RFC 5789 suggests, so clients can discover them from a PUT or a rejected PATCH
	c.Header("Accept-Patch", acceptedPatchTypes)

	if c.Request.Method != http.MethodPatch {
		var input T
		if err := BindJSON(c, &input); err != nil {
			return nil, err
		}
		return dto.Replace(input), nil
	}

	// Reject other patch formats such as JSON Patch, which would otherwise be misread as a merge patch
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || (mediaType != "application/merge-patch+json" && mediaType != "application/json") {
		return nil, apperror.UnsupportedPatchType(acceptedPatchTypes)
	}

	// Parse the patch before the service loads the resource, so a malformed body fails as invalid input
	var patch any
	if err := dto.DecodeJSON(c.Request.Body, &patch); err != nil {
		return nil, apperror.InvalidRequestBody(err)
	}
	return dto.MergePatch[T](patch), nil
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
