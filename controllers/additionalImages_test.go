package controllers

import (
	"fmt"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

func Test_additionalImageGetter_ImageForKey(t *testing.T) {
	t.Run("should fail on non-existing configmap", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		// given
		sut := newAdditionalImageGetter(fakeClient, testNamespace)

		// when
		_, err := sut.imageForKey(testCtx, config.ChownInitImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while getting configmap 'k8s-dogu-operator-additional-images':")
	})
	t.Run("should fail on missing configmap key", func(t *testing.T) {
		// given
		invalidCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: testNamespace,
			},
		}
		fakeClient := fake.NewSimpleClientset(invalidCM)
		sut := newAdditionalImageGetter(fakeClient, testNamespace)

		// when
		_, err := sut.imageForKey(testCtx, config.ChownInitImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "configmap 'k8s-dogu-operator-additional-images' must not contain empty image name for key chownInitImage")
	})
	t.Run("should fail on invalid image tag", func(t *testing.T) {
		// given
		invalidCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: testNamespace,
			},
			Data: map[string]string{config.ChownInitImageConfigmapNameKey: "busybox:::::123"},
		}
		fakeClient := fake.NewSimpleClientset(invalidCM)
		sut := newAdditionalImageGetter(fakeClient, testNamespace)

		// when
		_, err := sut.imageForKey(testCtx, config.ChownInitImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "configmap 'k8s-dogu-operator-additional-images' contains an invalid image tag: image tag 'busybox:::::123' seems invalid")
	})
	t.Run("should succeed on valid configmap", func(t *testing.T) {
		// given
		validCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: testNamespace,
			},
			Data: map[string]string{config.ChownInitImageConfigmapNameKey: "busybox:123"},
		}
		fakeClient := fake.NewSimpleClientset(validCM)
		sut := newAdditionalImageGetter(fakeClient, testNamespace)

		// when
		actual, err := sut.imageForKey(testCtx, config.ChownInitImageConfigmapNameKey)

		// then
		require.NoError(t, err)
		assert.Equal(t, "busybox:123", actual)
	})
}

func Test_verifyImageTag(t *testing.T) {
	tests := []struct {
		name     string
		imageTag string
		wantErr  assert.ErrorAssertionFunc
	}{
		{"valid simple w/o tag", "repo/image", assert.NoError},
		{"valid simple with tag", "repo/image:latest", assert.NoError},
		{"valid simple with version", "repo/image:v1.2.3", assert.NoError},
		{"valid inline-dashed simple with version", "repo/image-a:v1.2.3", assert.NoError},
		{"valid inline-underscore simple with version", "repo/image_a:v1.2.3", assert.NoError},
		{"valid double-inline-underscore simple with version", "repo/image__a:v1.2.3", assert.NoError},
		{"valid host w/o tag", "host.com/repo/image", assert.NoError},
		{"valid host with tag", "host.com/repo/image:latest", assert.NoError},
		{"valid host with version", "host.com/repo/image:v1.2.3", assert.NoError},
		{"valid inline-dashed host with version", "host.com/repo/image-a:v1.2.3", assert.NoError},
		{"valid host/port w/o tag", "host:8080/repo/image", assert.NoError},
		{"valid host/port with tag", "host:8080/repo/image:latest", assert.NoError},
		{"valid host/port with version", "host:8080/repo/image:v1.2.3", assert.NoError},
		{"valid inline-dashed host/port with version", "host:8080/repo/image-a:v1.2.3", assert.NoError},
		{"valid tag length", "host:8080/repo/image-a:superlongtagomgwhatisgoingonherethistagiswaylongerthaniexpectedbutweallknowthatatagmayconsistofupto128charachtersohwatchherewego", assert.NoError},

		{"invalid ending separator", "repo/image_", assert.Error},
		{"invalid ending separator", "repo/image-", assert.Error},
		{"invalid ending separator", "repo/image.", assert.Error},
		{"invalid ending separator", "repo/image_:v1.2.3", assert.Error},
		{"invalid ending separator", "repo/image-:v1.2.3", assert.Error},
		{"invalid ending separator", "repo/image.:v1.2.3", assert.Error},
		{"invalid ending separator", "repo/image_:latest", assert.Error},
		{"invalid ending separator", "repo/image-:latest", assert.Error},
		{"invalid ending separator", "repo/image.:latest", assert.Error},
		{"invalid ending separator", "host.com/repo/image_", assert.Error},
		{"invalid ending separator", "host.com/repo/image-:v1.2.3", assert.Error},
		{"invalid ending separator", "host.com/repo/image.:v1.2.3", assert.Error},
		{"invalid ending separator", "host.com/repo/image_:latest", assert.Error},
		{"invalid ending separator", "host.com/repo/image_:v1.2.3", assert.Error},
		{"invalid ending separator", "host:8080/repo/image_", assert.Error},
		{"invalid ending separator", "host:8080/repo/image-", assert.Error},
		{"invalid ending separator", "host:8080/repo/image.", assert.Error},
		{"invalid ending separator", "host:8080/repo/image_:latest", assert.Error},
		{"invalid ending separator", "host:8080/repo/image-:latest", assert.Error},
		{"invalid ending separator", "host:8080/repo/image.:latest", assert.Error},
		{"invalid ending separator", "host:8080/repo/image_:v1.2.3", assert.Error},
		{"invalid ending separator", "host:8080/repo/image-:v1.2.3", assert.Error},
		{"invalid ending separator", "host:8080/repo/image.:v1.2.3", assert.Error},

		{"invalid uppercase", "repo/Image", assert.Error},
		{"invalid uppercase", "repo/Image:v1.2.3", assert.Error},
		{"invalid uppercase", "repo/Image:latest", assert.Error},
		{"invalid uppercase", "host.com/repo/Image", assert.Error},
		{"invalid uppercase", "host.com/repo/Image:v1.2.3", assert.Error},
		{"invalid uppercase", "host.com/repo/Image:latest", assert.Error},
		{"invalid uppercase", "host:8080/repo/Image", assert.Error},
		{"invalid uppercase", "host:8080/repo/Image:latest", assert.Error},
		{"invalid uppercase", "host:8080/repo/Image:v1.2.3", assert.Error},

		{"invalid hostname length", "superlongtagomgwhatisgoingonherethistagiswaylongerthaniexpectedbutweallknowthatatagmayconsistofupto128charachtersohwatchherewegox:8080/repo/image:v1.2.3", assert.Error},

		{"invalid tag length", "repo/image:superlongtagomgwhatisgoingonherethistagiswaylongerthaniexpectedbutweallknowthatatagmayconsistofupto128charachtersohwatchherewegox", assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, verifyImageTag(tt.imageTag), fmt.Sprintf("verifyImageTag(%v)", tt.imageTag))
		})
	}
}
