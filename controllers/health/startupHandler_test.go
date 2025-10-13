package health

import (
	"sync"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestNewStartupHandler(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().Add(mock.Anything).Return(nil)

		// when
		handler, err := NewStartupHandler(managerMock, doguInterfaceMock, make(chan<- event.TypedGenericEvent[*v2.Dogu]))

		// then
		assert.Same(t, doguInterfaceMock, handler.doguInterface)
		assert.NoError(t, err)
	})
	t.Run("should fail to add handler", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().Add(mock.Anything).Return(assert.AnError)

		// when
		_, err := NewStartupHandler(managerMock, doguInterfaceMock, make(chan<- event.TypedGenericEvent[*v2.Dogu]))

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestStartupHandler_Start(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}

		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)

		doguEvents := make(chan event.TypedGenericEvent[*v2.Dogu])
		sut := StartupHandler{doguInterface: doguInterfaceMock, doguEvents: doguEvents}

		var wg sync.WaitGroup
		wg.Go(func() {
			expectedEvents := []event.TypedGenericEvent[*v2.Dogu]{
				{Object: casDogu}, {Object: ldapDogu},
			}
			for i, want := range expectedEvents {
				select {
				case got := <-doguEvents:
					assert.Equalf(t, want, got, "mismatch at index %d", i)
				case <-time.After(1 * time.Second):
					t.Errorf("timed out waiting for event %d; wanted %#v", i, want)
					return
				}
			}
		})

		// when
		err := sut.Start(testCtx)

		// then
		wg.Wait()
		require.NoError(t, err)
	})

	t.Run("should return error on dogu list error", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)

		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)

		sut := StartupHandler{doguInterface: doguInterfaceMock, doguEvents: nil}

		// when
		err := sut.Start(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
