package pricing

import (
	"strings"

	"smart-cost-optimizer/backend/internal/models"
)

type Catalog struct {
	prices map[string]float64
}

func NewCatalog() *Catalog {
	return &Catalog{
		prices: map[string]float64{
			"aws:m6i.large":       0.096,
			"aws:m6i.xlarge":      0.192,
			"aws:t3.large":        0.0832,
			"azure:standard_d4s":  0.192,
			"azure:standard_d8s":  0.384,
			"azure:standard_b4ms": 0.166,
			"gcp:e2-standard-4":   0.134,
			"gcp:e2-standard-8":   0.268,
		},
	}
}

func (c *Catalog) WorkerPoolPrice(cloud string, machineType string) float64 {
	key := strings.ToLower(cloud) + ":" + strings.ToLower(machineType)
	if price, ok := c.prices[key]; ok {
		return price
	}

	return 0.15
}

func (c *Catalog) EstimateClusterMonthlyCost(cluster *models.ClusterSummary) float64 {
	var total float64
	for idx := range cluster.WorkerPools {
		pool := &cluster.WorkerPools[idx]
		pool.HourlyPrice = c.WorkerPoolPrice(cluster.Cloud, pool.MachineType)
		total += pool.HourlyPrice * float64(max(pool.Minimum, 1)) * 730
	}

	return total
}

func (c *Catalog) EstimateWorkloadMonthlyCost(workload models.WorkloadSummary) float64 {
	return ((workload.CPURequest * 12) + (workload.MemoryGiB * 2)) * 30
}

func max(first int64, second int64) int64 {
	if first > second {
		return first
	}

	return second
}
