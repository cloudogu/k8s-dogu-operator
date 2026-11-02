package migration

import (
	"fmt"
	"testing"

	authreg "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	exp "github.com/cloudogu/k8s-exposition-lib/api/v1"
	warp "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netpol "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	owner      = types.UID("owner-uid")
	otherOwner = types.UID("other-owner-uid")
)

func TestV2ResourceFinder_FindOwnedByKind(t *testing.T) {

	doguRef := corev1.ObjectReference{
		Name: "dogu",
		UID:  owner,
	}
	gvk := schema.GroupVersionKind{Group: "k8s.cloudogu.com", Version: "v1", Kind: "Exposition"}

	t.Run("Should return error for non-listable type", func(t *testing.T) {
		mockClient := NewMockListReader(t)
		mockClient.EXPECT().List(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("error"))

		_, err := NewV2ResourceFinder(mockClient).findOwnedByKind(t.Context(), gvk, doguRef)

		assert.Error(t, err)
	})

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	for _, test := range []struct {
		name      string
		gvk       schema.GroupVersionKind
		toFind    []client.Object
		notToFind []client.Object
	}{
		{
			name:   "Should return empty list when no matching objects are found",
			gvk:    schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			toFind: []client.Object{},
			notToFind: []client.Object{
				// wrong kind only
				newSecret("secret", true, true),
			},
		},
		{
			name:   "Should return empty list when no objects with matching labels are found",
			gvk:    schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			toFind: []client.Object{},
			notToFind: []client.Object{
				newConfigMap("cm-a", false, false),
				newConfigMap("cm-b", false, true),
			},
		},
		{
			name:   "Should return empty list when no objects with matching owner are found",
			gvk:    schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			toFind: []client.Object{},
			notToFind: []client.Object{
				newConfigMap("cm-a", false, false),
				newConfigMap("cm-b", true, false),
			},
		},
		{
			name: "Should return specified kind only",
			gvk:  schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			toFind: []client.Object{
				newConfigMap("cm-a", true, true),
			},
			notToFind: []client.Object{
				newSecret("secret", true, true),
			},
		},
		{
			name: "Should return objects with matching label and owner",
			gvk:  schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			toFind: []client.Object{
				newConfigMap("cm-match-a", true, true),
				newConfigMap("cm-match-b", true, true),
			},
			notToFind: []client.Object{
				newSecret("secret", true, true),
				newConfigMap("cm-no-label", false, true),
				newConfigMap("cm-no-owner", true, false),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			allObjects := append(test.toFind, test.notToFind...)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(allObjects...).Build()

			result, err := NewV2ResourceFinder(fakeClient).findOwnedByKind(t.Context(), test.gvk, doguRef)

			assert.NoError(t, err)
			compare(t, test.toFind, result)
		})
	}
}

func TestV2ResourceFinder_FindV2Resources(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, authreg.AddToScheme(scheme))
	require.NoError(t, exp.AddToScheme(scheme))
	require.NoError(t, warp.AddToScheme(scheme))
	require.NoError(t, netpol.AddToScheme(scheme))

	toFind := []client.Object{
		newConfigMap("cm", true, true),
		newSecret("secret", true, true),
		newService("service", true, true),
		newPersistentVolumeClaim("pvc", true, true),
		newDeployment("deployment", true, true),
		newAuthRegistration("authreg", true, true),
		newExposition("exposition", true, true),
		newWarpMenuEntry("warp-menu-entry-1", true, true),
		newWarpMenuEntry("warp-menu-entry-2", true, true),
		newNetworkPolicy("network-policy", true, true),
	}
	notToFind := []client.Object{
		newNetworkPolicy("netpol-no-owner", true, false),
		newWarpMenuEntry("warp-unrelated", false, false),
		newExposition("exposition-unrelated", false, false),
		newAuthRegistration("authreg-no-label", false, true),
		newDeployment("deployment-no-owner", true, false),
		newPersistentVolumeClaim("pvc-no-owner", true, false),
		newService("service-no-owner", true, false),
		newSecret("secret-no-owner", true, false),
		newConfigMap("cm-no-label", false, true),
		newConfigMap("cm-no-owner", true, false),
	}
	doguRef := corev1.ObjectReference{
		Name: "dogu",
		UID:  owner,
	}

	allObjects := append(toFind, notToFind...)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(allObjects...).Build()

	result, err := NewV2ResourceFinder(fakeClient).FindV2Resources(t.Context(), doguRef)

	assert.NoError(t, err)
	compare(t, toFind, result)
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

func newConfigMap(name string, withLabel bool, withOwner bool) client.Object {
	cm := &corev1.ConfigMap{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
	return setupObject(cm, name, withLabel, withOwner)
}

func newSecret(name string, withLabel bool, withOwner bool) client.Object {
	secret := &corev1.Secret{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return setupObject(secret, name, withLabel, withOwner)
}

func newService(name string, withLabel bool, withOwner bool) client.Object {
	service := &corev1.Service{
		Kind:       "Service",
		APIVersion: "v1",
	}
	return setupObject(service, name, withLabel, withOwner)
}

func newPersistentVolumeClaim(name string, withLabel bool, withOwner bool) client.Object {
	pvc := &corev1.PersistentVolumeClaim{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}
	return setupObject(pvc, name, withLabel, withOwner)
}

func newDeployment(name string, withLabel bool, withOwner bool) client.Object {
	deployment := &appsv1.Deployment{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}
	return setupObject(deployment, name, withLabel, withOwner)
}

func newAuthRegistration(name string, withLabel bool, withOwner bool) client.Object {
	authReg := &authreg.AuthRegistration{
		Kind:       "AuthRegistration",
		APIVersion: "k8s.cloudogu.com/v1",
	}
	return setupObject(authReg, name, withLabel, withOwner)
}

func newExposition(name string, withLabel bool, withOwner bool) client.Object {
	exposition := &exp.Exposition{
		Kind:       "Exposition",
		APIVersion: "k8s.cloudogu.com/v1",
	}
	return setupObject(exposition, name, withLabel, withOwner)
}

func newWarpMenuEntry(name string, withLabel bool, withOwner bool) client.Object {
	warpMenuEntry := &warp.WarpMenuEntry{
		Kind:       "WarpMenuEntry",
		APIVersion: "k8s.cloudogu.com/v1",
	}
	return setupObject(warpMenuEntry, name, withLabel, withOwner)
}

func newNetworkPolicy(name string, withLabel bool, withOwner bool) client.Object {
	networkPolicy := &netpol.NetworkPolicy{
		Kind:       "NetworkPolicy",
		APIVersion: "networking.k8s.io/v1",
	}
	return setupObject(networkPolicy, name, withLabel, withOwner)
}

func setupObject(obj client.Object, name string, withLabel bool, withOwner bool) client.Object {
	obj.SetName(name)
	obj.SetNamespace("ecosystem")
	if withLabel {
		obj.SetLabels(map[string]string{"dogu.name": "dogu"})
	}
	if withOwner {
		obj.SetOwnerReferences(append(obj.GetOwnerReferences(), metav1.OwnerReference{Name: "x", UID: owner}))
	}
	return obj
}
