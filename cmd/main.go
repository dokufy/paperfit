package main

import (
	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/server"
)

func main() {
	config.Load()

	server.New(config.Config.ServerRESTPort)
}
