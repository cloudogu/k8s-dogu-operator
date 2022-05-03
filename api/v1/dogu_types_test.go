package v1_test

import (
	"fmt"
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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
