package warpmenuentry

import (
	"context"
	"strings"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	warpv1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewManager(t *testing.T) {
	manager := NewManager(nil, nil)

	require.NotNil(t, manager)
	assert.Nil(t, manager.client)
	assert.Nil(t, manager.doguFetcher)
}

func TestWarpMenuEntryManager_EnsureWarpMenuEntry(t *testing.T) {
	ctx := context.Background()
	doguResource := newDoguResource()

	t.Run("should fail if dogu resource is nil", func(t *testing.T) {
		manager := &WarpMenuEntryManager{}

		err := manager.EnsureWarpMenuEntry(ctx, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu resource must not be nil")
	})

	t.Run("should return fetch descriptor error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(nil, assert.AnError)
		manager := &WarpMenuEntryManager{client: newMockWarpMenuEntryClient(t), doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch dogu descriptor")
	})

	t.Run("should delete existing entry when dogu has no warp tag", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptorWithoutWarpTag(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should ignore NotFound when deleting absent entry for dogu without warp tag", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newDoguDescriptorWithoutWarpTag(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(notFoundErr("redmine"))
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should not create entry when warp is only a substring of a tag", func(t *testing.T) {
		descriptor := newWarpDoguDescriptor()
		descriptor.Tags = []string{"warpdrive", "menu"}
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(descriptor, nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should create new entry with derived spec", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, notFoundErr("redmine"))
		var created *warpv1.WarpMenuEntry
		client.EXPECT().Create(ctx, mock.Anything, metav1.CreateOptions{}).
			Run(func(_ context.Context, entry *warpv1.WarpMenuEntry, _ metav1.CreateOptions) { created = entry }).
			Return(&warpv1.WarpMenuEntry{}, nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "redmine", created.Name)
		assert.Equal(t, "/redmine", created.Spec.Path)
		assert.Equal(t, "Development Apps", created.Spec.Category)
		assert.Equal(t, "Redmine", created.Spec.DisplayName.DE)
		assert.Equal(t, "Redmine", created.Spec.DisplayName.EN)
		assert.False(t, created.Spec.Disabled)
		assert.Equal(t, expectedOwnerReferences(doguResource), created.OwnerReferences)
	})

	t.Run("should fall back to descriptor name when display name is empty", func(t *testing.T) {
		descriptor := newWarpDoguDescriptor()
		descriptor.DisplayName = ""
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(descriptor, nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, notFoundErr("redmine"))
		var created *warpv1.WarpMenuEntry
		client.EXPECT().Create(ctx, mock.Anything, metav1.CreateOptions{}).
			Run(func(_ context.Context, entry *warpv1.WarpMenuEntry, _ metav1.CreateOptions) { created = entry }).
			Return(&warpv1.WarpMenuEntry{}, nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "official/redmine", created.Spec.DisplayName.DE)
		assert.Equal(t, "official/redmine", created.Spec.DisplayName.EN)
	})

	t.Run("should not update an already matching entry", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(newExistingMatchingEntry(doguResource), nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
		client.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should update an entry with diverging spec", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		existing := newExistingMatchingEntry(doguResource)
		existing.Spec.Category = "Outdated"
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		var updated *warpv1.WarpMenuEntry
		client.EXPECT().Update(ctx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(_ context.Context, entry *warpv1.WarpMenuEntry, _ metav1.UpdateOptions) { updated = entry }).
			Return(&warpv1.WarpMenuEntry{}, nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Development Apps", updated.Spec.Category)
	})

	t.Run("should update an entry with diverging owner references", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		existing := newExistingMatchingEntry(doguResource)
		existing.OwnerReferences = nil
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		var updated *warpv1.WarpMenuEntry
		client.EXPECT().Update(ctx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(_ context.Context, entry *warpv1.WarpMenuEntry, _ metav1.UpdateOptions) { updated = entry }).
			Return(&warpv1.WarpMenuEntry{}, nil)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expectedOwnerReferences(doguResource), updated.OwnerReferences)
	})

	t.Run("should return get error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, assert.AnError)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get WarpMenuEntry")
	})

	t.Run("should return create error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(nil, notFoundErr("redmine"))
		client.EXPECT().Create(ctx, mock.Anything, metav1.CreateOptions{}).Return(nil, assert.AnError)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create WarpMenuEntry")
	})

	t.Run("should return update error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(newWarpDoguDescriptor(), nil)
		existing := newExistingMatchingEntry(doguResource)
		existing.Spec.Path = "/outdated"
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Get(ctx, "redmine", metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(ctx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
		manager := &WarpMenuEntryManager{client: client, doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update WarpMenuEntry")
	})

	t.Run("should return error for too long display name", func(t *testing.T) {
		descriptor := newWarpDoguDescriptor()
		descriptor.DisplayName = strings.Repeat("a", 51)
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(descriptor, nil)
		manager := &WarpMenuEntryManager{client: newMockWarpMenuEntryClient(t), doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "display name")
		assert.ErrorContains(t, err, "exceeds maximum length")
	})

	t.Run("should return error for empty category", func(t *testing.T) {
		descriptor := newWarpDoguDescriptor()
		descriptor.Category = ""
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(descriptor, nil)
		manager := &WarpMenuEntryManager{client: newMockWarpMenuEntryClient(t), doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "category")
		assert.ErrorContains(t, err, "must not be empty")
	})

	t.Run("should return error for too long category", func(t *testing.T) {
		descriptor := newWarpDoguDescriptor()
		descriptor.Category = strings.Repeat("a", 51)
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(descriptor, nil)
		manager := &WarpMenuEntryManager{client: newMockWarpMenuEntryClient(t), doguFetcher: fetcher}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "category")
		assert.ErrorContains(t, err, "exceeds maximum length")
	})
}

func TestWarpMenuEntryManager_RemoveWarpMenuEntry(t *testing.T) {
	ctx := context.Background()
	doguResource := newDoguResource()

	t.Run("should delete entry", func(t *testing.T) {
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{client: client}

		err := manager.RemoveWarpMenuEntry(ctx, doguResource.GetSimpleDoguName())

		require.NoError(t, err)
	})

	t.Run("should ignore NotFound", func(t *testing.T) {
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(notFoundErr("redmine"))
		manager := &WarpMenuEntryManager{client: client}

		err := manager.RemoveWarpMenuEntry(ctx, doguResource.GetSimpleDoguName())

		require.NoError(t, err)
	})

	t.Run("should return delete error", func(t *testing.T) {
		client := newMockWarpMenuEntryClient(t)
		client.EXPECT().Delete(ctx, "redmine", metav1.DeleteOptions{}).Return(assert.AnError)
		manager := &WarpMenuEntryManager{client: client}

		err := manager.RemoveWarpMenuEntry(ctx, doguResource.GetSimpleDoguName())

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete WarpMenuEntry")
	})
}

func notFoundErr(name string) error {
	return k8sErr.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "warpmenuentries"}, name)
}

func expectedOwnerReferences(doguResource *doguv2.Dogu) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind("Dogu")),
	}
}

func newExistingMatchingEntry(doguResource *doguv2.Dogu) *warpv1.WarpMenuEntry {
	return &warpv1.WarpMenuEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "redmine",
			OwnerReferences: expectedOwnerReferences(doguResource),
		},
		Spec: warpv1.WarpMenuEntrySpec{
			DisplayName: warpv1.DisplayName{DE: "Redmine", EN: "Redmine"},
			Category:    "Development Apps",
			Path:        "/redmine",
			Disabled:    false,
		},
	}
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

func newWarpDoguDescriptor() *cesappcore.Dogu {
	return &cesappcore.Dogu{
		Name:        "official/redmine",
		DisplayName: "Redmine",
		Category:    "Development Apps",
		Tags:        []string{"warp", "issue-tracker"},
	}
}

func newDoguDescriptorWithoutWarpTag() *cesappcore.Dogu {
	return &cesappcore.Dogu{
		Name:        "official/redmine",
		DisplayName: "Redmine",
		Category:    "Development Apps",
		Tags:        []string{"issue-tracker"},
	}
}
