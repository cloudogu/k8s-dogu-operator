package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/assert"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_getNetPolObjectKey(t *testing.T) {
	t.Run("should get objectKey for network Policy", func(t *testing.T) {
		netPol := &netv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testNetPol",
				Namespace: "testNetPolNamespace",
			},
		}

		result := getNetPolObjectKey(netPol)

		assert.Equal(t, "testNetPol", result.Name)
		assert.Equal(t, "testNetPolNamespace", result.Namespace)
	})
}

func Test_generateNetPolWithOwner(t *testing.T) {
	t.Run("should generate network policy with owner", func(t *testing.T) {
		doguResource := &k8sv2.Dogu{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Dogu",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "MyDogu",
				Namespace: "testNamespace",
				UID:       "DoguUid-1",
			},
		}

		spec := netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dogu.name": "redmine",
				},
			},
		}

		result := generateNetPolWithOwner("testNetPol", doguResource, spec)

		assert.Equal(t, "testNetPol", result.Name)
		assert.Equal(t, doguResource.Namespace, result.Namespace)
		assert.Equal(t, "ces", result.Labels["app"])
		assert.Equal(t, "MyDogu", result.Labels["dogu.name"])
		assert.Equal(t, spec, result.Spec)
		assert.Len(t, result.OwnerReferences, 1)
		assert.Equal(t, doguResource.APIVersion, result.OwnerReferences[0].APIVersion)
		assert.Equal(t, doguResource.Kind, result.OwnerReferences[0].Kind)
		assert.Equal(t, doguResource.Name, result.OwnerReferences[0].Name)
		assert.Equal(t, doguResource.UID, result.OwnerReferences[0].UID)
	})
}

func Test_generateIngressNetPol(t *testing.T) {
	t.Run("should generate ingress network policy", func(t *testing.T) {
		doguResource := &k8sv2.Dogu{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Dogu",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "MyDogu",
				Namespace: "testNamespace",
				UID:       "DoguUid-1",
			},
		}
		dogu := &core.Dogu{Name: "official/redmine"}

		result := generateIngressNetPol(doguResource, dogu)
		assert.Equal(t, "redmine-ingress", result.Name)
		assert.Equal(t, "redmine", result.Spec.PodSelector.MatchLabels["dogu.name"])
		assert.Len(t, result.Spec.PolicyTypes, 1)
		assert.Equal(t, netv1.PolicyTypeIngress, result.Spec.PolicyTypes[0])
		assert.Len(t, result.Spec.Ingress, 1)
		assert.Len(t, result.Spec.Ingress[0].From, 1)
		assert.Equal(t, "nginx-ingress", result.Spec.Ingress[0].From[0].PodSelector.MatchLabels["dogu.name"])
		assert.Equal(t, "testNamespace", result.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	})
}

func Test_generateDoguDepNetPol(t *testing.T) {
	t.Run("should generate dependency network policy", func(t *testing.T) {
		doguResource := &k8sv2.Dogu{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Dogu",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "MyDogu",
				Namespace: "testNamespace",
				UID:       "DoguUid-1",
			},
		}
		dogu := &core.Dogu{Name: "official/redmine"}

		result := generateDoguDepNetPol(doguResource, dogu, "cas")
		assert.Equal(t, "redmine-dependency-dogu-cas", result.Name)
		assert.Equal(t, "cas", result.Spec.PodSelector.MatchLabels["dogu.name"])
		assert.Len(t, result.Spec.PolicyTypes, 1)
		assert.Equal(t, netv1.PolicyTypeIngress, result.Spec.PolicyTypes[0])
		assert.Len(t, result.Spec.Ingress, 1)
		assert.Len(t, result.Spec.Ingress[0].From, 1)
		assert.Equal(t, "redmine", result.Spec.Ingress[0].From[0].PodSelector.MatchLabels["dogu.name"])
		assert.Equal(t, "testNamespace", result.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	})
}

func Test_generateDenyAllPolicy(t *testing.T) {
	t.Run("should generate deny-all network policy", func(t *testing.T) {
		doguResource := &k8sv2.Dogu{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Dogu",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "MyDogu",
				Namespace: "testNamespace",
				UID:       "DoguUid-1",
			},
		}
		dogu := &core.Dogu{Name: "official/redmine"}

		result := generateDenyAllPolicy(doguResource, dogu)
		assert.Equal(t, "redmine-deny-all", result.Name)
		assert.Equal(t, "redmine", result.Spec.PodSelector.MatchLabels["dogu.name"])
		assert.Len(t, result.Spec.PolicyTypes, 1)
		assert.Equal(t, netv1.PolicyTypeIngress, result.Spec.PolicyTypes[0])
		assert.Len(t, result.Spec.Ingress, 0)
	})
}
