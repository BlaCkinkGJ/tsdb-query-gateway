package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/example/prom-router/internal/client"
	"github.com/example/prom-router/internal/config"
	"github.com/example/prom-router/internal/handler"
	"github.com/example/prom-router/internal/middleware"
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

	// Create the base router service
	baseRouterService := service.NewRouterService(cfg.PromEndpoints, promClient)

	// Build the middleware chain
	var middlewares []middleware.Middleware

	// Example: Add statistical middleware if enabled
	if cfg.EnableStatisticalMW {
		middlewares = append(middlewares, middleware.NewStatisticalMiddleware())
	}

	// Example: Add AI middleware if enabled
	if cfg.EnableAIMW {
		aiClient := client.NewAIClient()
		middlewares = append(middlewares, middleware.NewAIMiddleware(cfg.AIEndpoint, aiClient))
	}

	// Chain the middlewares around the base service.
	// The request will flow: Handler -> middlewares[0] -> middlewares[1] -> ... -> baseRouterService
	finalService := middleware.Chain(baseRouterService, middlewares...)

	queryHandler := handler.NewQueryHandler(finalService)

	mux := http.NewServeMux()
	queryHandler.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
