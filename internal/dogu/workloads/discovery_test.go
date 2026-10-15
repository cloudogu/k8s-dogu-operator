package workloads

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace = "ecosystem"
	testDoguName  = "nexus"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	return scheme
}

func objectMeta(name, namespace, doguName string) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{Name: name, Namespace: namespace}
	if doguName != "" {
		meta.Labels = map[string]string{v3beta1.DoguLabelName: doguName}
	}
	return meta
}

func TestDiscovery_Workloads(t *testing.T) {
	t.Run("should return all workloads of the dogu", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(
			&appsv1.StatefulSet{ObjectMeta: objectMeta("nexus-nexus", testNamespace, testDoguName)},
			&appsv1.StatefulSet{ObjectMeta: objectMeta("nexus-nexus-postgresql", testNamespace, testDoguName)},
			&appsv1.Deployment{ObjectMeta: objectMeta("nexus-proxy", testNamespace, testDoguName)},
			&appsv1.DaemonSet{ObjectMeta: objectMeta("nexus-agent", testNamespace, testDoguName)},
		).Build()

		got, err := NewDiscovery(c).Workloads(context.Background(), testNamespace, testDoguName)

		require.NoError(t, err)
		require.Len(t, got.StatefulSets, 2)
		var statefulSetNames []string
		for _, statefulSet := range got.StatefulSets {
			statefulSetNames = append(statefulSetNames, statefulSet.Name)
		}
		assert.ElementsMatch(t, []string{"nexus-nexus", "nexus-nexus-postgresql"}, statefulSetNames)
		require.Len(t, got.Deployments, 1)
		assert.Equal(t, "nexus-proxy", got.Deployments[0].Name)
		require.Len(t, got.DaemonSets, 1)
		assert.Equal(t, "nexus-agent", got.DaemonSets[0].Name)
		assert.False(t, got.IsEmpty())
	})

	t.Run("should ignore workloads of other dogus, without label and in other namespaces", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(
			&appsv1.StatefulSet{ObjectMeta: objectMeta("nexus-nexus", testNamespace, testDoguName)},
			&appsv1.StatefulSet{ObjectMeta: objectMeta("scm", testNamespace, "scm")},
			&appsv1.Deployment{ObjectMeta: objectMeta("unlabeled", testNamespace, "")},
			&appsv1.DaemonSet{ObjectMeta: objectMeta("nexus-agent", "other", testDoguName)},
		).Build()

		got, err := NewDiscovery(c).Workloads(context.Background(), testNamespace, testDoguName)

		require.NoError(t, err)
		require.Len(t, got.StatefulSets, 1)
		assert.Equal(t, "nexus-nexus", got.StatefulSets[0].Name)
		assert.Empty(t, got.Deployments)
		assert.Empty(t, got.DaemonSets)
	})

	t.Run("should return no workloads if the dogu has none", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()

		got, err := NewDiscovery(c).Workloads(context.Background(), testNamespace, testDoguName)

		require.NoError(t, err)
		assert.True(t, got.IsEmpty())
	})

	listErrorTests := []struct {
		name    string
		failFor client.ObjectList
		wantErr string
	}{
		{name: "deployments", failFor: &appsv1.DeploymentList{}, wantErr: `failed to list deployments of dogu "nexus"`},
		{name: "statefulsets", failFor: &appsv1.StatefulSetList{}, wantErr: `failed to list statefulsets of dogu "nexus"`},
		{name: "daemonsets", failFor: &appsv1.DaemonSetList{}, wantErr: `failed to list daemonsets of dogu "nexus"`},
	}
	for _, tt := range listErrorTests {
		t.Run("should return error if listing "+tt.name+" fails", func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					if assert.ObjectsAreEqual(list, tt.failFor) {
						return assert.AnError
					}
					return client.List(ctx, list, opts...)
				},
			}).Build()

			_, err := NewDiscovery(c).Workloads(context.Background(), testNamespace, testDoguName)

			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
