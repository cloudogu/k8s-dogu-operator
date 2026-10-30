package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	owner      = types.UID("owner-uid")
	otherOwner = types.UID("other-owner-uid")
)

func TestV2ResourceFinder_FindV2Resources(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

	doguRef := corev1.ObjectReference{
		Name: "dogu",
		UID:  owner,
	}

	for _, test := range []struct {
		name      string
		toFind    []client.Object
		notToFind []client.Object
	}{
		{
			name: "should find only objects with dogu.name label and matching owner",
			toFind: []client.Object{
				newConfigMap("cm-11", true, true),
			},
			notToFind: []client.Object{
				newConfigMap("cm-21", false, false),
				newConfigMap("cm-22", false, true),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			allObjects := append(test.toFind, test.notToFind...)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(allObjects...).Build()

			result, err := NewV2ResourceFinder(fakeClient).FindV2Resources(t.Context(), doguRef)

			assert.NoError(t, err)
			compare(t, test.toFind, result)
		})
	}
}

func compare(t *testing.T, expected, actual []client.Object) {
	for i, object := range expected {
		assert.Equal(t, object.GetObjectKind(), actual[i].GetObjectKind())
		assert.Equal(t, object.GetNamespace(), actual[i].GetNamespace())
		assert.Equal(t, object.GetName(), actual[i].GetName())
		assert.Equal(t, object.GetLabels(), actual[i].GetLabels())
		assert.Equal(t, object.GetOwnerReferences(), actual[i].GetOwnerReferences())
	}
}

func newConfigMap(name string, withLabel bool, withOwner bool) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		Kind:       "ConfigMap",
		APIVersion: "v1",
		Name:       name,
		Namespace:  "ecosystem",
	}
	if withLabel {
		cm.Labels = map[string]string{"dogu.name": "dogu"}
	}
	if withOwner {
		addOwnerReference(&cm.ObjectMeta, owner)
	}
	return cm
}

func addOwnerReference(meta *metav1.ObjectMeta, owner types.UID) {
	meta.OwnerReferences = append(meta.OwnerReferences, metav1.OwnerReference{Name: "x", UID: owner})
}

//{Version: "v1", Kind: "ConfigMap"},
//{Version: "v1", Kind: "Secret"},
//{Version: "v1", Kind: "Service"},
//{Version: "v1", Kind: "PersistentVolumeClaim"},
//{Group: "apps", Version: "v1", Kind: "Deployment"},
//{Group: "k8s.cloudogu.com", Version: "v1", Kind: "AuthRegistration"},
//{Group: "k8s.cloudogu.com", Version: "v1", Kind: "Exposition"},
//{Group: "k8s.cloudogu.com", Version: "v1", Kind: "WarpMenuEntry"},
//{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
