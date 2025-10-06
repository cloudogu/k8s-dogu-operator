package health

import (
	_ "embed"
	"encoding/json"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

//go:embed testdata/redmine-dogu.json
var redmineBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/postgresql-dogu.json
var postgresqlBytes []byte

//go:embed testdata/mandatory1-dogu.json
var mandatory1Bytes []byte

//go:embed testdata/mandatory2-dogu.json
var mandatory2Bytes []byte

//go:embed testdata/optional1-dogu.json
var optional1Bytes []byte

//go:embed testdata/optional2-dogu.json
var optional2Bytes []byte

//go:embed testdata/ignore-nginx-registrator.json
var ignoreNginxRegistratorBytes []byte

func readTestDataLdapCr(t *testing.T) *doguv2.Dogu {
	t.Helper()

	ldapCr := &doguv2.Dogu{}
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return ldapCr
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
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogu",
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
		Kind:    "Pod",
	}, &v1.Pod{})

	return scheme
}
