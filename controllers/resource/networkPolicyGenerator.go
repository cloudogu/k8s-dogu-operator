package resource

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func generateDenyAllPolicy(doguResource *k8sv2.Dogu, dogu *core.Dogu) *netv1.NetworkPolicy {
	return generateNetPolWithOwner(
		fmt.Sprintf("%s-deny-all", dogu.GetSimpleName()),
		doguResource,
		netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dogu.name": dogu.GetSimpleName(),
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	)

}

func generateDoguDepNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, dependencyName string) *netv1.NetworkPolicy {
	return generateNetPolWithOwner(fmt.Sprintf("%s-dependency-%s", dogu.GetSimpleName(), dependencyName), doguResource, netv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"dogu.name": dependencyName,
			},
		}, PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
		},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				From: []netv1.NetworkPolicyPeer{
					{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"dogu.name": dogu.GetSimpleName(),
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": doguResource.Namespace,
							},
						},
					},
				},
			},
		},
	})
}

func generateIngressNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu) *netv1.NetworkPolicy {
	return generateNetPolWithOwner(
		fmt.Sprintf("%s-ingress", dogu.GetSimpleName()),
		doguResource,
		netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dogu.name": dogu.GetSimpleName(),
				},
			}, PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					From: []netv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"dogu.name": k8sNginxIngressDoguName,
								},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": doguResource.Namespace,
								},
							},
						},
					},
				},
			},
		},
	)
}

func generateNetPolWithOwner(name string, parentDoguResource *k8sv2.Dogu, spec netv1.NetworkPolicySpec) *netv1.NetworkPolicy {
	return &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: parentDoguResource.APIVersion,
					Kind:       parentDoguResource.Kind,
					Name:       parentDoguResource.Name,
					UID:        parentDoguResource.UID,
				},
			},
			Namespace: parentDoguResource.Namespace,
			Labels:    GetAppLabel().Add(parentDoguResource.GetDoguNameLabel()),
		},
		Spec: spec,
	}
}

// GetObjectKey returns the object key with the actual name and namespace from the netPol resource
func getNetPolObjectKey(netPol *netv1.NetworkPolicy) client.ObjectKey {
	return client.ObjectKey{
		Namespace: netPol.Namespace,
		Name:      netPol.Name,
	}
}
