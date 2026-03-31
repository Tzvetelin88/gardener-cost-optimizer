package gardener

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"smart-cost-optimizer/backend/internal/models"
	"smart-cost-optimizer/backend/internal/pricing"
)

var shootGVR = schema.GroupVersionResource{
	Group:    "core.gardener.cloud",
	Version:  "v1beta1",
	Resource: "shoots",
}

type Client struct {
	dynamic dynamic.Interface
	catalog *pricing.Catalog
}

func NewClient(kubeconfigPath string, contextName string, catalog *pricing.Catalog) (*Client, error) {
	var (
		cfg *rest.Config
		err error
	)

	if kubeconfigPath == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
		overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("load gardener config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create gardener dynamic client: %w", err)
	}

	return &Client{
		dynamic: dynamicClient,
		catalog: catalog,
	}, nil
}

func (c *Client) ListClusters(ctx context.Context) ([]models.ClusterSummary, error) {
	list, err := c.dynamic.Resource(shootGVR).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list shoots: %w", err)
	}

	clusters := make([]models.ClusterSummary, 0, len(list.Items))
	for _, item := range list.Items {
		cluster := c.toCluster(item)
		cluster.MonthlyCost = c.catalog.EstimateClusterMonthlyCost(&cluster)
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (c *Client) SetHibernation(ctx context.Context, namespace string, name string, enabled bool) error {
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"hibernation": map[string]interface{}{
				"enabled": enabled,
			},
		},
	}

	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal hibernation patch: %w", err)
	}

	_, err = c.dynamic.Resource(shootGVR).Namespace(namespace).Patch(ctx, name, k8stypes.MergePatchType, payload, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patch hibernation for shoot %s/%s: %w", namespace, name, err)
	}

	return nil
}

func (c *Client) ScaleWorkerPool(ctx context.Context, namespace string, name string, poolName string, minimum int64, maximum int64) error {
	shoot, err := c.dynamic.Resource(shootGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get shoot %s/%s: %w", namespace, name, err)
	}

	workers, found, err := unstructured.NestedSlice(shoot.Object, "spec", "provider", "workers")
	if err != nil || !found {
		return fmt.Errorf("read worker pools for %s/%s: %w", namespace, name, err)
	}

	for _, item := range workers {
		worker, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if worker["name"] == poolName {
			worker["minimum"] = minimum
			worker["maximum"] = maximum
		}
	}

	if err := unstructured.SetNestedSlice(shoot.Object, workers, "spec", "provider", "workers"); err != nil {
		return fmt.Errorf("set worker pools for %s/%s: %w", namespace, name, err)
	}

	if _, err := c.dynamic.Resource(shootGVR).Namespace(namespace).Update(ctx, shoot, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update worker pools for %s/%s: %w", namespace, name, err)
	}

	return nil
}

func (c *Client) toCluster(item unstructured.Unstructured) models.ClusterSummary {
	namespace := item.GetNamespace()
	name := item.GetName()
	cloud, _, _ := unstructured.NestedString(item.Object, "spec", "provider", "type")
	region, _, _ := unstructured.NestedString(item.Object, "spec", "region")
	seed, _, _ := unstructured.NestedString(item.Object, "status", "seedName")
	if seed == "" {
		seed, _, _ = unstructured.NestedString(item.Object, "spec", "seedName")
	}
	hibernated, _, _ := unstructured.NestedBool(item.Object, "spec", "hibernation", "enabled")
	purpose, _, _ := unstructured.NestedString(item.Object, "metadata", "labels", "optimizer.sap.io/purpose")
	if purpose == "" {
		purpose = "general"
	}

	return models.ClusterSummary{
		ID:          namespace + "/" + name,
		Name:        name,
		Project:     namespace,
		Cloud:       cloud,
		Region:      region,
		Seed:        seed,
		Purpose:     purpose,
		Hibernated:  hibernated,
		WorkerPools: extractWorkerPools(item.Object),
		Labels:      item.GetLabels(),
	}
}

func extractWorkerPools(object map[string]interface{}) []models.WorkerPool {
	workers, found, err := unstructured.NestedSlice(object, "spec", "provider", "workers")
	if err != nil || !found {
		return nil
	}

	pools := make([]models.WorkerPool, 0, len(workers))
	for _, item := range workers {
		worker, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		machine, _ := worker["machine"].(map[string]interface{})
		machineType, _ := machine["type"].(string)
		name, _ := worker["name"].(string)
		minimum := numericValue(worker["minimum"])
		maximum := numericValue(worker["maximum"])

		pools = append(pools, models.WorkerPool{
			Name:        name,
			MachineType: machineType,
			Minimum:     minimum,
			Maximum:     maximum,
		})
	}

	return pools
}

func numericValue(raw interface{}) int64 {
	switch value := raw.(type) {
	case int:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}
