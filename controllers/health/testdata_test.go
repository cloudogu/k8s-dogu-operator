package health

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/redmine-cr.yaml
var redmineCrBytes []byte

//go:embed testdata/redmine-dogu.json
var redmineBytes []byte

//go:embed testdata/postgresql-cr.yaml
var postgresqlCrBytes []byte

//go:embed testdata/postgresql-dogu.json
var postgresqlBytes []byte

func readTestDataPostgresqlCr(t *testing.T) *corev1.Dogu {
	t.Helper()

	PostgresqlCr := &corev1.Dogu{}
	err := yaml.Unmarshal(postgresqlCrBytes, PostgresqlCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return PostgresqlCr
}

func readTestDataPostgresqlDogu(t *testing.T) *core.Dogu {
	t.Helper()

	PostgresqlDogu := &core.Dogu{}
	err := json.Unmarshal(postgresqlBytes, PostgresqlDogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return PostgresqlDogu
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

	redmineDogu := &core.Dogu{}
	err := json.Unmarshal(redmineBytes, redmineDogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return redmineDogu
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
		Kind:    "Pod",
	}, &v1.Pod{})

	return scheme
}
