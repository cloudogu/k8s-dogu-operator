package migration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_newNetworkPolicies(t *testing.T) {
	t.Run("with producer credentials", func(t *testing.T) {
		dogu := v3Dogu("redmine")
		dogu.UID = "redmine-uid"
		secret := &corev1.Secret{Data: map[string][]byte{
			"postgresql_username": []byte("user"),
			"postgresql_password": []byte("secret"),
			"nexus_username":      []byte("user"),
		}}

		policies := newNetworkPolicies(dogu, secret)

		require.Len(t, policies, 2)
		assert.Equal(t, "redmine-migration-nexus", *policies[0].Name)
		assert.Equal(t, "redmine-migration-postgresql", *policies[1].Name)

		policy := policies[1]
		assert.Equal(t, testNamespace, *policy.Namespace)
		assert.Equal(t, map[string]string{
			"k8s.cloudogu.com/app":       "ces",
			"k8s.cloudogu.com/dogu.name": "redmine",
			"k8s.cloudogu.com/type":      "migration",
		}, policy.Labels)

		require.Len(t, policy.OwnerReferences, 1)
		assert.Equal(t, "Dogu", *policy.OwnerReferences[0].Kind)
		assert.Equal(t, "redmine", *policy.OwnerReferences[0].Name)
		assert.Equal(t, "redmine-uid", string(*policy.OwnerReferences[0].UID))

		assert.Equal(t, map[string]string{"dogu.name": "postgresql"}, policy.Spec.PodSelector.MatchLabels)
		assert.Equal(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}, policy.Spec.PolicyTypes)
		require.Len(t, policy.Spec.Ingress, 1)
		require.Len(t, policy.Spec.Ingress[0].From, 1)
		assert.Equal(t, map[string]string{
			"k8s.cloudogu.com/migration-postgresql": "true",
			"k8s.cloudogu.com/dogu.name":            "redmine",
		}, policy.Spec.Ingress[0].From[0].PodSelector.MatchLabels)
		assert.Empty(t, policy.Spec.Ingress[0].Ports)
	})

	t.Run("without credentials", func(t *testing.T) {
		policies := newNetworkPolicies(v3Dogu("redmine"), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "redmine-migration"}})

		assert.Empty(t, policies)
	})
}

func TestApplyNetworkPolicies(t *testing.T) {
	secret := &corev1.Secret{Data: map[string][]byte{"postgresql_username": []byte("user"), "nexus_username": []byte("user")}}
	existing := &networkingv1.NetworkPolicy{Name: "redmine-migration-nexus", Namespace: testNamespace}
	// The deduced type converter is needed, because the default one fails to apply network policies.
	cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithTypeConverters(managedfields.NewDeducedTypeConverter()).WithObjects(existing).Build()

	created, err := ApplyNetworkPolicies(context.Background(), cl, v3Dogu("redmine"), secret)

	require.NoError(t, err)
	assert.Equal(t, []string{"redmine-migration-postgresql"}, created)
	policy := &networkingv1.NetworkPolicy{}
	require.NoError(t, cl.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: "redmine-migration-postgresql"}, policy))
	assert.Equal(t, map[string]string{"dogu.name": "postgresql"}, policy.Spec.PodSelector.MatchLabels)
}
