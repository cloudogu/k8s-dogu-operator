package cesregistry

import (
	_ "embed"
	"encoding/json"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"

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

//go:embed testdata/redmine-descriptor-cm.yaml
var redmineCrConfigMapBytes []byte

func readTestDataRedmineCr(t *testing.T) *k8sv2.Dogu {
	t.Helper()

	redmineCr := &k8sv2.Dogu{}
	err := yaml.Unmarshal(redmineCrBytes, redmineCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return redmineCr
}

func readDoguDescriptorConfigMap(t *testing.T, descriptorBytes []byte) *k8sv2.DevelopmentDoguMap {
	t.Helper()

	descriptorCM := &v1.ConfigMap{}
	err := yaml.Unmarshal(descriptorBytes, descriptorCM)
	if err != nil {
		t.Fatal(err.Error())
	}

	doguDevMap := k8sv2.DevelopmentDoguMap(*descriptorCM)
	return &doguDevMap
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
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogu",
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

	return scheme
}
