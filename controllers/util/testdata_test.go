package util

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapDoguResourceBytes []byte

func readLdapDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(ldapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguResource(t *testing.T) *doguv2.Dogu {
	t.Helper()

	data := &doguv2.Dogu{}
	err := yaml.Unmarshal(ldapDoguResourceBytes, data)
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
	}, &doguv2.Dogu{})
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
