package main

import (
	_ "time/tzdata"

	"github.com/pocket-id/pocket-id/backend/internal/cmds"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// @title Pocket ID API
// @version 1.0
// @description.markdown

func main() {
	if err := common.ValidateEnvConfig(&common.EnvConfig); err != nil {
		panic(err)
	}
	cmds.Execute()
}
