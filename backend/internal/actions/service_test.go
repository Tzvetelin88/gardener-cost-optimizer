package actions

import (
	"context"
	"testing"

	"smart-cost-optimizer/backend/internal/gardener"
	"smart-cost-optimizer/backend/internal/pricing"
)

func TestMockActionsMutateLandscape(t *testing.T) {
	catalog := pricing.NewCatalog()
	mockLandscape := gardener.NewMockLandscape(catalog)
	service, err := NewService(nil, map[string]string{}, mockLandscape)
	if err != nil {
		t.Fatalf("create action service: %v", err)
	}

	if _, err := service.HibernateCluster(context.Background(), "garden-dev/dev-aws-b"); err != nil {
		t.Fatalf("hibernate cluster: %v", err)
	}

	clusters, err := mockLandscape.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("list clusters: %v", err)
	}

	var hibernated bool
	for _, cluster := range clusters {
		if cluster.ID == "garden-dev/dev-aws-b" {
			hibernated = cluster.Hibernated
		}
	}
	if !hibernated {
		t.Fatalf("expected garden-dev/dev-aws-b to be hibernated")
	}

	if _, err := service.MoveWorkload(context.Background(), "garden-dev/dev-aws-a", "garden-dev/dev-aws-b", "web", "catalog-api"); err != nil {
		t.Fatalf("move workload: %v", err)
	}

	targetMetrics, err := mockLandscape.ClusterMetrics(context.Background(), clusters[1])
	if err != nil {
		t.Fatalf("target metrics: %v", err)
	}

	var foundCatalogAPI bool
	for _, workload := range targetMetrics.Workloads {
		if workload.Namespace == "web" && workload.Name == "catalog-api" && workload.Cluster == "dev-aws-b" {
			foundCatalogAPI = true
		}
	}
	if !foundCatalogAPI {
		t.Fatalf("expected catalog-api workload to be present on dev-aws-b")
	}
}
