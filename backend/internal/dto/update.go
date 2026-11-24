package dto

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

// Update computes the DTO an update saves from the current state of the resource
// Services call it inside their transaction once they have locked the resource, so a PATCH never merges onto a state that a concurrent update is about to overwrite
type Update[T any] func(current any) (T, error)

// Replace is an update that saves input as it is, like a PUT or an internal caller that already holds the full DTO
func Replace[T any](input T) Update[T] {
	return func(any) (T, error) {
		return input, nil
	}
}

// MergePatch is an update that applies a JSON merge patch (RFC 7396) to the current state mapped onto T, then normalizes and validates the result
// The patch is the decoded request body, so objects merge recursively, arrays replace, null removes a member and members match case-insensitively
func MergePatch[T any](patch any) Update[T] {
	return func(current any) (T, error) {
		var value T

		// Serialize the current state in the shape of the update DTO
		err := MapStruct(current, &value)
		if err != nil {
			return value, err
		}
		currentJSON, err := json.Marshal(value)
		if err != nil {
			return value, err
		}
		var target any
		err = DecodeJSON(bytes.NewReader(currentJSON), &target)
		if err != nil {
			return value, err
		}

		// Decode the merged document into an empty DTO, because decoding onto the current one would let replaced array elements keep the fields their replacements leave out
		merged, err := json.Marshal(mergePatch(target, patch))
		if err != nil {
			return value, err
		}
		var result T
		err = json.Unmarshal(merged, &result)
		if err != nil {
			return result, apperror.InvalidRequestBody(err)
		}

		// Validate the merged result, since the patch alone may leave a required field empty
		return result, ValidateUpdate(&value, &result)
	}
}

// ValidateUpdate normalizes and validates the updated DTO against the current one, and reports only the violations the update introduces
// A value that broke a rule when it was stored would otherwise keep every other field of the resource from being changed
// The whole DTO is still validated, because some rules read more than one field
func ValidateUpdate(current, updated any) error {
	Normalize(updated)
	err := binding.Validator.ValidateStruct(updated)
	violations, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}

	stored, _ := errors.AsType[validator.ValidationErrors](binding.Validator.ValidateStruct(current))
	violations = slices.DeleteFunc(violations, func(violation validator.FieldError) bool {
		return slices.ContainsFunc(stored, func(known validator.FieldError) bool {
			return known.Namespace() == violation.Namespace() && known.Tag() == violation.Tag() && reflect.DeepEqual(known.Value(), violation.Value())
		})
	})
	if len(violations) == 0 {
		return nil
	}
	return violations
}

// DecodeJSON keeps numbers as json.Number, so large integers survive the round trip through a merge
func DecodeJSON(r io.Reader, value any) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(value)
}

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
