package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/client"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/config"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/handler"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/middleware"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("Starting tsdb-query-gateway on port %d with %d prometheus endpoints", cfg.Port, len(cfg.PromEndpoints))

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	// API Group
	api := engine.Group("/api/v1")

	// Initialize Core Clients
	// You can load timeout from config here; using 0 triggers the default 30s
	promClient := client.NewPrometheusClient(0)

	// Create the base gateway service
	baseGatewayService := service.NewGatewayService(cfg.PromEndpoints, promClient)

	// Build the middleware chain
	var middlewares []middleware.Middleware

	// Dynamically resolve middlewares from structured config map
	for _, mwConfig := range cfg.Middlewares {
		mwName := mwConfig.Name
		if mwName == "" {
			log.Fatalf("Fatal: Middleware configuration missing 'name' field.")
		}

		factory := middleware.Get(mwName)
		if factory == nil {
			log.Fatalf("Fatal: Middleware '%s' is declared in config but not registered.", mwName)
		}
		// Pass the specific config block and router to the factory
		middlewares = append(middlewares, factory(mwConfig, api))
	}

	// Chain the middlewares around the base service.
	finalService := middleware.Chain(baseGatewayService, middlewares...)

	queryHandler := handler.NewQueryHandler(finalService)

	// Register core Prometheus routes
	queryHandler.RegisterRoutes(api)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
