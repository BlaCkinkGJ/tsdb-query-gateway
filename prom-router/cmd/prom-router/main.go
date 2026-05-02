package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/example/prom-router/internal/client"
	"github.com/example/prom-router/internal/config"
	"github.com/example/prom-router/internal/handler"
	"github.com/example/prom-router/internal/service"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("Starting prom-router on port %d with %d prometheus endpoints", cfg.Port, len(cfg.PromEndpoints))

	promClient := client.NewPrometheusClient()
	aiClient := client.NewAIClient()

	routerService := service.NewRouterService(cfg.PromEndpoints, cfg.AIEndpoint, promClient, aiClient)

	queryHandler := handler.NewQueryHandler(routerService)

	mux := http.NewServeMux()
	queryHandler.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
