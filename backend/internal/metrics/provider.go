package metrics

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"smart-cost-optimizer/backend/internal/models"
	"smart-cost-optimizer/backend/internal/pricing"
)

type Provider struct {
	clients map[string]*kubernetes.Clientset
	catalog *pricing.Catalog
	mock    mockMetricsProvider
}

type mockMetricsProvider interface {
	ClusterMetrics(context.Context, models.ClusterSummary) (models.ClusterMetrics, error)
}

func NewProvider(kubeconfigs map[string]string, catalog *pricing.Catalog, mock mockMetricsProvider) (*Provider, error) {
	clients := map[string]*kubernetes.Clientset{}
	for name, kubeconfig := range kubeconfigs {
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build shoot kubeconfig for %s: %w", name, err)
		}

		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("create shoot client for %s: %w", name, err)
		}

		clients[name] = clientset
	}

	return &Provider{
		clients: clients,
		catalog: catalog,
		mock:    mock,
	}, nil
}

func (p *Provider) ClusterMetrics(ctx context.Context, cluster models.ClusterSummary) (models.ClusterMetrics, error) {
	clientset, ok := p.clients[cluster.Name]
	if !ok {
		if p.mock != nil {
			return p.mock.ClusterMetrics(ctx, cluster)
		}
		return fallbackClusterMetrics(cluster), nil
	}

	deployments, err := clientset.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return models.ClusterMetrics{}, fmt.Errorf("list deployments for %s: %w", cluster.Name, err)
	}

	statefulSets, err := clientset.AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return models.ClusterMetrics{}, fmt.Errorf("list statefulsets for %s: %w", cluster.Name, err)
	}

	daemonSets, err := clientset.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return models.ClusterMetrics{}, fmt.Errorf("list daemonsets for %s: %w", cluster.Name, err)
	}

	workloads := make([]models.WorkloadSummary, 0, len(deployments.Items)+len(statefulSets.Items)+len(daemonSets.Items))
	workloads = append(workloads, collectDeployments(cluster.Name, deployments.Items, false, p.catalog)...)
	workloads = append(workloads, collectStatefulSets(cluster.Name, statefulSets.Items, p.catalog)...)
	workloads = append(workloads, collectDaemonSets(cluster.Name, daemonSets.Items, p.catalog)...)

	var cpuRequested float64
	var memoryRequested float64
	for _, workload := range workloads {
		cpuRequested += workload.CPURequest
		memoryRequested += workload.MemoryGiB
	}

	nodeCount := 0
	for _, pool := range cluster.WorkerPools {
		nodeCount += int(pool.Minimum)
	}
	if nodeCount == 0 {
		nodeCount = 1
	}

	cpuUtilization := min(100, (cpuRequested/float64(nodeCount*4))*100)
	memoryUtilization := min(100, (memoryRequested/float64(nodeCount*16))*100)
	idleScore := 100 - ((cpuUtilization + memoryUtilization) / 2)

	return models.ClusterMetrics{
		CPUUtilizationPercent:    cpuUtilization,
		MemoryUtilizationPercent: memoryUtilization,
		NodeCount:                nodeCount,
		IdleScore:                idleScore,
		Workloads:                workloads,
	}, nil
}

func collectDeployments(cluster string, items []appsv1.Deployment, stateful bool, catalog *pricing.Catalog) []models.WorkloadSummary {
	workloads := make([]models.WorkloadSummary, 0, len(items))
	for _, item := range items {
		workload := summarizeWorkload(cluster, item.Namespace, item.Name, "Deployment", derefReplicas(item.Spec.Replicas), stateful, item.Spec.Template.Spec.Containers)
		workload.MonthlyCost = catalog.EstimateWorkloadMonthlyCost(workload)
		workloads = append(workloads, workload)
	}

	return workloads
}

func collectStatefulSets(cluster string, items []appsv1.StatefulSet, catalog *pricing.Catalog) []models.WorkloadSummary {
	workloads := make([]models.WorkloadSummary, 0, len(items))
	for _, item := range items {
		workload := summarizeWorkload(cluster, item.Namespace, item.Name, "StatefulSet", derefReplicas(item.Spec.Replicas), true, item.Spec.Template.Spec.Containers)
		workload.MonthlyCost = catalog.EstimateWorkloadMonthlyCost(workload)
		workloads = append(workloads, workload)
	}

	return workloads
}

func collectDaemonSets(cluster string, items []appsv1.DaemonSet, catalog *pricing.Catalog) []models.WorkloadSummary {
	workloads := make([]models.WorkloadSummary, 0, len(items))
	for _, item := range items {
		workload := summarizeWorkload(cluster, item.Namespace, item.Name, "DaemonSet", item.Status.DesiredNumberScheduled, false, item.Spec.Template.Spec.Containers)
		workload.MonthlyCost = catalog.EstimateWorkloadMonthlyCost(workload)
		workloads = append(workloads, workload)
	}

	return workloads
}

func summarizeWorkload(cluster string, namespace string, name string, kind string, replicas int32, stateful bool, containers []corev1.Container) models.WorkloadSummary {
	var cpuRequested float64
	var memoryRequested float64
	for _, container := range containers {
		cpuRequested += float64(container.Resources.Requests.Cpu().MilliValue()) / 1000
		memoryRequested += container.Resources.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024)
	}

	return models.WorkloadSummary{
		Cluster:    cluster,
		Namespace:  namespace,
		Name:       name,
		Kind:       kind,
		Replicas:   replicas,
		CPURequest: cpuRequested * float64(maxInt32(replicas, 1)),
		MemoryGiB:  memoryRequested * float64(maxInt32(replicas, 1)),
		Stateful:   stateful,
	}
}

func fallbackClusterMetrics(cluster models.ClusterSummary) models.ClusterMetrics {
	idleScore := 80.0
	if cluster.Hibernated {
		idleScore = 95
	}

	return models.ClusterMetrics{
		CPUUtilizationPercent:    100 - idleScore,
		MemoryUtilizationPercent: 100 - idleScore,
		NodeCount:                maxInt(len(cluster.WorkerPools), 1),
		IdleScore:                idleScore,
		Workloads:                []models.WorkloadSummary{},
	}
}

func derefReplicas(value *int32) int32 {
	if value == nil {
		return 1
	}

	return *value
}

func min(a float64, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func maxInt(first int, second int) int {
	if first > second {
		return first
	}

	return second
}

func maxInt32(first int32, second int32) int32 {
	if first > second {
		return first
	}

	return second
}
