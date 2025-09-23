package main

import (
	_ "embed"
	"encoding/json"
	"testing"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"sigs.k8s.io/yaml"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

//go:embed testdata/redmine-cr.yaml
var redmineCrBytes []byte

//go:embed testdata/redmine-dogu.json
var redmineDoguDescriptorBytes []byte

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/ldap-dogu-local-config-volume.json
var ldapDoguDescriptorWithLocalConfigVolumeBytes []byte

//go:embed testdata/image-config.json
var imageConfigBytes []byte

func readDoguCr(t *testing.T, bytes []byte) *doguv2.Dogu {
	t.Helper()

	doguCr := &doguv2.Dogu{}
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
