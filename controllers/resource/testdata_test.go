package resource

import (
	_ "embed"
	"encoding/json"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapDoguResourceBytes []byte

//go:embed testdata/image-config.json
var imageConfBytes []byte

//go:embed testdata/ldap_expectedDeployment.yaml
var expectedDeploymentBytes []byte

//go:embed testdata/ldap_expectedDeployment_Development.yaml
var expectedDeploymentDevelopBytes []byte

//go:embed testdata/ldap_expectedDoguPVC.yaml
var expectedDoguPVCBytes []byte

//go:embed testdata/ldap_expectedPVC_withCustomSize.yaml
var expectedDoguPVCWithCustomSizeBytes []byte

//go:embed testdata/ldap_expectedService.yaml
var expectedServiceBytes []byte

//go:embed testdata/nginx-ingress-dogu.json
var nginxIngressBytes []byte

//go:embed testdata/nginx-ingress-cr.yaml
var nginxIngressDoguResourceBytes []byte

//go:embed testdata/nginx-ingress-only_expectedLoadbalancer.yaml
var expectedNginxIngressOnlyLoadBalancer []byte

//go:embed testdata/nginx-ingress-scm_expectedLoadbalancer.yaml
var expectedNginxIngressSCMLoadBalancer []byte

//go:embed testdata/cas-dogu.json
var casBytes []byte

func readCasDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(casBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(ldapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguResource(t *testing.T) *k8sv2.Dogu {
	t.Helper()

	data := &k8sv2.Dogu{}
	err := yaml.Unmarshal(ldapDoguResourceBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguImageConfig(t *testing.T) *imagev1.ConfigFile {
	t.Helper()

	data := &imagev1.ConfigFile{}
	err := json.Unmarshal(imageConfBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedDeployment(t *testing.T) *appsv1.Deployment {
	t.Helper()

	data := &appsv1.Deployment{}
	err := yaml.Unmarshal(expectedDeploymentBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedDevelopDeployment(t *testing.T) *appsv1.Deployment {
	t.Helper()

	data := &appsv1.Deployment{}
	err := yaml.Unmarshal(expectedDeploymentDevelopBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedDoguPVC(t *testing.T) *v1.PersistentVolumeClaim {
	t.Helper()

	data := &v1.PersistentVolumeClaim{}
	err := yaml.Unmarshal(expectedDoguPVCBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedDoguPVCWithCustomSize(t *testing.T) *v1.PersistentVolumeClaim {
	t.Helper()

	data := &v1.PersistentVolumeClaim{}
	err := yaml.Unmarshal(expectedDoguPVCWithCustomSizeBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedService(t *testing.T) *v1.Service {
	t.Helper()

	data := &v1.Service{}
	err := yaml.Unmarshal(expectedServiceBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readNginxIngressDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(nginxIngressBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readNginxIngressDoguResource(t *testing.T) *k8sv2.Dogu {
	t.Helper()

	data := &k8sv2.Dogu{}
	err := yaml.Unmarshal(nginxIngressDoguResourceBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readNginxIngressOnlyExpectedLoadBalancer(t *testing.T) *v1.Service {
	t.Helper()

	data := &v1.Service{}
	err := yaml.Unmarshal(expectedNginxIngressOnlyLoadBalancer, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readNginxIngressSCMExpectedLoadBalancer(t *testing.T) *v1.Service {
	t.Helper()

	data := &v1.Service{}
	err := yaml.Unmarshal(expectedNginxIngressSCMLoadBalancer, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "Dogu",
	}, &k8sv2.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &appsv1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &v1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &v1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &v1.PersistentVolumeClaim{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &v1.ConfigMap{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	}, &eventV1.Event{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}, &v1.Pod{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PodList",
	}, &v1.PodList{})

	return scheme
}
