package main

import (
	"github.com/emrgen/tinydoc/internal/server"
	"os"
)

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "4000"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "4001"
	}

	err := server.Start(grpcPort, httpPort)
	if err != nil {
		return
	}
}
