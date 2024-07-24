package main

import (
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
)

func main() {
	config.Load()

	server.New(config.Config.ServerRESTPort)
}
