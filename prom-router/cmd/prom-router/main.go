package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/client"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
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

	promClient := client.NewPrometheusClient()

	// Create the base router service
	baseRouterService := service.NewRouterService(cfg.PromEndpoints, promClient)

	// Build the middleware chain
	var middlewares []middleware.Middleware

	// Dynamically resolve middlewares from config
	for _, mwName := range cfg.Middlewares {
		factory := middleware.Get(mwName)
		if factory == nil {
			log.Printf("Warning: Middleware '%s' is declared in config but not registered.", mwName)
			continue
		}
		// Pass config and the router group to allow middlewares to register their own management routes if needed
		middlewares = append(middlewares, factory(cfg, api))
	}

	// Chain the middlewares around the base service.
	// If no middlewares are configured, Chain will safely return baseRouterService directly (bypassing middlewares).
	// The request will flow: Handler -> middlewares[0] -> middlewares[1] -> ... -> baseRouterService
	finalService := middleware.Chain(baseRouterService, middlewares...)

	queryHandler := handler.NewQueryHandler(finalService)

	// Register core Prometheus routes
	queryHandler.RegisterRoutes(api)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
