package actions

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type deploymentDependencies struct {
	serviceAccount string
	configMaps     map[string]struct{}
	secrets        map[string]struct{}
}

func (s *Service) prepareTargetNamespace(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string) error {
	if _, err := targetClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get target namespace %s: %w", namespace, err)
	}

	sourceNamespace, err := sourceClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get source namespace %s: %w", namespace, err)
		}

		sourceNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
	}

	namespaceCopy := sourceNamespace.DeepCopy()
	namespaceCopy.ResourceVersion = ""
	namespaceCopy.UID = ""
	namespaceCopy.ManagedFields = nil
	namespaceCopy.Spec = corev1.NamespaceSpec{}
	namespaceCopy.Status = corev1.NamespaceStatus{}

	if _, err := targetClient.CoreV1().Namespaces().Create(ctx, namespaceCopy, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create target namespace %s: %w", namespace, err)
	}

	return nil
}

func (s *Service) syncDeploymentDependencies(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string, deployment *appsv1.Deployment) error {
	dependencies := collectDeploymentDependencies(deployment)

	if dependencies.serviceAccount != "" && dependencies.serviceAccount != "default" {
		if err := syncServiceAccount(ctx, sourceClient, targetClient, namespace, dependencies.serviceAccount); err != nil {
			return err
		}
	}

	for name := range dependencies.configMaps {
		if err := syncConfigMap(ctx, sourceClient, targetClient, namespace, name); err != nil {
			return err
		}
	}

	for name := range dependencies.secrets {
		if err := syncSecret(ctx, sourceClient, targetClient, namespace, name); err != nil {
			return err
		}
	}

	if err := syncServicesForDeployment(ctx, sourceClient, targetClient, namespace, deployment); err != nil {
		return err
	}

	return nil
}

func collectDeploymentDependencies(deployment *appsv1.Deployment) deploymentDependencies {
	dependencies := deploymentDependencies{
		serviceAccount: deployment.Spec.Template.Spec.ServiceAccountName,
		configMaps:     map[string]struct{}{},
		secrets:        map[string]struct{}{},
	}

	for _, secretRef := range deployment.Spec.Template.Spec.ImagePullSecrets {
		dependencies.secrets[secretRef.Name] = struct{}{}
	}

	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			dependencies.configMaps[volume.ConfigMap.Name] = struct{}{}
		}
		if volume.Secret != nil {
			dependencies.secrets[volume.Secret.SecretName] = struct{}{}
		}
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.ConfigMap != nil {
					dependencies.configMaps[source.ConfigMap.Name] = struct{}{}
				}
				if source.Secret != nil {
					dependencies.secrets[source.Secret.Name] = struct{}{}
				}
			}
		}
	}

	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, envVar := range container.Env {
			if envVar.ValueFrom == nil {
				continue
			}
			if envVar.ValueFrom.ConfigMapKeyRef != nil {
				dependencies.configMaps[envVar.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
			}
			if envVar.ValueFrom.SecretKeyRef != nil {
				dependencies.secrets[envVar.ValueFrom.SecretKeyRef.Name] = struct{}{}
			}
		}

		for _, source := range container.EnvFrom {
			if source.ConfigMapRef != nil {
				dependencies.configMaps[source.ConfigMapRef.Name] = struct{}{}
			}
			if source.SecretRef != nil {
				dependencies.secrets[source.SecretRef.Name] = struct{}{}
			}
		}
	}

	return dependencies
}

func syncServiceAccount(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string, name string) error {
	account, err := sourceClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get source service account %s/%s: %w", namespace, name, err)
	}

	copyAccount := account.DeepCopy()
	copyAccount.ResourceVersion = ""
	copyAccount.UID = ""
	copyAccount.ManagedFields = nil
	copyAccount.Secrets = nil

	current, err := targetClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		copyAccount.ResourceVersion = current.ResourceVersion
		if _, err := targetClient.CoreV1().ServiceAccounts(namespace).Update(ctx, copyAccount, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update target service account %s/%s: %w", namespace, name, err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get target service account %s/%s: %w", namespace, name, err)
	}

	if _, err := targetClient.CoreV1().ServiceAccounts(namespace).Create(ctx, copyAccount, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create target service account %s/%s: %w", namespace, name, err)
	}

	return nil
}

func syncConfigMap(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string, name string) error {
	configMap, err := sourceClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get source configmap %s/%s: %w", namespace, name, err)
	}

	copyConfigMap := configMap.DeepCopy()
	copyConfigMap.ResourceVersion = ""
	copyConfigMap.UID = ""
	copyConfigMap.ManagedFields = nil

	current, err := targetClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		copyConfigMap.ResourceVersion = current.ResourceVersion
		if _, err := targetClient.CoreV1().ConfigMaps(namespace).Update(ctx, copyConfigMap, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update target configmap %s/%s: %w", namespace, name, err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get target configmap %s/%s: %w", namespace, name, err)
	}

	if _, err := targetClient.CoreV1().ConfigMaps(namespace).Create(ctx, copyConfigMap, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create target configmap %s/%s: %w", namespace, name, err)
	}

	return nil
}

func syncSecret(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string, name string) error {
	secret, err := sourceClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get source secret %s/%s: %w", namespace, name, err)
	}
	if secret.Type == corev1.SecretTypeServiceAccountToken {
		return nil
	}

	copySecret := secret.DeepCopy()
	copySecret.ResourceVersion = ""
	copySecret.UID = ""
	copySecret.ManagedFields = nil

	current, err := targetClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		copySecret.ResourceVersion = current.ResourceVersion
		if _, err := targetClient.CoreV1().Secrets(namespace).Update(ctx, copySecret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update target secret %s/%s: %w", namespace, name, err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get target secret %s/%s: %w", namespace, name, err)
	}

	if _, err := targetClient.CoreV1().Secrets(namespace).Create(ctx, copySecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create target secret %s/%s: %w", namespace, name, err)
	}

	return nil
}

func syncServicesForDeployment(ctx context.Context, sourceClient *kubernetes.Clientset, targetClient *kubernetes.Clientset, namespace string, deployment *appsv1.Deployment) error {
	services, err := sourceClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list source services in %s: %w", namespace, err)
	}

	for _, service := range services.Items {
		if !selectorMatches(service.Spec.Selector, deployment.Spec.Template.Labels) {
			continue
		}

		copyService := service.DeepCopy()
		copyService.ResourceVersion = ""
		copyService.UID = ""
		copyService.ManagedFields = nil
		if copyService.Spec.ClusterIP == corev1.ClusterIPNone {
			copyService.Spec.ClusterIP = corev1.ClusterIPNone
		} else {
			copyService.Spec.ClusterIP = ""
			copyService.Spec.ClusterIPs = nil
		}
		copyService.Spec.HealthCheckNodePort = 0
		copyService.Spec.IPFamilies = nil
		copyService.Spec.IPFamilyPolicy = nil
		copyService.Status = corev1.ServiceStatus{}

		current, err := targetClient.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
		if err == nil {
			copyService.ResourceVersion = current.ResourceVersion
			copyService.Spec.ClusterIP = current.Spec.ClusterIP
			copyService.Spec.ClusterIPs = current.Spec.ClusterIPs
			copyService.Spec.HealthCheckNodePort = current.Spec.HealthCheckNodePort
			if _, err := targetClient.CoreV1().Services(namespace).Update(ctx, copyService, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update target service %s/%s: %w", namespace, service.Name, err)
			}
			continue
		}
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get target service %s/%s: %w", namespace, service.Name, err)
		}

		if _, err := targetClient.CoreV1().Services(namespace).Create(ctx, copyService, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create target service %s/%s: %w", namespace, service.Name, err)
		}
	}

	return nil
}

func selectorMatches(selector map[string]string, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}

	return true
}
