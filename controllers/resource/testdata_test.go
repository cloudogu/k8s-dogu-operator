package resource

import (
	_ "embed"
	"encoding/json"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
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

//go:embed testdata/ldap_expectedDeployment_withCustomValues.yaml
var expectedCustomDeploymentBytes []byte

//go:embed testdata/ldap_expectedDeployment_Development.yaml
var expectedDeploymentDevelopBytes []byte

//go:embed testdata/ldap_expectedPVC.yaml
var expectedPVCBytes []byte

//go:embed testdata/ldap_expectedSecret.yaml
var expectedSecretBytes []byte

//go:embed testdata/ldap_expectedService.yaml
var expectedServiceBytes []byte

//go:embed testdata/ldap_expectedExposedServices.yaml
var expectedExposedServicesBytes []byte

func readLdapDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(ldapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguResource(t *testing.T) *corev1.Dogu {
	t.Helper()

	data := &corev1.Dogu{}
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

func readLdapDoguExpectedCustomDeployment(t *testing.T) *appsv1.Deployment {
	t.Helper()

	data := &appsv1.Deployment{}
	err := yaml.Unmarshal(expectedCustomDeploymentBytes, data)
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

func readLdapDoguExpectedPVC(t *testing.T) *v1.PersistentVolumeClaim {
	t.Helper()

	data := &v1.PersistentVolumeClaim{}
	err := yaml.Unmarshal(expectedPVCBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguExpectedSecret(t *testing.T) *v1.Secret {
	t.Helper()

	data := &v1.Secret{}
	err := yaml.Unmarshal(expectedSecretBytes, data)
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

func readLdapDoguExpectedExposedServices(t *testing.T) []*v1.Service {
	t.Helper()

	data := &[]*v1.Service{}
	err := yaml.Unmarshal(expectedExposedServicesBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return *data
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &corev1.Dogu{})
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

	return scheme
}
