package dto

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"golang.org/x/text/unicode/norm"
)

// Normalize iterates through an object and performs Unicode normalization on all string fields with the `unorm` tag.
func Normalize(obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()

	// Handle case where obj is a slice of models
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
				Normalize(elem.Interface())
			} else if elem.Kind() == reflect.Struct && elem.CanAddr() {
				Normalize(elem.Addr().Interface())
			}
		}
		return
	}

	if v.Kind() != reflect.Struct {
		return
	}

	// Iterate through all fields looking for those with the "unorm" tag
	t := v.Type()
loop:
	for i := range t.NumField() {
		field := t.Field(i)

		unormTag := field.Tag.Get("unorm")
		if unormTag == "" {
			continue
		}

		fv := v.Field(i)
		if !fv.CanSet() || fv.Kind() != reflect.String {
			continue
		}

		var form norm.Form
		switch unormTag {
		case "nfc":
			form = norm.NFC
		case "nfkc":
			form = norm.NFKC
		case "nfd":
			form = norm.NFD
		case "nfkd":
			form = norm.NFKD
		default:
			continue loop
		}

		val := fv.String()
		val = form.String(val)
		fv.SetString(val)
	}
}

func ShouldBindWithNormalizedJSON(ctx *gin.Context, obj any) error {
	return ctx.ShouldBindWith(obj, binding.JSON)
}

type NormalizerJSONBinding struct{}

func (NormalizerJSONBinding) Name() string {
	return "json"
}

func (NormalizerJSONBinding) Bind(req *http.Request, obj any) error {
	// Use the default JSON binder
	err := binding.JSON.Bind(req, obj)
	if err != nil {
		return err
	}

	// Perform normalization
	Normalize(obj)

	return nil
}
