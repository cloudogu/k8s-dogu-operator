package exposition

import (
	"context"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newNormalizedRoutes() []serviceaccess.Route {
	return []serviceaccess.Route{
		{
			Name:     "admin",
			Port:     80,
			Location: "/admin",
			Pass:     "/admin",
		},
		{
			Name:     "admin-api",
			Port:     80,
			Location: "/api",
			Pass:     "/admin/api/v2/",
		},
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager(nil, nil, nil)

	require.NotNil(t, manager)
	assert.Nil(t, manager.client)
	assert.Nil(t, manager.doguFetcher)
	assert.Nil(t, manager.imageRegistry)
}

func TestExpositionManager_EnsureExposition(t *testing.T) {
	ctx := context.Background()
	doguResource := newDoguResource()
	doguService := newDoguService()

	t.Run("should fail if dogu resource is nil", func(t *testing.T) {
		manager := &ExpositionManager{}

		err := manager.EnsureExposition(ctx, nil, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu resource must not be nil")
	})

	t.Run("should return fetch descriptor error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(nil, assert.AnError)
		manager := &ExpositionManager{
			client:        newMockExpositionClient(t),
			doguFetcher:   fetcher,
			imageRegistry: newMockImageRegistry(t),
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch dogu descriptor")
	})

	t.Run("should return image config error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(nil, assert.AnError)
		manager := &ExpositionManager{
			client:        newMockExpositionClient(t),
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to pull image config")
	})

	t.Run("should return collect routes error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(&imagev1.ConfigFile{
			Config: imagev1.Config{
				Env: []string{
					"SERVICE_TAGS-invalidEnvironmentVariable",
				},
			},
		}, nil)
		manager := &ExpositionManager{
			client:        newMockExpositionClient(t),
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to collect web routes")
	})

	t.Run("should delete existing exposition if no routes exist", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(&imagev1.ConfigFile{}, nil)
		client := newMockExpositionClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(nil)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.NoError(t, err)
	})

	t.Run("should create exposition if it does not exist", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, newNotFoundErr("redmine"))
		client.EXPECT().Create(ctx, mock.MatchedBy(func(exp *expv1.Exposition) bool {
			return exp != nil &&
				exp.Name == "redmine" &&
				len(exp.Spec.HTTP) == 2 &&
				exp.Spec.HTTP[0].Service == "redmine"
		}), metav1.CreateOptions{}).Return(&expv1.Exposition{}, nil)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.NoError(t, err)
	})

	t.Run("should return get exposition error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, assert.AnError)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get Exposition")
	})

	t.Run("should return create exposition error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, newNotFoundErr("redmine"))
		client.EXPECT().Create(ctx, mock.Anything, metav1.CreateOptions{}).Return(nil, assert.AnError)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		resultErr := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, resultErr)
		assert.ErrorContains(t, resultErr, "failed to create Exposition")
	})

	t.Run("should not update if spec already matches", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		existingSpec, buildErr := buildSpec("redmine", newNormalizedRoutes())
		require.NoError(t, buildErr)
		existing := &expv1.Exposition{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind("Dogu"))},
			},
			Spec: existingSpec,
		}
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		resultErr := manager.EnsureExposition(ctx, doguResource, doguService)

		require.NoError(t, resultErr)
	})

	t.Run("should update exposition if spec differs", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		existing := &expv1.Exposition{
			ObjectMeta: metav1.ObjectMeta{Name: "redmine"},
			Spec:       expv1.ExpositionSpec{HTTP: []expv1.HTTPEntry{{Name: "old", Service: "redmine", Port: 80, Path: "/old"}}},
		}
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(ctx, mock.MatchedBy(func(exp *expv1.Exposition) bool {
			return exp != nil && len(exp.Spec.HTTP) == 2
		}), metav1.UpdateOptions{}).Return(existing, nil)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		resultErr := manager.EnsureExposition(ctx, doguResource, doguService)

		require.NoError(t, resultErr)
	})

	t.Run("should update exposition if owner references differ", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		existingSpec, buildErr := buildSpec("redmine", newNormalizedRoutes())
		require.NoError(t, buildErr)
		existing := &expv1.Exposition{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "redmine",
				OwnerReferences: []metav1.OwnerReference{},
			},
			Spec: existingSpec,
		}
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(ctx, mock.MatchedBy(func(exp *expv1.Exposition) bool {
			return exp != nil && len(exp.OwnerReferences) == 1
		}), metav1.UpdateOptions{}).Return(existing, nil)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		resultErr := manager.EnsureExposition(ctx, doguResource, doguService)

		require.NoError(t, resultErr)
	})

	t.Run("should return update exposition error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptor(), nil)
		imageRegistry := newMockImageRegistry(t)
		imageRegistry.EXPECT().PullImageConfig(ctx, "cloudogu/redmine:1.0.0").Return(newImageConfigWithRoutes(), nil)
		existing := &expv1.Exposition{
			ObjectMeta: metav1.ObjectMeta{Name: "redmine"},
			Spec:       expv1.ExpositionSpec{HTTP: []expv1.HTTPEntry{{Name: "old", Service: "redmine", Port: 80, Path: "/old"}}},
		}
		client := newMockExpositionClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(ctx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
		manager := &ExpositionManager{
			client:        client,
			doguFetcher:   fetcher,
			imageRegistry: imageRegistry,
		}

		err := manager.EnsureExposition(ctx, doguResource, doguService)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update Exposition")
	})
}

func TestExpositionManager_RemoveExposition(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove exposition", func(t *testing.T) {
		client := newMockExpositionClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(nil)
		manager := &ExpositionManager{client: client}

		err := manager.RemoveExposition(ctx, "redmine")

		require.NoError(t, err)
	})

	t.Run("should ignore not found", func(t *testing.T) {
		client := newMockExpositionClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(newNotFoundErr("redmine"))
		manager := &ExpositionManager{client: client}

		err := manager.RemoveExposition(ctx, "redmine")

		require.NoError(t, err)
	})

	t.Run("should return delete error", func(t *testing.T) {
		client := newMockExpositionClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(assert.AnError)
		manager := &ExpositionManager{client: client}

		err := manager.RemoveExposition(ctx, "redmine")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete Exposition")
	})
}

func TestCreateExpositionName(t *testing.T) {
	assert.Equal(t, "redmine", "redmine")
}

func newDoguResource() *doguv2.Dogu {
	return &doguv2.Dogu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s.cloudogu.com/v2",
			Kind:       "Dogu",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "redmine",
			UID:  "1234-5678",
		},
	}
}

func newDoguDescriptor() *cesappcore.Dogu {
	return &cesappcore.Dogu{
		Name:    "official/redmine",
		Image:   "cloudogu/redmine",
		Version: "1.0.0",
	}
}

func newDoguService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redmine",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
		},
	}
}

func newImageConfigWithRoutes() *imagev1.ConfigFile {
	return &imagev1.ConfigFile{
		Config: imagev1.Config{
			Labels: map[string]string{
				"SERVICE_TAGS": "webapp",
				"SERVICE_NAME": "admin",
			},
			Env: []string{
				"SERVICE_ADDITIONAL_SERVICES=[{\"name\":\"admin-api\",\"port\":80,\"location\":\"api\",\"pass\":\"admin/api/v2/\"}]",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		},
	}
}

func newNotFoundErr(name string) error {
	return k8sErr.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "expositions"}, name)
}
