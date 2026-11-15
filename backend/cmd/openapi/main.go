// Command openapi prints the OpenAPI spec of the Pocket ID API, which the documentation site renders its API reference from
package main

import (
	"fmt"
	"os"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
)

func main() {
	spec, err := bootstrap.OpenAPISpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate the OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(spec)
}
