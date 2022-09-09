package upgrade

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

	"github.com/coreos/etcd/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

var registryKeyNotFoundTestErr = fmt.Errorf("oh no: %w", &client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"})

const defaultNamespace = ""
const testNamespace = "test-namespace"

//go:embed testdata/redmine-cr.yaml
var redmineCrBytes []byte

//go:embed testdata/redmine-dogu.json
var redmineBytes []byte

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

func readTestDataLdapDogu(t *testing.T) *cesappcore.Dogu {
	t.Helper()

	return readTestDataDogu(t, ldapBytes)
}

func readTestDataRedmineCr(t *testing.T) *k8sv1.Dogu {
	t.Helper()

	redmineCr := &k8sv1.Dogu{}
	err := yaml.Unmarshal(redmineCrBytes, redmineCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return redmineCr
}

func readTestDataDogu(t *testing.T, doguBytes []byte) *cesappcore.Dogu {
	t.Helper()

	dogu := &cesappcore.Dogu{}
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
	}, &k8sv1.Dogu{})
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

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
}

func createReadyDeployment(doguName string) *appsv1.Deployment {
	return createDeployment(doguName, 1, 1)
}

func createDeployment(doguName string, replicas, replicasReady int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: defaultNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{Spec: v1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: replicas, ReadyReplicas: replicasReady},
	}
}
