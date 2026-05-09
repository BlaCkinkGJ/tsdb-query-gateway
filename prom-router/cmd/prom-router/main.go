package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/client"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/discovery"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/handler"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/middleware"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("Starting prom-router on port %d with %d prometheus endpoints", cfg.Port, len(cfg.PromEndpoints))

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	// API Group
	api := engine.Group("/api/v1")

	// Initialize Service Registry
	registry := discovery.NewRegistry()

	// Register Core Clients
	promClient := client.NewPrometheusClient()
	registry.Register("PrometheusClient", promClient)

	// Create the base router service
	baseRouterService := service.NewRouterService(cfg.PromEndpoints, promClient)

	// Build the middleware chain
	var middlewares []middleware.Middleware

	// Dynamically resolve middlewares from structured config map
	for _, mwConfig := range cfg.Middlewares {
		mwName, ok := mwConfig["name"].(string)
		if !ok || mwName == "" {
			log.Fatalf("Middleware configuration missing 'name' field")
		}

		factory := middleware.Get(mwName)
		if factory == nil {
			log.Fatalf("Middleware '%s' is declared in config but not registered.", mwName)
		}
		// Pass the specific config block, router, and registry to the factory
		middlewares = append(middlewares, factory(mwConfig, api, registry))
	}

	// Chain the middlewares around the base service.
	finalService := middleware.Chain(baseRouterService, middlewares...)
	registry.Register("QueryService", finalService)

	queryHandler := handler.NewQueryHandler(finalService)

	// Register core Prometheus routes
	queryHandler.RegisterRoutes(api)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
