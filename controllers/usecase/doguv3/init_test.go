package doguv3

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWireDoguV3(t *testing.T) {
	version := core.Version{Raw: "1.2.3"}
	operatorConfig := &config.OperatorConfig{
		Namespace: "ecosystem",
		Version:   &version,
		DoguV3Registry: config.DoguRegistryData{
			Endpoint:  "https://v3.dogu.cloudogu.com",
			URLSchema: "default",
			Username:  "user",
			Password:  "pass",
		},
	}

	installUseCase, deleteUseCase, err := NewDoguV3UseCases(
		fake.NewClientBuilder().Build(),
		record.NewFakeRecorder(10),
		&rest.Config{},
		k8sfake.NewSimpleClientset(),
		operatorConfig,
	)

	require.NoError(t, err)
	assert.NotNil(t, installUseCase, "install-or-change use-case must be assembled")
	assert.NotNil(t, deleteUseCase, "delete use-case must be assembled")
}
