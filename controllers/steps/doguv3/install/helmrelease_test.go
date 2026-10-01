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
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(doguMetadataValues, true, nil)
		valueAsmMock.EXPECT().Assemble(testCtx, inputDogu, doguMetadataValues).Return(assembledDoguValues, nil)

		// when
		actual, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.NoError(t, err)
		expectedVal := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		expectedBytes, err := json.Marshal(expectedVal)
		expectedJson := v1.JSON{Raw: expectedBytes}
		assert.Equal(t, expectedJson, *actual)
	})

	t.Run("should fail on dogu metadata value service retrieval", func(t *testing.T) {
		// given
		inputDogu := &v3beta1.Dogu{}
		valueAsmMock := newMockDoguPatchAssembler(t)
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(nil, false, assert.AnError)

		// when
		_, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to retrieve dogu metadata values: ")
	})

	t.Run("should fail on dogu value assembly", func(t *testing.T) {
		// given
		inputDogu := &v3beta1.Dogu{}
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(doguMetadataValues, true, nil)
		valueAsmMock := newMockDoguPatchAssembler(t)
		valueAsmMock.EXPECT().Assemble(testCtx, inputDogu, doguMetadataValues).Return(nil, assert.AnError)

		// when
		_, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to assemble dogu values: ")
	})
}
