package main

import (
	"log"
	"os"

	"github.com/maansaake/github-actions-base/sample-go-app/internal/somepkg"
)

func main() {
	ver := os.Getenv("VERSION")

	//nolint:gosec // welp
	log.Println("Hello, World!", "Version="+ver)
	log.Println("Calling util: ", somepkg.Util())
}
