package main

import (
	"log"
	"net/http"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/companion"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

func main() {
	// get the all configs based on the flags
	config := bootstrap.NewConfig(os.Args[1:])

	// setup the required files for the mods
	factorio.ModStartUp()

	// install and enable the embedded companion mod
	if err := companion.EnsureInstalled(); err != nil {
		log.Printf("Error installing the companion mod: %v\n", err)
	}

	// Initialize Factorio Server struct
	err := factorio.NewFactorioServer()
	if err != nil {
		log.Printf("Error occurred during Server initialization: %v\n", err)
		return
	}

	// Initialize authentication system
	api.SetupAuth()

	// Initialize HTTP router -- also initializes websocket
	router := api.NewRouter()

	log.Printf("Starting server on: %s:%s", config.ServerIP, config.ServerPort)
	log.Fatal(http.ListenAndServe(config.ServerIP+":"+config.ServerPort, router))

}
