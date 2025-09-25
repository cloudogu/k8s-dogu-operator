package main

import (
	_ "embed"
	"encoding/json"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

//go:embed testdata/additional-images-config.yaml
var additionalImagesCmBytes []byte

//go:embed testdata/global-config.yaml
var globalConfigBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/ldap-dogu-local-config-volume.json
var ldapDoguDescriptorWithLocalConfigVolumeBytes []byte

//go:embed testdata/image-config.json
var imageConfigBytes []byte

func readDoguCr(t ginkgo.GinkgoTInterface, bytes []byte) *doguv2.Dogu {
	t.Helper()

	doguCr := &doguv2.Dogu{}
	err := yaml.Unmarshal(bytes, doguCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return doguCr
}

func readImageConfig(t ginkgo.GinkgoTInterface, bytes []byte) *imagev1.ConfigFile {
	t.Helper()

	imageConfig := &imagev1.ConfigFile{}
	err := json.Unmarshal(bytes, imageConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	return imageConfig
}

func readConfigMap(t ginkgo.GinkgoTInterface, bytes []byte) *corev1.ConfigMap {
	t.Helper()

	configMap := &corev1.ConfigMap{}
	err := yaml.Unmarshal(bytes, configMap)
	if err != nil {
		t.Fatal(err.Error())
	}

	return configMap
}

func readDoguDescriptor(t ginkgo.GinkgoTInterface, doguBytes []byte) *core.Dogu {
	t.Helper()

	dogu := &core.Dogu{}
	err := json.Unmarshal(doguBytes, dogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return dogu
}
