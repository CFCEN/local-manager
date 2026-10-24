package main

import (
	"log"
	"net/http"

	"local-dev-workspace/internal/api"
	"local-dev-workspace/internal/config"
	"local-dev-workspace/internal/repository"
)

func main() {
	cfg := config.Load()

	repository.InitDB(cfg.DBPath)
	defer repository.CloseDB()

	router := api.NewRouter(cfg)
	handler := api.CORSWrapper(router, cfg.AllowedOrigins)

	addr := cfg.Host + ":" + cfg.Port
	log.Printf("Local Dev Workspace starting on http://%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
