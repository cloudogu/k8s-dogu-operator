package install

import (
	"encoding/json"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func Test_combineValues(t *testing.T) {
	t.Run("should combine values from without error", func(t *testing.T) {
		// given
		inputDogu := &v3beta1.Dogu{}
		assembledDoguValues := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		valueAsmMock := newMockDoguPatchAssembler(t)
		var patchtpl []byte
		valueAsmMock.EXPECT().Assemble(testCtx, inputDogu, patchtpl).Return(assembledDoguValues, nil)

		// when
		actual, err := combineValues(testCtx, inputDogu, valueAsmMock)

		// then
		require.NoError(t, err)
		expectedVal := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		expectedBytes, err := json.Marshal(expectedVal)
		expectedJson := v1.JSON{Raw: expectedBytes}
		assert.Equal(t, expectedJson, *actual)
	})
}
