package resource

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	depenendcyLabel    = "k8s.cloudogu.com/dependency"
	componentNameLabel = "k8s.cloudogu.com/component.name"
)

type NetPolType int

const (
	netPolTypeDogu NetPolType = iota
	netPolTypeComponent
	netPolTypeIngress
)

func (c NetPolType) String() string {
	return [...]string{"Dogu", "Component", "Ingress"}[c]
}

func generateDenyAllPolicy(doguResource *k8sv2.Dogu, dogu *core.Dogu) *netv1.NetworkPolicy {
	return generateNetPolWithOwner(
		fmt.Sprintf("%s-deny-all", dogu.GetSimpleName()),
		doguResource,
		netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					k8sv2.DoguLabelName: dogu.GetSimpleName(),
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	)
}

func getSelectors(doguResource *k8sv2.Dogu, coreDogu *core.Dogu, dependencyName string, netPolType NetPolType) (netPolName string, podSelector map[string]string, namespaceSelector map[string]string, matchLabels map[string]string) {
	switch netPolType {
	case netPolTypeDogu:
		netPolName = fmt.Sprintf("%s-dependency-dogu-%s", coreDogu.GetSimpleName(), dependencyName)
		podSelector = map[string]string{
			k8sv2.DoguLabelName: coreDogu.GetSimpleName(),
		}
		namespaceSelector = map[string]string{
			"kubernetes.io/metadata.name": doguResource.Namespace,
		}
		matchLabels = map[string]string{
			k8sv2.DoguLabelName: dependencyName,
		}
	case netPolTypeIngress:
		netPolName = fmt.Sprintf("%s-ingress", coreDogu.GetSimpleName())
		podSelector = map[string]string{
			k8sv2.DoguLabelName: k8sNginxIngressDoguName,
		}
		namespaceSelector = map[string]string{
			"kubernetes.io/metadata.name": doguResource.Namespace,
		}
		matchLabels = map[string]string{
			k8sv2.DoguLabelName: coreDogu.GetSimpleName(),
		}
	case netPolTypeComponent:
		netPolName = fmt.Sprintf("%s-dependency-component-%s", coreDogu.GetSimpleName(), dependencyName)
		podSelector = map[string]string{
			k8sv2.DoguLabelName: coreDogu.GetSimpleName(),
		}
		namespaceSelector = map[string]string{
			"kubernetes.io/metadata.name": doguResource.Namespace,
		}
		matchLabels = map[string]string{
			componentNameLabel: dependencyName,
		}
	}

	return
}

func generateNetPol(doguResource *k8sv2.Dogu, coreDogu *core.Dogu, dependencyName string, netPolType NetPolType) *netv1.NetworkPolicy {
	netPolName, podSelector, namespaceSelector, matchLabels := getSelectors(doguResource, coreDogu, dependencyName, netPolType)

	netPol := generateNetPolWithOwner(
		netPolName,
		doguResource,
		netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
			}, PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					From: []netv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: podSelector,
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: namespaceSelector,
							},
						},
					},
				},
			},
		},
	)

	netPol.Labels["k8s.cloudogu.com/dependency"] = dependencyName

	return netPol
}

func generateDoguDepNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, dependencyName string) *netv1.NetworkPolicy {
	return generateNetPol(doguResource, dogu, dependencyName, netPolTypeDogu)
}

func generateComponentDepNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, dependencyName string) *netv1.NetworkPolicy {
	return generateNetPol(doguResource, dogu, dependencyName, netPolTypeComponent)
}

func generateIngressNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu) *netv1.NetworkPolicy {
	return generateNetPol(doguResource, dogu, "", netPolTypeIngress)
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
