package actions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"smart-cost-optimizer/backend/internal/gardener"
	"smart-cost-optimizer/backend/internal/models"
)

type Service struct {
	gardener    *gardener.Client
	mock        mockBackend
	shoots      map[string]*kubernetes.Clientset
	mu          sync.RWMutex
	actionStore []models.ActionRecord
	logPath     string
}

type mockBackend interface {
	SetHibernation(context.Context, string, string, bool) error
	ScaleWorkerPool(context.Context, string, string, string, int64, int64) error
	MoveWorkload(context.Context, string, string, string, string) error
}

func NewService(gardenerClient *gardener.Client, shootKubeconfigs map[string]string, mock mockBackend) (*Service, error) {
	return NewServiceWithLogPath(gardenerClient, shootKubeconfigs, mock, "")
}

func NewServiceWithLogPath(gardenerClient *gardener.Client, shootKubeconfigs map[string]string, mock mockBackend, logPath string) (*Service, error) {
	shootClients := map[string]*kubernetes.Clientset{}
	for name, kubeconfig := range shootKubeconfigs {
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build shoot client config for %s: %w", name, err)
		}

		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("create shoot client for %s: %w", name, err)
		}

		shootClients[name] = clientset
	}

	svc := &Service{
		gardener: gardenerClient,
		mock:     mock,
		shoots:   shootClients,
		logPath:  logPath,
	}

	if logPath != "" {
		svc.actionStore = loadActionLog(logPath)
	}

	return svc, nil
}

func loadActionLog(path string) []models.ActionRecord {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var records []models.ActionRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec models.ActionRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			log.Printf("skip malformed action log line: %v", err)
			continue
		}

		records = append(records, rec)
	}

	return records
}

func (s *Service) appendActionLog(record models.ActionRecord) {
	if s.logPath == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o755); err != nil {
		log.Printf("create action log dir: %v", err)
		return
	}

	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("open action log: %v", err)
		return
	}
	defer f.Close()

	line, err := json.Marshal(record)
	if err != nil {
		log.Printf("marshal action record: %v", err)
		return
	}

	_, _ = f.Write(append(line, '\n'))
}

func (s *Service) List() []models.ActionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]models.ActionRecord, len(s.actionStore))
	copy(records, s.actionStore)
	return records
}

func (s *Service) HibernateCluster(ctx context.Context, clusterID string) (models.ActionRecord, error) {
	namespace, name := splitClusterID(clusterID)
	if s.gardener != nil {
		if err := s.gardener.SetHibernation(ctx, namespace, name, true); err != nil {
			return s.record("hibernate-cluster", clusterID, "failed", err.Error()), err
		}
		return s.record("hibernate-cluster", clusterID, "completed", "Cluster hibernation requested through Gardener."), nil
	}

	if s.mock == nil {
		err := fmt.Errorf("gardener client is not configured")
		return s.record("hibernate-cluster", clusterID, "failed", err.Error()), err
	}

	if err := s.mock.SetHibernation(ctx, namespace, name, true); err != nil {
		return s.record("hibernate-cluster", clusterID, "failed", err.Error()), err
	}

	return s.record("hibernate-cluster", clusterID, "completed", "Mock cluster hibernation completed."), nil
}

func (s *Service) WakeCluster(ctx context.Context, clusterID string) (models.ActionRecord, error) {
	namespace, name := splitClusterID(clusterID)
	if s.gardener != nil {
		if err := s.gardener.SetHibernation(ctx, namespace, name, false); err != nil {
			return s.record("wake-cluster", clusterID, "failed", err.Error()), err
		}
		return s.record("wake-cluster", clusterID, "completed", "Cluster wake-up requested through Gardener."), nil
	}

	if s.mock == nil {
		err := fmt.Errorf("gardener client is not configured")
		return s.record("wake-cluster", clusterID, "failed", err.Error()), err
	}

	if err := s.mock.SetHibernation(ctx, namespace, name, false); err != nil {
		return s.record("wake-cluster", clusterID, "failed", err.Error()), err
	}

	return s.record("wake-cluster", clusterID, "completed", "Mock cluster wake-up completed."), nil
}

func (s *Service) ScaleNodePool(ctx context.Context, clusterID string, workerPool string, minimum int64, maximum int64) (models.ActionRecord, error) {
	namespace, name := splitClusterID(clusterID)
	if s.gardener != nil {
		if err := s.gardener.ScaleWorkerPool(ctx, namespace, name, workerPool, minimum, maximum); err != nil {
			return s.record("scale-nodepool", clusterID, "failed", err.Error()), err
		}
		return s.record("scale-nodepool", clusterID, "completed", "Worker pool sizing updated in Gardener."), nil
	}

	if s.mock == nil {
		err := fmt.Errorf("gardener client is not configured")
		return s.record("scale-nodepool", clusterID, "failed", err.Error()), err
	}

	if err := s.mock.ScaleWorkerPool(ctx, namespace, name, workerPool, minimum, maximum); err != nil {
		return s.record("scale-nodepool", clusterID, "failed", err.Error()), err
	}

	return s.record("scale-nodepool", clusterID, "completed", "Mock worker pool sizing updated."), nil
}

func (s *Service) MoveWorkload(ctx context.Context, sourceCluster string, targetCluster string, namespace string, workloadName string) (models.ActionRecord, error) {
	sourceClient, ok := s.shoots[clusterName(sourceCluster)]
	if !ok {
		if s.mock != nil {
			if err := s.mock.MoveWorkload(ctx, sourceCluster, targetCluster, namespace, workloadName); err != nil {
				return s.record("move-workload", workloadName, "failed", err.Error()), err
			}
			return s.record("move-workload", namespace+"/"+workloadName, "completed", "Mock deployment moved to cheaper target cluster."), nil
		}

		err := fmt.Errorf("no shoot kubeconfig configured for source cluster %s", sourceCluster)
		return s.record("move-workload", workloadName, "failed", err.Error()), err
	}

	targetClient, ok := s.shoots[clusterName(targetCluster)]
	if !ok {
		err := fmt.Errorf("no shoot kubeconfig configured for target cluster %s", targetCluster)
		return s.record("move-workload", workloadName, "failed", err.Error()), err
	}

	deployment, err := sourceClient.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{})
	if err != nil {
		return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
	}

	if err := s.prepareTargetNamespace(ctx, sourceClient, targetClient, namespace); err != nil {
		return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
	}

	if err := s.syncDeploymentDependencies(ctx, sourceClient, targetClient, namespace, deployment); err != nil {
		return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
	}

	cloned := cloneDeployment(deployment)
	if _, err := targetClient.AppsV1().Deployments(namespace).Create(ctx, cloned, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			current, getErr := targetClient.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{})
			if getErr != nil {
				return s.record("move-workload", namespace+"/"+workloadName, "failed", getErr.Error()), getErr
			}
			cloned.ResourceVersion = current.ResourceVersion
			if _, err = targetClient.AppsV1().Deployments(namespace).Update(ctx, cloned, metav1.UpdateOptions{}); err != nil {
				return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
			}
		} else {
			return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
		}
	}

	scaleToZero := int32(0)
	deployment.Spec.Replicas = &scaleToZero
	if _, err := sourceClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return s.record("move-workload", namespace+"/"+workloadName, "failed", err.Error()), err
	}

	return s.record("move-workload", namespace+"/"+workloadName, "completed", "Deployment moved to cheaper target cluster and source scaled to zero."), nil
}

func (s *Service) record(actionType string, target string, status string, message string) models.ActionRecord {
	rec := models.ActionRecord{
		ID:        fmt.Sprintf("%s-%d", actionType, time.Now().UnixNano()),
		Type:      actionType,
		Status:    status,
		Target:    target,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.actionStore = append([]models.ActionRecord{rec}, s.actionStore...)
	s.mu.Unlock()

	s.appendActionLog(rec)
	return rec
}

func splitClusterID(clusterID string) (string, string) {
	parts := strings.SplitN(clusterID, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "garden", clusterID
}

func clusterName(clusterID string) string {
	_, name := splitClusterID(clusterID)
	return name
}

func cloneDeployment(source *appsv1.Deployment) *appsv1.Deployment {
	cloned := source.DeepCopy()
	cloned.ResourceVersion = ""
	cloned.UID = ""
	cloned.ManagedFields = nil
	cloned.Status = appsv1.DeploymentStatus{}
	if cloned.Annotations == nil {
		cloned.Annotations = map[string]string{}
	}
	cloned.Annotations["optimizer.sap.io/moved-from"] = source.Namespace + "/" + source.Name
	return cloned
}
