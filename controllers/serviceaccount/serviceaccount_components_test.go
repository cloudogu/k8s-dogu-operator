package serviceaccount

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func Test_creator_createComponentServiceAccount(t *testing.T) {
	ctx := context.TODO()
	validPubKey := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApbhnnaIIXCADt0V7UCM7\nZfBEhpEeB5LTlvISkPQ91g+l06/soWFD65ba0PcZbIeKFqr7vkMB0nDNxX1p8PGv\nVJdUmwdB7U/bQlnO6c1DoY10g29O7itDfk92RCKeU5Vks9uRQ5ayZMjxEuahg2BW\nua72wi3GCiwLa9FZxGIP3hcYB21O6PfpxXsQYR8o3HULgL1ppDpuLv4fk/+jD31Z\n9ACoWOg6upyyNUsiA3hS9Kn1p3scVgsIN2jSSpxW42NvMo6KQY1Zo0N4Aw/mqySd\n+zdKytLqFto1t0gCbTCFPNMIObhWYXmAe26+h1b1xUI8ymsrXklwJVn0I77j9MM1\nHQIDAQAB\n-----END PUBLIC KEY-----"

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-provider-svc",
			Namespace: "testNs",
			Annotations: map[string]string{
				saAnnotationPort:       "9977",
				saAnnotationPath:       "/sa-management",
				saAnnotationSecretName: "k8s-prometheus-api-key",
				saAnnotationSecretKey:  "theApiKey",
			},
			Labels: map[string]string{
				"app":              "ces",
				saLabelProviderSvc: "k8s-prometheus",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}

	apiKeySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-prometheus-api-key",
			Namespace: "testNs",
		},
		Data: map[string][]byte{
			"theApiKey": []byte("secretKey"),
		},
	}

	t.Run("success create component service account", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount("http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
			apiClient: mockApiClient,
			registry:  registry,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/password", mock.Anything).Return(nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.NoError(t, err)
	})

	t.Run("fail on error checking service account exists", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, assert.AnError)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("success when service account exists", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(true, nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.NoError(t, err)
	})

	t.Run("success on error get service and is optional", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		serviceAccountCreator := creator{
			clientSet: fakeClient,
		}

		dogu := &core.Dogu{
			Name:         "official/grafana",
			Dependencies: []core.Dependency{},
			OptionalDependencies: []core.Dependency{
				{Name: "k8s-prometheus"},
			},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.NoError(t, err)
	})

	t.Run("fail on error get service and is not optional", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		serviceAccountCreator := creator{
			clientSet: fakeClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service: found no services for labelSelector ces.cloudogu.com/serviceaccount-provider=k8s-prometheus")
	})

	t.Run("success create component service account with default annotation values", func(t *testing.T) {
		svcDefaultAnnotaions := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sa-provider-svc",
				Namespace: "testNs",
				Annotations: map[string]string{
					saAnnotationSecretName: "k8s-prometheus-api-key",
				},
				Labels: map[string]string{
					"app":              "ces",
					saLabelProviderSvc: "k8s-prometheus",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "1.2.3.4",
			},
		}

		apiKeySecretDefaultAnnotations := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k8s-prometheus-api-key",
				Namespace: "testNs",
			},
			Data: map[string][]byte{
				"apiKey": []byte("defaultApiKeySecret"),
			},
		}

		fakeClient := fake.NewSimpleClientset(svcDefaultAnnotaions, apiKeySecretDefaultAnnotations)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount("http://1.2.3.4:8080/serviceaccounts", "defaultApiKeySecret", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
			apiClient: mockApiClient,
			registry:  registry,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/password", mock.Anything).Return(nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.NoError(t, err)
	})

	t.Run("fail on error read apiKey secret", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get apiKey-secret: error reading apiKeySecret k8s-prometheus-api-key: secrets \"k8s-prometheus-api-key\" not found")
	})

	t.Run("fail on error getting credentials", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount("http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
			Return(nil, assert.AnError)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
			apiClient: mockApiClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get credetials for service account:")
	})

	t.Run("fail on error saving credentials", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount("http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		serviceAccountCreator := creator{
			clientSet: fakeClient,
			apiClient: mockApiClient,
			registry:  registry,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "/sa/k8s-prometheus").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-k8s-prometheus/password", mock.Anything).Return(assert.AnError)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, doguConfig, serviceAccount, "/sa/k8s-prometheus")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to save the service account credentials: failed to write service account: failed to set encrypted sa value of key password:")
	})
}
