package controllers

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/redmine-cr.yaml
var redmineCrBytes []byte

//go:embed testdata/redmine-dogu.json
var redmineDoguDescriptorBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/ldap-dogu.json
var ldapDoguDescriptorBytes []byte

//go:embed testdata/ldap-descriptor-cm.yaml
var ldapDoguDevelopmentMapBytes []byte

//go:embed testdata/ldap-dogu-upgrade.json
var ldapUpgradeDoguDescriptorBytes []byte

//go:embed testdata/image-config.json
var imageConfigBytes []byte

//go:embed testdata/ldap_expectedPodTemplate_support_on.yaml
var expectedPodTemplateSupportOnBytes []byte

//go:embed testdata/ldap_expectedPodTemplate_support_off.yaml
var expectedPodTemplateSupportOffBytes []byte

func readDoguCr(t *testing.T, bytes []byte) *corev1.Dogu {
	t.Helper()

	doguCr := &corev1.Dogu{}
	err := yaml.Unmarshal(bytes, doguCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return doguCr
}

func readImageConfig(t *testing.T, bytes []byte) *imagev1.ConfigFile {
	t.Helper()

	imageConfig := &imagev1.ConfigFile{}
	err := json.Unmarshal(bytes, imageConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	return imageConfig
}

func readDoguDescriptor(t *testing.T, doguBytes []byte) *core.Dogu {
	t.Helper()

	dogu := &core.Dogu{}
	err := json.Unmarshal(doguBytes, dogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return dogu
}

func readDoguDevelopmentMap(t *testing.T, devMapBytes []byte) *corev1.DevelopmentDoguMap {
	t.Helper()

	descriptorCM := &v1.ConfigMap{}
	err := yaml.Unmarshal(devMapBytes, descriptorCM)
	if err != nil {
		t.Fatal(err.Error())
	}

	doguDevMap := corev1.DevelopmentDoguMap(*descriptorCM)
	return &doguDevMap
}

func readLdapDoguExpectedPodTemplateSupportOn(t *testing.T) *v1.PodTemplateSpec {
	t.Helper()

	data := &v1.PodTemplateSpec{}
	err := yaml.Unmarshal(expectedPodTemplateSupportOnBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
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
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PodList",
	}, &v1.PodList{})

	return scheme
}
