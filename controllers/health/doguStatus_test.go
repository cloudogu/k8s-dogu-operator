package health

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewDoguStatusUpdater(t *testing.T) {
	// given
	doguClientMock := mocks.NewDoguInterface(t)
	recorderMock := mocks.NewEventRecorder(t)

	// when
	actual := NewDoguStatusUpdater(doguClientMock, recorderMock)

	// then
	assert.NotEmpty(t, actual)
}
