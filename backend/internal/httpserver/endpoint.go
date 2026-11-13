package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

// Endpoint is the handler of a route together with the OpenAPI documentation that its types imply
type Endpoint struct {
	Handler gin.HandlerFunc
	Docs    []fuego.RouteOption
}

// JSON serves a handler that answers with a JSON value of type T
func JSON[T any](status int, handler func(*gin.Context) (T, error)) Endpoint {
	return Endpoint{
		Handler: handle(func(c *gin.Context) error {
			value, err := handler(c)
			if err != nil {
				return err
			}
			c.JSON(status, value)
			return nil
		}),
		Docs: []fuego.RouteOption{openapi.Response[T](status, "")},
	}
}

// BodyJSON serves a handler that takes a JSON body of type B and answers with a JSON value of type T
func BodyJSON[B, T any](status int, handler func(*gin.Context, B) (T, error)) Endpoint {
	endpoint := JSON(status, func(c *gin.Context) (T, error) {
		var input B
		if err := BindJSON(c, &input); err != nil {
			return *new(T), err
		}
		return handler(c, input)
	})
	endpoint.Docs = append(endpoint.Docs, openapi.Body[B]())
	return endpoint
}

// UpdateJSON serves a PUT or PATCH handler that updates a resource with a body of type B and answers with the updated resource of type T
func UpdateJSON[B, T any](handler func(*gin.Context, dto.Update[B]) (T, error)) Endpoint {
	endpoint := JSON(http.StatusOK, func(c *gin.Context) (T, error) {
		update, err := BindUpdate[B](c)
		if err != nil {
			return *new(T), err
		}
		return handler(c, update)
	})
	endpoint.Docs = append(endpoint.Docs, openapi.UpdateBody[B]())
	return endpoint
}

// NoContent serves a handler that answers with 204 No Content
func NoContent(handler HandlerFunc) Endpoint {
	return Endpoint{
		Handler: handle(func(c *gin.Context) error {
			if err := handler(c); err != nil {
				return err
			}
			c.Status(http.StatusNoContent)
			return nil
		}),
		Docs: []fuego.RouteOption{openapi.Status(http.StatusNoContent, "")},
	}
}

// BodyNoContent serves a handler that takes a JSON body of type B and answers with 204 No Content
func BodyNoContent[B any](handler func(*gin.Context, B) error) Endpoint {
	endpoint := NoContent(func(c *gin.Context) error {
		var input B
		if err := BindJSON(c, &input); err != nil {
			return err
		}
		return handler(c, input)
	})
	endpoint.Docs = append(endpoint.Docs, openapi.Body[B]())
	return endpoint
}
