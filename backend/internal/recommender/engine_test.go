package recommender

import (
	"context"
	"testing"

	"smart-cost-optimizer/backend/internal/gardener"
	"smart-cost-optimizer/backend/internal/metrics"
	"smart-cost-optimizer/backend/internal/pricing"
)

func TestBuildSnapshotIncludesCoreRecommendations(t *testing.T) {
	catalog := pricing.NewCatalog()
	mockLandscape := gardener.NewMockLandscape(catalog)
	metricsProvider, err := metrics.NewProvider(map[string]string{}, catalog, mockLandscape)
	if err != nil {
		t.Fatalf("create metrics provider: %v", err)
	}

	engine := NewEngine(metricsProvider)
	clusters, err := mockLandscape.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("list mock clusters: %v", err)
	}

	snapshot, _, err := engine.BuildSnapshot(context.Background(), clusters)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	var foundIdle bool
	var foundCheaperPlacement bool
	for _, recommendation := range snapshot.Recommendations {
		if recommendation.Kind == "idle-cluster" && recommendation.TargetCluster == "garden-dev/dev-aws-b" {
			foundIdle = true
		}
		if recommendation.Kind == "cheaper-placement" && recommendation.SourceCluster == "garden-dev/dev-aws-a" {
			foundCheaperPlacement = true
		}
	}

	if !foundIdle {
		t.Fatalf("expected idle-cluster recommendation for garden-dev/dev-aws-b")
	}
	if !foundCheaperPlacement {
		t.Fatalf("expected cheaper-placement recommendation for garden-dev/dev-aws-a")
	}
}
