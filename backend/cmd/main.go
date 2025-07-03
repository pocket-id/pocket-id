package main

import (
	_ "time/tzdata"

	"github.com/pocket-id/pocket-id/backend/internal/cmds"
)

// @title Pocket ID API
// @version 1.0
// @description.markdown

func main() {
	cmds.Execute()
}
