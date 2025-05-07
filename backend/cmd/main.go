package main

import (
	"log"

	_ "time/tzdata"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
)

// @title Pocket ID API
// @version 1.0
// @description.markdown

func main() {
	err := bootstrap.Bootstrap()
	if err != nil {
		log.Fatal(err.Error())
	}
}
