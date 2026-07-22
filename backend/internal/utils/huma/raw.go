package humautils

import (
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// AddRawOperation documents a Gin endpoint that must retain direct response control
func AddRawOperation(api huma.API, operation huma.Operation, statuses ...int) {
	if len(statuses) == 0 {
		statuses = []int{http.StatusOK}
	}
	responses := make(map[string]*huma.Response, len(statuses))
	for _, status := range statuses {
		responses[strconv.Itoa(status)] = &huma.Response{Description: http.StatusText(status)}
	}
	operation.Responses = responses
	api.OpenAPI().AddOperation(&operation)
}
