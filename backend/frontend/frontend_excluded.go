//go:build !include_frontend

package frontend

import "github.com/gin-gonic/gin"

func RegisterFrontend(router *gin.Engine) error {
	return ErrFrontendNotIncluded
}
