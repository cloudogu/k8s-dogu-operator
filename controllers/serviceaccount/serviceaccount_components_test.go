package serviceaccount

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	testingk8s "k8s.io/client-go/testing"
	"testing"
)

func Test_creator_createComponentServiceAccount(t *testing.T) {
	ctx := context.TODO()

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
			EXPECT().createServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		serviceAccountCreator := creator{
			clientSet:         fakeClient,
			apiClient:         mockApiClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

		require.NoError(t, err)
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

		entries := config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		}

		doguCfg := config.CreateDoguConfig("test", entries)

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

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

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

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

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

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
			EXPECT().createServiceAccount(ctx, "http://1.2.3.4:8080/serviceaccounts", "defaultApiKeySecret", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		serviceAccountCreator := creator{
			clientSet:         fakeClient,
			apiClient:         mockApiClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

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

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get apiKey-secret: error reading apiKeySecret k8s-prometheus-api-key: secrets \"k8s-prometheus-api-key\" not found")
	})

	t.Run("fail on error getting credentials", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
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

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get credetials for service account:")
	})

	t.Run("fail on error saving credentials", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().createServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana", []string{"param1", "42"}).
			Return(map[string]string{"username": "adminUser", "password": "password123"}, nil)

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, assert.AnError).Once()

		serviceAccountCreator := creator{
			clientSet:         fakeClient,
			apiClient:         mockApiClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", make(config.Entries))

		err := serviceAccountCreator.createComponentServiceAccount(ctx, dogu, &doguCfg, serviceAccount, "sa-k8s-prometheus")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to save the service account credentials")
	})
}

func Test_readApiKeySecret(t *testing.T) {
	ctx := context.TODO()

	apiKeySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mySecret",
			Namespace: "testNs",
		},
		Data: map[string][]byte{
			"theKey": []byte("secretKey"),
		},
	}

	t.Run("success create component service account", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(apiKeySecret)
		fakeSecretsClient := fakeClient.CoreV1().Secrets("testNs")

		apiKey, err := readApiKeySecret(ctx, fakeSecretsClient, "mySecret", "theKey")

		require.NoError(t, err)
		assert.Equal(t, "secretKey", apiKey)
	})

	t.Run("fail on error getting secret", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(apiKeySecret)
		fakeSecretsClient := fakeClient.CoreV1().Secrets("testNs")

		_, err := readApiKeySecret(ctx, fakeSecretsClient, "noExist", "theKey")

		require.Error(t, err)
		assert.ErrorContains(t, err, "error reading apiKeySecret noExist:")
	})

	t.Run("fail on error getting key from secret", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(apiKeySecret)
		fakeSecretsClient := fakeClient.CoreV1().Secrets("testNs")

		_, err := readApiKeySecret(ctx, fakeSecretsClient, "mySecret", "otherKey")

		require.Error(t, err)
		assert.ErrorContains(t, err, "could not find key 'otherKey' in secret 'mySecret'")
	})
}

func Test_getServiceForLabels(t *testing.T) {
	ctx := context.TODO()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-provider-svc",
			Namespace: "testNs",
			Labels: map[string]string{
				saLabelProviderSvc: "k8s-prometheus",
			},
		},
	}

	t.Run("success create component service account", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc)
		fakeServicesClient := fakeClient.CoreV1().Services("testNs")

		result, err := getServiceForLabels(ctx, fakeServicesClient, fmt.Sprintf("%s=%s", saLabelProviderSvc, "k8s-prometheus"))

		require.NoError(t, err)
		assert.Equal(t, svc, result)
	})

	t.Run("fail on error listing services", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc)
		fakeClient.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("list", "services", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, assert.AnError
		})
		fakeServicesClient := fakeClient.CoreV1().Services("testNs")

		_, err := getServiceForLabels(ctx, fakeServicesClient, "a=b")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get services")
	})

	t.Run("fail no not finding matching services", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		fakeServicesClient := fakeClient.CoreV1().Services("testNs")

		_, err := getServiceForLabels(ctx, fakeServicesClient, fmt.Sprintf("%s=%s", saLabelProviderSvc, "k8s-prometheus"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "found no services for labelSelector ces.cloudogu.com/serviceaccount-provider=k8s-prometheus")
	})

	t.Run("fail no not finding matching services", func(t *testing.T) {
		svc2 := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sa-provider-svc-2",
				Namespace: "testNs",
				Labels: map[string]string{
					saLabelProviderSvc: "k8s-prometheus",
				},
			},
		}

		fakeClient := fake.NewSimpleClientset(svc, svc2)
		fakeServicesClient := fakeClient.CoreV1().Services("testNs")

		_, err := getServiceForLabels(ctx, fakeServicesClient, fmt.Sprintf("%s=%s", saLabelProviderSvc, "k8s-prometheus"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "found more than one service for labelSelector ces.cloudogu.com/serviceaccount-provider=k8s-prometheus")
	})
}

func Test_remover_removeComponentServiceAccount(t *testing.T) {
	ctx := context.TODO()

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

	t.Run("success remove component service account", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().deleteServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana").
			Return(nil)

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		serviceAccountRemover := remover{
			clientSet:         fakeClient,
			apiClient:         mockApiClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.NoError(t, err)
	})

	t.Run("success remove component service account that does not exists", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)

		serviceAccountRemover := remover{
			clientSet: fakeClient,
			apiClient: mockApiClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.NoError(t, err)
	})

	t.Run("fail on error getting service", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)

		serviceAccountRemover := remover{
			clientSet: fakeClient,
			apiClient: mockApiClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service: found no services for labelSelector ces.cloudogu.com/serviceaccount-provider=k8s-prometheus")
	})

	t.Run("fail on error getting apiKey", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc)

		mockApiClient := newMockServiceAccountApiClient(t)

		serviceAccountRemover := remover{
			clientSet: fakeClient,
			apiClient: mockApiClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error getting apiKey: failed to get apiKey-secret: error reading apiKeySecret k8s-prometheus-api-key: secrets \"k8s-prometheus-api-key\" not found")
	})

	t.Run("fail on error deleting service account", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().deleteServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana").
			Return(assert.AnError)

		serviceAccountRemover := remover{
			clientSet: fakeClient,
			apiClient: mockApiClient,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to remove service account:")
	})

	t.Run("fail on error deleting service account from doguConfig", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(svc, apiKeySecret)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.
			EXPECT().deleteServiceAccount(ctx, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana").
			Return(nil)

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		serviceAccountRemover := remover{
			clientSet:         fakeClient,
			apiClient:         mockApiClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
		}

		dogu := &core.Dogu{
			Name:                 "official/grafana",
			Dependencies:         []core.Dependency{},
			OptionalDependencies: []core.Dependency{},
		}

		serviceAccount := core.ServiceAccount{
			Type:   "k8s-prometheus",
			Kind:   "component",
			Params: []string{"param1", "42"},
		}

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		err := serviceAccountRemover.removeComponentServiceAccount(ctx, dogu, serviceAccount, &doguCfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed write config for dogu test after updating")
	})

}
