package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"smart-cost-optimizer/backend/internal/config"
	"smart-cost-optimizer/backend/internal/gardener"
	"smart-cost-optimizer/backend/internal/metrics"
	"smart-cost-optimizer/backend/internal/models"
	"smart-cost-optimizer/backend/internal/pricing"
	"smart-cost-optimizer/backend/internal/recommender"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	catalog := pricing.NewCatalog()
	switch cfg.DataSourceMode {
	case "mock":
		log.Printf("operator running in mock mode")
		runLoop(ctx, gardener.NewMockLandscape(catalog), catalog, cfg.RefreshInterval)
	case "real":
		reader, err := gardener.NewClient(cfg.GardenerKubeconfig, cfg.GardenerContext, catalog)
		if err != nil {
			log.Fatalf("unable to initialize Gardener client in real mode: %v", err)
		}
		runLoopWithShootConfigs(ctx, reader, catalog, cfg.RefreshInterval, cfg.ShootKubeconfigMap)
	default:
		reader, err := gardener.NewClient(cfg.GardenerKubeconfig, cfg.GardenerContext, catalog)
		if err != nil {
			if !cfg.EnableFallbackData {
				log.Fatalf("unable to initialize Gardener client: %v", err)
			}
			log.Printf("real Gardener unavailable, switching to mock mode: %v", err)
			runLoop(ctx, gardener.NewMockLandscape(catalog), catalog, cfg.RefreshInterval)
			return
		}

		runLoopWithShootConfigs(ctx, reader, catalog, cfg.RefreshInterval, cfg.ShootKubeconfigMap)
	}
}

type clusterReader interface {
	ListClusters(context.Context) ([]models.ClusterSummary, error)
}

func runLoop(ctx context.Context, reader clusterReader, catalog *pricing.Catalog, refresh time.Duration) {
	runLoopWithShootConfigs(ctx, reader, catalog, refresh, map[string]string{})
}

func runLoopWithShootConfigs(ctx context.Context, reader clusterReader, catalog *pricing.Catalog, refresh time.Duration, shootKubeconfigs map[string]string) {
	var mockLandscape *gardener.MockLandscape
	if typed, ok := reader.(*gardener.MockLandscape); ok {
		mockLandscape = typed
	}

	metricsProvider, err := metrics.NewProvider(shootKubeconfigs, catalog, mockLandscape)
	if err != nil {
		log.Fatalf("unable to initialize metrics provider: %v", err)
	}

	engine := recommender.NewEngine(metricsProvider)
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	for {
		clusters, err := reader.ListClusters(ctx)
		if err != nil {
			log.Printf("inventory refresh failed: %v", err)
		} else if snapshot, _, err := engine.BuildSnapshot(ctx, clusters); err != nil {
			log.Printf("recommendation refresh failed: %v", err)
		} else {
			log.Printf("refreshed %d clusters and produced %d recommendations", len(snapshot.Clusters), len(snapshot.Recommendations))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
