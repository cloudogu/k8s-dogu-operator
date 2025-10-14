package resource

import (
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
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

func generateDenyAllPolicy(doguResource *k8sv2.Dogu, dogu *core.Dogu, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
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
		scheme,
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
			k8sCesGatewayComponentLabel: k8sCesGatewayComponentName,
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

func generateNetPol(doguResource *k8sv2.Dogu, coreDogu *core.Dogu, dependencyName string, netPolType NetPolType, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
	netPolName, podSelector, namespaceSelector, matchLabels := getSelectors(doguResource, coreDogu, dependencyName, netPolType)

	netPol, err := generateNetPolWithOwner(
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
		scheme,
	)

	if netPol != nil {
		netPol.Labels["k8s.cloudogu.com/dependency"] = dependencyName
	}

	return netPol, err
}

func generateDoguDepNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, dependencyName string, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
	return generateNetPol(doguResource, dogu, dependencyName, netPolTypeDogu, scheme)
}

func generateComponentDepNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, dependencyName string, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
	return generateNetPol(doguResource, dogu, dependencyName, netPolTypeComponent, scheme)
}

func generateIngressNetPol(doguResource *k8sv2.Dogu, dogu *core.Dogu, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
	return generateNetPol(doguResource, dogu, "", netPolTypeIngress, scheme)
}

func generateNetPolWithOwner(name string, parentDoguResource *k8sv2.Dogu, spec netv1.NetworkPolicySpec, scheme *runtime.Scheme) (*netv1.NetworkPolicy, error) {
	netpol := &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: parentDoguResource.Namespace,
			Labels:    GetAppLabel().Add(parentDoguResource.GetDoguNameLabel()),
		},
		Spec: spec,
	}

	err := ctrl.SetControllerReference(parentDoguResource, netpol, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to set owner reference on network policy for dogu %s: %v", name, err)
	}

	return netpol, nil
}

// GetObjectKey returns the object key with the actual name and namespace from the netPol resource
func getNetPolObjectKey(netPol *netv1.NetworkPolicy) client.ObjectKey {
	return client.ObjectKey{
		Namespace: netPol.Namespace,
		Name:      netPol.Name,
	}
}
