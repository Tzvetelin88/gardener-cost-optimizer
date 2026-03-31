package recommender

import (
	"context"
	"fmt"
	"sort"
	"time"

	"smart-cost-optimizer/backend/internal/metrics"
	"smart-cost-optimizer/backend/internal/models"
)


type EngineConfig struct {
	IdleThresholdPercent     float64
	TargetUtilizationPercent float64
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		IdleThresholdPercent:     75,
		TargetUtilizationPercent: 70,
	}
}

type Engine struct {
	metrics *metrics.Provider
	cfg     EngineConfig
}

func NewEngine(metricsProvider *metrics.Provider) *Engine {
	return NewEngineWithConfig(metricsProvider, DefaultEngineConfig())
}

func NewEngineWithConfig(metricsProvider *metrics.Provider, cfg EngineConfig) *Engine {
	return &Engine{metrics: metricsProvider, cfg: cfg}
}


func (e *Engine) BuildSnapshot(ctx context.Context, clusters []models.ClusterSummary) (models.InventorySnapshot, map[string]models.ClusterMetrics, error) {
	metricsMap := make(map[string]models.ClusterMetrics, len(clusters))
	enriched := make([]models.ClusterSummary, 0, len(clusters))

	var totalSpend float64
	for _, cluster := range clusters {
		clusterMetrics, err := e.metrics.ClusterMetrics(ctx, cluster)
		if err != nil {
			return models.InventorySnapshot{}, nil, fmt.Errorf("collect metrics for %s: %w", cluster.Name, err)
		}

		cluster.WorkloadCount = len(clusterMetrics.Workloads)
		cluster.UtilizationScore = int((clusterMetrics.CPUUtilizationPercent + clusterMetrics.MemoryUtilizationPercent) / 2)
		metricsMap[cluster.Name] = clusterMetrics
		enriched = append(enriched, cluster)
		totalSpend += cluster.MonthlyCost
	}

	recommendations := e.recommend(enriched, metricsMap)

	var actionable int
	var totalSavings float64
	for _, recommendation := range recommendations {
		totalSavings += recommendation.MonthlySavings
		if recommendation.Executable {
			actionable++
		}
	}

	return models.InventorySnapshot{
		Clusters:        enriched,
		Recommendations: recommendations,
		Summary: models.SavingsSummary{
			TotalMonthlySpend:   totalSpend,
			TotalMonthlySavings: totalSavings,
			ActionableCount:     actionable,
			AdvisoryCount:       len(recommendations) - actionable,
		},
	}, metricsMap, nil
}

func (e *Engine) recommend(clusters []models.ClusterSummary, metricsMap map[string]models.ClusterMetrics) []models.Recommendation {
	recommendations := make([]models.Recommendation, 0)
	now := time.Now().UTC()

	for _, cluster := range clusters {
		clusterMetrics := metricsMap[cluster.Name]
		if !cluster.Hibernated && clusterMetrics.IdleScore >= e.cfg.IdleThresholdPercent && cluster.WorkloadCount == 0 {
			recommendations = append(recommendations, models.Recommendation{
				ID:             "hibernate-" + cluster.Name,
				Kind:           "idle-cluster",
				Subject:        cluster.Name,
				Reason:         "Cluster has no discovered workloads and has stayed mostly idle.",
				Evidence:       []string{"Idle score above 75", "No active deployments discovered", "Suitable for non-prod hibernation"},
				MonthlySavings: cluster.MonthlyCost * 0.65,
				Risk:           "low",
				Executable:     true,
				TargetCluster:  cluster.Project + "/" + cluster.Name,
				ActionType:     "hibernate-cluster",
				CreatedAt:      now,
			})
		}

		for _, workload := range clusterMetrics.Workloads {
			if workload.Stateful || cluster.Hibernated {
				continue
			}

		target := cheapestTarget(clusters, cluster, e.cfg.TargetUtilizationPercent)
		if target == nil {
				continue
			}

			recommendations = append(recommendations, models.Recommendation{
				ID:             "move-" + cluster.Name + "-" + workload.Namespace + "-" + workload.Name,
				Kind:           "cheaper-placement",
				Subject:        workload.Namespace + "/" + workload.Name,
				Reason:         "Stateless workload is running on a more expensive cluster than a comparable lower-cost target.",
				Evidence:       []string{"Workload is stateless", "Target cluster is cheaper", "Target cluster utilization remains below 70%"},
				MonthlySavings: workload.MonthlyCost * 0.2,
				Risk:           "medium",
				Executable:     true,
				SourceCluster:  cluster.Project + "/" + cluster.Name,
				TargetCluster:  target.Project + "/" + target.Name,
				TargetWorkload: workload.Namespace + "/" + workload.Name,
				ActionType:     "move-workload",
				CreatedAt:      now,
			})
			break
		}
	}

	for _, recommendation := range e.consolidationRecommendations(clusters, metricsMap, now) {
		recommendations = append(recommendations, recommendation)
	}

	sort.Slice(recommendations, func(first int, second int) bool {
		return recommendations[first].MonthlySavings > recommendations[second].MonthlySavings
	})

	return recommendations
}

func cheapestTarget(clusters []models.ClusterSummary, source models.ClusterSummary, maxUtilization float64) *models.ClusterSummary {
	var best *models.ClusterSummary
	for idx := range clusters {
		cluster := &clusters[idx]
		if cluster.Name == source.Name || cluster.Hibernated || cluster.Region != source.Region {
			continue
		}
		if cluster.MonthlyCost >= source.MonthlyCost || float64(cluster.UtilizationScore) > maxUtilization {
			continue
		}
		if cluster.Purpose != source.Purpose {
			continue
		}

		if best == nil || cluster.MonthlyCost < best.MonthlyCost {
			best = cluster
		}
	}

	return best
}

func (e *Engine) consolidationRecommendations(clusters []models.ClusterSummary, metricsMap map[string]models.ClusterMetrics, now time.Time) []models.Recommendation {
	recommendations := make([]models.Recommendation, 0)
	for idx := 0; idx < len(clusters); idx++ {
		first := clusters[idx]
		if first.Hibernated || first.WorkloadCount == 0 {
			continue
		}

		for otherIdx := idx + 1; otherIdx < len(clusters); otherIdx++ {
			second := clusters[otherIdx]
			if second.Hibernated || second.WorkloadCount == 0 {
				continue
			}
			if first.Cloud != second.Cloud || first.Region != second.Region || first.Purpose != second.Purpose {
				continue
			}
			if metricsMap[first.Name].IdleScore < 50 || metricsMap[second.Name].IdleScore < 50 {
				continue
			}

			recommendations = append(recommendations, models.Recommendation{
				ID:             "consolidate-" + first.Name + "-" + second.Name,
				Kind:           "cluster-consolidation",
				Subject:        first.Name + " + " + second.Name,
				Reason:         "Both clusters are underutilized and serve similar purpose, region, and cloud boundaries.",
				Evidence:       []string{"Same cloud and region", "Similar purpose and isolation model", "Combined utilization remains manageable"},
				MonthlySavings: minFloat(first.MonthlyCost, second.MonthlyCost) * 0.5,
				Risk:           "high",
				Executable:     false,
				TargetCluster:  first.Project + "/" + first.Name,
				ActionType:     "consolidate-cluster",
				CreatedAt:      now,
			})
			break
		}
	}

	return recommendations
}

func minFloat(first float64, second float64) float64 {
	if first < second {
		return first
	}

	return second
}
