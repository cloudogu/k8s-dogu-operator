package serviceaccount

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const saLabelProviderSvc = "ces.cloudogu.com/serviceaccount-provider"
const saAnnotationPort = "ces.cloudogu.com/serviceaccount-port"
const saAnnotationPath = "ces.cloudogu.com/serviceaccount-path"
const saAnnotationSecretName = "ces.cloudogu.com/serviceaccount-secret-name"
const saAnnotationSecretKey = "ces.cloudogu.com/serviceaccount-secret-key"

func (c *creator) createComponentServiceAccount(ctx context.Context, dogu *core.Dogu, senDoguCfg SensitiveDoguConfig, serviceAccount core.ServiceAccount, registryCredentialPath string) error {
	logger := log.FromContext(ctx)
	exists, err := serviceAccountExists(ctx, registryCredentialPath, senDoguCfg)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	saIsOptional := c.isOptionalServiceAccount(dogu, serviceAccount.Type)

	// get service for component of service account
	labelSelector := fmt.Sprintf("%s=%s", saLabelProviderSvc, serviceAccount.Type)
	servicesClient := c.clientSet.CoreV1().Services(c.namespace)
	service, err := getServiceForLabels(ctx, servicesClient, labelSelector)
	if err != nil && saIsOptional {
		logger.Info("Skipping creation of service account % because the service was not found and the service account is optional", serviceAccount.Type)
		return nil
	}
	if err != nil && !saIsOptional {
		return fmt.Errorf("failed to get service: %w", err)
	}

	saApiURL := getSaApiBaseUrl(service)

	apiKey, err := getApiKey(ctx, c.clientSet, service)
	if err != nil {
		return fmt.Errorf("error getting apiKey: %w", err)
	}

	saCredentials, err := c.apiClient.createServiceAccount(ctx, saApiURL, apiKey, dogu.GetSimpleName(), serviceAccount.Params)
	if err != nil {
		return fmt.Errorf("failed to get credetials for service account: %w", err)
	}

	err = c.writeServiceAccounts(ctx, senDoguCfg, serviceAccount, saCredentials)
	if err != nil {
		return fmt.Errorf("failed to save the service account credentials: %w", err)
	}

	return nil
}

// GetServiceForLabels returns a service for the given dogu labels. An error is returned if either no service or more than one service is found.
func getServiceForLabels(ctx context.Context, servicesClient v1.ServiceInterface, labelSelector string) (*corev1.Service, error) {
	services, err := servicesClient.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("found no services for labelSelector %s", labelSelector)
	}
	if len(services.Items) > 1 {
		return nil, fmt.Errorf("found more than one service for labelSelector %s", labelSelector)
	}

	return &services.Items[0], nil
}

func getSaApiBaseUrl(service *corev1.Service) string {
	port := getAnnotationOrDefault(service, saAnnotationPort, "8080")
	path := getAnnotationOrDefault(service, saAnnotationPath, "/serviceaccounts")

	return fmt.Sprintf("http://%s:%s%s", service.Spec.ClusterIP, port, path)
}

func getApiKey(ctx context.Context, clientSet kubernetes.Interface, service *corev1.Service) (string, error) {
	apiKeySecretName := getAnnotationOrDefault(service, saAnnotationSecretName, "")
	apiKeySecretKey := getAnnotationOrDefault(service, saAnnotationSecretKey, "apiKey")

	secretsClient := clientSet.CoreV1().Secrets(service.Namespace)
	apiKey, err := readApiKeySecret(ctx, secretsClient, apiKeySecretName, apiKeySecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to get apiKey-secret: %w", err)
	}

	return apiKey, nil
}

func readApiKeySecret(ctx context.Context, secretsClient v1.SecretInterface, secretName string, secretKey string) (string, error) {
	secret, err := secretsClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error reading apiKeySecret %s: %w", secretName, err)
	}

	apiKey, exists := secret.Data[secretKey]
	if !exists {
		return "", fmt.Errorf("could not find key '%s' in secret '%s'", secretKey, secretName)
	}

	return string(apiKey), nil
}

func getAnnotationOrDefault(pod *corev1.Service, name string, defaultValue string) string {
	value := pod.Annotations[name]
	if value == "" {
		return defaultValue
	}

	return value
}

func (r *remover) removeComponentServiceAccount(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, senDoguCfg SensitiveDoguConfig) error {
	logger := log.FromContext(ctx)
	registryCredentialPath := "sa-" + serviceAccount.Type

	exists, err := serviceAccountExists(ctx, registryCredentialPath, senDoguCfg)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("skipping removal of service account because it does not exists")
		return nil
	}

	// get service for component of service account
	labelSelector := fmt.Sprintf("%s=%s", saLabelProviderSvc, serviceAccount.Type)
	servicesClient := r.clientSet.CoreV1().Services(r.namespace)
	service, err := getServiceForLabels(ctx, servicesClient, labelSelector)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	saApiURL := getSaApiBaseUrl(service)

	apiKey, err := getApiKey(ctx, r.clientSet, service)
	if err != nil {
		return fmt.Errorf("error getting apiKey: %w", err)
	}

	if lErr := r.apiClient.deleteServiceAccount(ctx, saApiURL, apiKey, dogu.GetSimpleName()); lErr != nil {
		return fmt.Errorf("failed to remove service account: %w", lErr)
	}

	err = senDoguCfg.DeleteRecursive(ctx, registryCredentialPath)
	if err != nil {
		return fmt.Errorf("failed to remove service account from config: %w", err)
	}

	return nil
}
