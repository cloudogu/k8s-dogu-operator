package migration

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	networkingv1ac "k8s.io/client-go/applyconfigurations/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	migrationProducerLabelPrefix = "k8s.cloudogu.com/migration-"
	networkPolicyFieldManager    = "k8s-dogu-operator-migration-network-policy"
)

// ApplyNetworkPolicies applies one network policy per producer in the migration secret and returns the names of the newly created ones.
func ApplyNetworkPolicies(ctx context.Context, k8sClient client.Client, dogu *v3beta1.Dogu, migrationSecret *corev1.Secret) ([]string, error) {
	var created []string
	for _, policy := range newNetworkPolicies(dogu, migrationSecret) {
		key := client.ObjectKey{Namespace: *policy.Namespace, Name: *policy.Name}

		err := k8sClient.Get(ctx, key, &networkingv1.NetworkPolicy{})
		if client.IgnoreNotFound(err) != nil {
			return created, fmt.Errorf("failed to get migration network policy %s: %w", key, err)
		}

		if aErr := k8sClient.Apply(ctx, policy, client.FieldOwner(networkPolicyFieldManager), client.ForceOwnership); aErr != nil {
			return created, fmt.Errorf("failed to apply migration network policy %s: %w", key, aErr)
		}

		if apierrors.IsNotFound(err) {
			created = append(created, key.Name)
		}
	}

	return created, nil
}

// newNetworkPolicies builds one network policy per producer in the migration secret.
// Each policy admits only the dogu's migration job pods labeled for this producer to the v2 producer dogu.
func newNetworkPolicies(dogu *v3beta1.Dogu, migrationSecret *corev1.Secret) []*networkingv1ac.NetworkPolicyApplyConfiguration {
	var policies []*networkingv1ac.NetworkPolicyApplyConfiguration

	for _, producer := range producers(migrationSecret) {
		policies = append(policies, newNetworkPolicy(dogu, producer))
	}

	return policies
}

// producers returns the sorted service account producers, which prefix the migration secret keys.
func producers(migrationSecret *corev1.Secret) []string {
	found := make(map[string]bool)

	for key := range migrationSecret.Data {
		producer, _, _ := strings.Cut(key, credentialKeySeparator)
		found[producer] = true
	}

	return slices.Sorted(maps.Keys(found))
}

func newNetworkPolicy(dogu *v3beta1.Dogu, producer string) *networkingv1ac.NetworkPolicyApplyConfiguration {
	jobPods := metav1ac.LabelSelector().WithMatchLabels(map[string]string{
		migrationProducerLabelPrefix + producer: "true",
		v3beta1.DoguLabelName:                   dogu.Spec.Name,
	})

	return networkingv1ac.NetworkPolicy(dogu.Spec.Name+"-migration-"+producer, dogu.Namespace).
		WithLabels(migrationLabels(dogu)).
		WithOwnerReferences(metav1ac.OwnerReference().
			WithAPIVersion(v3beta1.GroupVersion.String()).
			WithKind(doguKind).
			WithName(dogu.Name).
			WithUID(dogu.UID).
			WithController(true).
			WithBlockOwnerDeletion(true)).
		WithSpec(networkingv1ac.NetworkPolicySpec().
			WithPodSelector(metav1ac.LabelSelector().WithMatchLabels(map[string]string{doguv2.DoguLabelName: producer})).
			WithPolicyTypes(networkingv1.PolicyTypeIngress).
			WithIngress(networkingv1ac.NetworkPolicyIngressRule().
				WithFrom(networkingv1ac.NetworkPolicyPeer().WithPodSelector(jobPods))))
}
