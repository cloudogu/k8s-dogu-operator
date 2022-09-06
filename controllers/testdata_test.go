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
var redmineBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/image-config.json
var imageConfigBytes []byte

//go:embed testdata/ldap-descriptor-cm.yaml
var ldapDescriptorBytes []byte

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte

//go:embed testdata/postgresql-dogu.json
var postgresqlBytes []byte

//go:embed testdata/simple-postfix-dogu.json
var postfixBytes []byte

//go:embed testdata/simple-nginx-dogu.json
var nginxBytes []byte

//go:embed testdata/simple-cas-dogu.json
var casBytes []byte

func readTestDataLdapCr(t *testing.T) *corev1.Dogu {
	t.Helper()

	ldapCr := &corev1.Dogu{}
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return ldapCr
}

func readTestDataLdapDogu(t *testing.T) *core.Dogu {
	t.Helper()

	return readTestDataDogu(t, ldapBytes)
}
func readTestDataLdapDescriptor(t *testing.T) *v1.ConfigMap {
	t.Helper()

	ldapDescriptor := &v1.ConfigMap{}
	err := yaml.Unmarshal(ldapDescriptorBytes, ldapDescriptor)
	if err != nil {
		t.Fatal(err.Error())
	}

	return ldapDescriptor
}

func readTestDataRedmineCr(t *testing.T) *corev1.Dogu {
	t.Helper()

	redmineCr := &corev1.Dogu{}
	err := yaml.Unmarshal(redmineCrBytes, redmineCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return redmineCr
}

func readTestDataRedmineDogu(t *testing.T) *core.Dogu {
	t.Helper()

	return readTestDataDogu(t, redmineBytes)
}

func readTestDataImageConfig(t *testing.T) *imagev1.ConfigFile {
	t.Helper()

	imageConfig := &imagev1.ConfigFile{}
	err := json.Unmarshal(imageConfigBytes, imageConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	return imageConfig
}

func readTestDataDogu(t *testing.T, doguBytes []byte) *core.Dogu {
	t.Helper()

	dogu := &core.Dogu{}
	err := json.Unmarshal(doguBytes, dogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return dogu
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
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
