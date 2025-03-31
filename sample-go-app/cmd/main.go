package main

import (
	"log"

	"github.com/maansaake/github-actions-base/sample-go-app/internal/somepkg"
)

//nolint:gochecknoglobals // .
var Version = "unset"

func main() {
	log.Println("Hello, World!", "Version="+Version)
	log.Println("Calling util: ", somepkg.Util())
}
