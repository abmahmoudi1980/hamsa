// Hamsa — Building Management MVP (P0) API server.
package main

import (
	"log"

	"hamsa/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("hamsa api starting (env=%s)", cfg.App.Env)
	// HTTP server, database wiring, and migrations are added in Phase 2 (T006/T007).
	_ = cfg
}
