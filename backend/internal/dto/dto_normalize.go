package dto

import (
	"reflect"

	"golang.org/x/text/unicode/norm"
)

// Normalize iterates through an object and performs Unicode normalization on all string fields with the `unorm` tag
func Normalize(obj any) {
	normalizeValue(reflect.ValueOf(obj))
}

func normalizeValue(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	// Unwrap interfaces and pointers so nested DTOs share the same traversal
	kind := value.Kind()
	if kind == reflect.Interface || kind == reflect.Pointer {
		if !value.IsNil() {
			normalizeValue(value.Elem())
		}
		return
	}

	// Walk collections because request DTOs may contain nested slices or arrays
	if kind == reflect.Slice || kind == reflect.Array {
		for i := range value.Len() {
			normalizeValue(value.Index(i))
		}
		return
	}

	// Ignore scalar values because normalization is opt-in through struct field tags
	if kind != reflect.Struct {
		return
	}

	// Normalize tagged fields directly and recursively inspect untagged nested values
	valueType := value.Type()
	for i := range value.NumField() {
		field := value.Field(i)
		form, tagged := normalizationForm(valueType.Field(i).Tag.Get("unorm"))
		if tagged {
			normalizeString(field, form)
			continue
		}

		normalizeValue(field)
	}
}

func normalizationForm(tag string) (norm.Form, bool) {
	switch tag {
	case "nfc":
		return norm.NFC, true
	case "nfkc":
		return norm.NFKC, true
	case "nfd":
		return norm.NFD, true
	case "nfkd":
		return norm.NFKD, true
	default:
		return 0, false
	}
}

func normalizeString(value reflect.Value, form norm.Form) {
	// Dereference optional string fields while leaving nil values unchanged
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}

	// Ignore incompatible or read-only fields because reflection cannot safely update them
	if value.Kind() != reflect.String || !value.CanSet() {
		return
	}

	value.SetString(form.String(value.String()))
}
