package main

import (
	"flag"
	"fmt"
	"log"

	_ "time/tzdata"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/cmds"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// @title Pocket ID API
// @version 1.0
// @description.markdown

func main() {
	// Get the command
	// By default, this starts the server
	var cmd string
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		cmd = args[0]
	}

	var err error
	switch cmd {
	case "version":
		fmt.Println("pocket-id " + common.Version)
	case "one-time-access-token":
		err = cmds.OneTimeAccessToken(args)
	default:
		// Start the server
		err = bootstrap.Bootstrap()
	}

	if err != nil {
		log.Fatal(err.Error())
	}
}
