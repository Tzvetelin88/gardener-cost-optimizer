package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"smart-cost-optimizer/backend/internal/actions"
	"smart-cost-optimizer/backend/internal/config"
	"smart-cost-optimizer/backend/internal/gardener"
	httpapi "smart-cost-optimizer/backend/internal/http"
	"smart-cost-optimizer/backend/internal/metrics"
	"smart-cost-optimizer/backend/internal/pricing"
	"smart-cost-optimizer/backend/internal/recommender"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	catalog := pricing.NewCatalog()

	var reader httpapi.ClusterReader
	var mockLandscape *gardener.MockLandscape
	var gardenerClient *gardener.Client
	var err error

	switch cfg.DataSourceMode {
	case "mock":
		mockLandscape = gardener.NewMockLandscape(catalog)
		reader = mockLandscape
		log.Printf("using mock data source")
	case "real":
		gardenerClient, err = gardener.NewClient(cfg.GardenerKubeconfig, cfg.GardenerContext, catalog)
		if err != nil {
			log.Fatalf("unable to initialize Gardener client in real mode: %v", err)
		}
		reader = gardenerClient
		log.Printf("using real Gardener data source")
	default:
		gardenerClient, err = gardener.NewClient(cfg.GardenerKubeconfig, cfg.GardenerContext, catalog)
		if err != nil {
			if !cfg.EnableFallbackData {
				log.Fatalf("unable to initialize Gardener client: %v", err)
			}
			log.Printf("real Gardener unavailable, switching to mock data source: %v", err)
			mockLandscape = gardener.NewMockLandscape(catalog)
			reader = mockLandscape
		} else {
			reader = gardenerClient
			log.Printf("using real Gardener data source")
		}
	}

	metricsProvider, err := metrics.NewProvider(cfg.ShootKubeconfigMap, catalog, mockLandscape)
	if err != nil {
		log.Fatalf("unable to initialize metrics provider: %v", err)
	}

	actionService, err := actions.NewServiceWithLogPath(gardenerClient, cfg.ShootKubeconfigMap, mockLandscape, cfg.ActionLogPath)
	if err != nil {
		log.Fatalf("unable to initialize action service: %v", err)
	}

	engineCfg := recommender.EngineConfig{
		IdleThresholdPercent:     cfg.IdleThresholdPercent,
		TargetUtilizationPercent: cfg.TargetUtilizationPercent,
	}
	server := httpapi.NewServer(reader, recommender.NewEngineWithConfig(metricsProvider, engineCfg), actionService, cfg.RefreshInterval, cfg.FrontendOrigin)
	server.Start(ctx)

	httpServer := &http.Server{
		Addr:    cfg.APIAddress,
		Handler: server.Handler(),
	}

	log.Printf("smart cost optimizer API listening on %s", cfg.APIAddress)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server exited: %v", err)
	}
}
