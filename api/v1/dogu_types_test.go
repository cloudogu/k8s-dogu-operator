package v1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/api/v1/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testDogu = &v1.Dogu{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "k8s.cloudogu.com/v1",
		Kind:       "Dogu",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "dogu",
		Namespace: "ecosystem",
	},
	Spec: v1.DoguSpec{
		Name:          "namespace/dogu",
		Version:       "1.2.3-4",
		UpgradeConfig: v1.UpgradeConfig{},
	},
	Status: v1.DoguStatus{Status: ""},
}

func TestDoguStatus_GetRequeueTime(t *testing.T) {
	tests := []struct {
		requeueCount        time.Duration
		expectedRequeueTime time.Duration
	}{
		{requeueCount: time.Second, expectedRequeueTime: time.Second * 2},
		{requeueCount: time.Second * 17, expectedRequeueTime: time.Second * 34},
		{requeueCount: time.Minute, expectedRequeueTime: time.Minute * 2},
		{requeueCount: time.Minute * 7, expectedRequeueTime: time.Minute * 14},
		{requeueCount: time.Minute * 45, expectedRequeueTime: time.Hour*1 + time.Minute*30},
		{requeueCount: time.Hour * 2, expectedRequeueTime: time.Hour * 4},
		{requeueCount: time.Hour * 3, expectedRequeueTime: time.Hour * 6},
		{requeueCount: time.Hour * 5, expectedRequeueTime: time.Hour * 6},
		{requeueCount: time.Hour * 100, expectedRequeueTime: time.Hour * 6},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("calculate next requeue time for current time %s", tt.requeueCount), func(t *testing.T) {
			// given
			ds := &v1.DoguStatus{
				RequeueTime: tt.requeueCount,
			}

			// when
			actualRequeueTime := ds.NextRequeue()

			// then
			assert.Equal(t, tt.expectedRequeueTime, actualRequeueTime)
		})
	}
}

func TestDoguStatus_ResetRequeueTime(t *testing.T) {
	t.Run("reset requeue time", func(t *testing.T) {
		// given
		ds := &v1.DoguStatus{
			RequeueTime: time.Hour * 3,
		}

		// when
		ds.ResetRequeueTime()

		// then
		assert.Equal(t, v1.RequeueTimeInitialRequeueTime, ds.RequeueTime)
	})
}

func TestDogu_GetSecretObjectKey(t *testing.T) {
	// given
	ds := &v1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myspecialdogu",
			Namespace: "testnamespace",
		},
	}

	// when
	key := ds.GetSecretObjectKey()

	// then
	assert.Equal(t, "myspecialdogu-secrets", key.Name)
	assert.Equal(t, "testnamespace", key.Namespace)
}

func Test_Dogu_ChangeState(t *testing.T) {
	ctx := context.TODO()

	t.Run("should set the dogu resource's status to upgrade", func(t *testing.T) {
		sut := &v1.Dogu{}
		myClient := new(mocks.Client)
		statusMock := new(mocks.StatusWriter)
		myClient.On("Status").Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(nil)

		// when
		err := sut.ChangeState(ctx, myClient, v1.DoguStatusUpgrading)

		// then
		require.NoError(t, err)
		assert.Equal(t, v1.DoguStatusUpgrading, sut.Status.Status)
		myClient.AssertExpectations(t)
		statusMock.AssertExpectations(t)
	})
	t.Run("should fail on client error", func(t *testing.T) {
		sut := &v1.Dogu{}
		myClient := new(mocks.Client)
		statusMock := new(mocks.StatusWriter)
		myClient.On("Status").Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(assert.AnError)

		// when
		err := sut.ChangeState(ctx, myClient, v1.DoguStatusUpgrading)

		// then
		require.ErrorIs(t, err, assert.AnError)
		myClient.AssertExpectations(t)
		statusMock.AssertExpectations(t)
	})
}

func TestDogu_GetObjectKey(t *testing.T) {
	actual := testDogu.GetObjectKey()

	expectedObjKey := client.ObjectKey{
		Namespace: "ecosystem",
		Name:      "dogu",
	}
	assert.Equal(t, expectedObjKey, actual)
}

func TestDogu_GetObjectMeta(t *testing.T) {
	actual := testDogu.GetObjectMeta()

	expectedObjKey := &metav1.ObjectMeta{
		Namespace: "ecosystem",
		Name:      "dogu",
	}
	assert.Equal(t, expectedObjKey, actual)
}

func TestDogu_GetDataVolumeName(t *testing.T) {
	actual := testDogu.GetDataVolumeName()

	assert.Equal(t, "dogu-data", actual)
}

func TestDogu_GetPrivateVolumeName(t *testing.T) {
	actual := testDogu.GetPrivateVolumeName()

	assert.Equal(t, "dogu-private", actual)
}
