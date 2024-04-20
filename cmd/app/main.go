package main

import (
	"runtime"

	"github.com/zihaolam/golang-media-upload-server/internal/api"
)

func main() {
	runtime.GOMAXPROCS(10)
	api := api.NewApi()
	api.Setup()
}
