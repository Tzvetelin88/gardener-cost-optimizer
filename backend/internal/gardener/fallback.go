package gardener

import (
	"context"
	"fmt"
	"sync"

	"smart-cost-optimizer/backend/internal/models"
	"smart-cost-optimizer/backend/internal/pricing"
)

type MockLandscape struct {
	mu        sync.RWMutex
	catalog   *pricing.Catalog
	clusters  []models.ClusterSummary
	workloads map[string][]models.WorkloadSummary
}

func NewMockLandscape(catalog *pricing.Catalog) *MockLandscape {
	landscape := &MockLandscape{
		catalog: catalog,
		clusters: []models.ClusterSummary{
			{
				ID:         "garden-dev/dev-aws-a",
				Name:       "dev-aws-a",
				Project:    "garden-dev",
				Cloud:      "aws",
				Region:     "eu-central-1",
				Seed:       "seed-aws-eu1",
				Purpose:    "dev",
				Hibernated: false,
				WorkerPools: []models.WorkerPool{
					{Name: "system", MachineType: "m6i.large", Minimum: 3, Maximum: 5},
				},
			},
			{
				ID:         "garden-dev/dev-aws-b",
				Name:       "dev-aws-b",
				Project:    "garden-dev",
				Cloud:      "aws",
				Region:     "eu-central-1",
				Seed:       "seed-aws-eu1",
				Purpose:    "dev",
				Hibernated: false,
				WorkerPools: []models.WorkerPool{
					{Name: "system", MachineType: "t3.large", Minimum: 2, Maximum: 4},
				},
			},
			{
				ID:         "garden-prod/prod-azure-a",
				Name:       "prod-azure-a",
				Project:    "garden-prod",
				Cloud:      "azure",
				Region:     "westeurope",
				Seed:       "seed-azure-weu",
				Purpose:    "prod",
				Hibernated: false,
				WorkerPools: []models.WorkerPool{
					{Name: "system", MachineType: "standard_d4s", Minimum: 4, Maximum: 8},
				},
			},
		},
		workloads: map[string][]models.WorkloadSummary{
			"dev-aws-a": {
				{
					Cluster:    "dev-aws-a",
					Namespace:  "web",
					Name:       "catalog-api",
					Kind:       "Deployment",
					Replicas:   2,
					CPURequest: 1.2,
					MemoryGiB:  2.5,
					Stateful:   false,
				},
				{
					Cluster:    "dev-aws-a",
					Namespace:  "batch",
					Name:       "cleanup-job",
					Kind:       "Deployment",
					Replicas:   1,
					CPURequest: 0.4,
					MemoryGiB:  0.5,
					Stateful:   false,
				},
			},
			"dev-aws-b": {},
			"prod-azure-a": {
				{
					Cluster:    "prod-azure-a",
					Namespace:  "checkout",
					Name:       "payments-api",
					Kind:       "Deployment",
					Replicas:   3,
					CPURequest: 3,
					MemoryGiB:  6,
					Stateful:   false,
				},
				{
					Cluster:    "prod-azure-a",
					Namespace:  "checkout",
					Name:       "orders-db",
					Kind:       "StatefulSet",
					Replicas:   1,
					CPURequest: 1,
					MemoryGiB:  4,
					Stateful:   true,
				},
			},
		},
	}

	landscape.refreshDerivedStateLocked()
	return landscape
}

func NewFallbackReader(catalog *pricing.Catalog) *MockLandscape {
	return NewMockLandscape(catalog)
}

func (m *MockLandscape) ListClusters(_ context.Context) ([]models.ClusterSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]models.ClusterSummary, len(m.clusters))
	for idx, cluster := range m.clusters {
		clusters[idx] = cloneCluster(cluster)
	}

	return clusters, nil
}

func (m *MockLandscape) ClusterMetrics(_ context.Context, cluster models.ClusterSummary) (models.ClusterMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workloads := cloneWorkloads(m.workloads[cluster.Name])
	var cpuRequested float64
	var memoryRequested float64
	for idx := range workloads {
		workloads[idx].MonthlyCost = m.catalog.EstimateWorkloadMonthlyCost(workloads[idx])
		cpuRequested += workloads[idx].CPURequest
		memoryRequested += workloads[idx].MemoryGiB
	}

	nodeCount := 0
	for _, pool := range cluster.WorkerPools {
		nodeCount += int(pool.Minimum)
	}
	if nodeCount == 0 {
		nodeCount = 1
	}

	cpuUtilization := minFloat(100, (cpuRequested/float64(nodeCount*4))*100)
	memoryUtilization := minFloat(100, (memoryRequested/float64(nodeCount*16))*100)
	idleScore := 100 - ((cpuUtilization + memoryUtilization) / 2)

	return models.ClusterMetrics{
		CPUUtilizationPercent:    cpuUtilization,
		MemoryUtilizationPercent: memoryUtilization,
		NodeCount:                nodeCount,
		IdleScore:                idleScore,
		Workloads:                workloads,
	}, nil
}

func (m *MockLandscape) SetHibernation(_ context.Context, namespace string, name string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for idx := range m.clusters {
		if m.clusters[idx].Project == namespace && m.clusters[idx].Name == name {
			m.clusters[idx].Hibernated = enabled
			m.refreshDerivedStateLocked()
			return nil
		}
	}

	return fmt.Errorf("mock cluster %s/%s not found", namespace, name)
}

func (m *MockLandscape) ScaleWorkerPool(_ context.Context, namespace string, name string, poolName string, minimum int64, maximum int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for clusterIdx := range m.clusters {
		cluster := &m.clusters[clusterIdx]
		if cluster.Project != namespace || cluster.Name != name {
			continue
		}

		for poolIdx := range cluster.WorkerPools {
			pool := &cluster.WorkerPools[poolIdx]
			if pool.Name == poolName {
				pool.Minimum = minimum
				pool.Maximum = maximum
				m.refreshDerivedStateLocked()
				return nil
			}
		}
	}

	return fmt.Errorf("mock worker pool %s not found for %s/%s", poolName, namespace, name)
}

func (m *MockLandscape) MoveWorkload(_ context.Context, sourceCluster string, targetCluster string, namespace string, workloadName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sourceName := clusterName(sourceCluster)
	targetName := clusterName(targetCluster)

	sourceWorkloads := m.workloads[sourceName]
	targetWorkloads := m.workloads[targetName]

	for idx := range sourceWorkloads {
		workload := sourceWorkloads[idx]
		if workload.Namespace != namespace || workload.Name != workloadName {
			continue
		}
		if workload.Stateful {
			return fmt.Errorf("mock workload %s/%s is stateful and cannot be moved automatically", namespace, workloadName)
		}

		sourceWorkloads = append(sourceWorkloads[:idx], sourceWorkloads[idx+1:]...)
		workload.Cluster = targetName
		targetWorkloads = append(targetWorkloads, workload)
		m.workloads[sourceName] = sourceWorkloads
		m.workloads[targetName] = targetWorkloads
		m.refreshDerivedStateLocked()
		return nil
	}

	return fmt.Errorf("mock workload %s/%s not found in %s", namespace, workloadName, sourceCluster)
}

func (m *MockLandscape) refreshDerivedStateLocked() {
	for idx := range m.clusters {
		cluster := &m.clusters[idx]
		cluster.WorkloadCount = len(m.workloads[cluster.Name])
		cluster.MonthlyCost = m.catalog.EstimateClusterMonthlyCost(cluster)
	}
}

func cloneCluster(cluster models.ClusterSummary) models.ClusterSummary {
	cloned := cluster
	cloned.WorkerPools = append([]models.WorkerPool(nil), cluster.WorkerPools...)
	if cluster.Labels != nil {
		cloned.Labels = map[string]string{}
		for key, value := range cluster.Labels {
			cloned.Labels[key] = value
		}
	}
	return cloned
}

func cloneWorkloads(workloads []models.WorkloadSummary) []models.WorkloadSummary {
	cloned := make([]models.WorkloadSummary, len(workloads))
	copy(cloned, workloads)
	return cloned
}

func clusterName(clusterID string) string {
	for idx := len(clusterID) - 1; idx >= 0; idx-- {
		if clusterID[idx] == '/' {
			return clusterID[idx+1:]
		}
	}
	return clusterID
}

func minFloat(first float64, second float64) float64 {
	if first < second {
		return first
	}
	return second
}
