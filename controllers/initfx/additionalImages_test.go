package initfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func TestGetAdditionalImages(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) configMapInterface
		want              resource.AdditionalImages
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail on getting config map",
			configMapClientFn: func(t *testing.T) configMapInterface {
				mck := newMockConfigMapInterface(t)
				mck.EXPECT().Get(testCtx, "k8s-dogu-operator-additional-images", metav1.GetOptions{}).Return(nil, assert.AnError)
				return mck
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAdditionalImages(tt.configMapClientFn(t))
			if !tt.wantErr(t, err, fmt.Sprintf("GetAdditionalImages(%v)", tt.configMapClientFn(t))) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetAdditionalImages(%v)", tt.configMapClientFn(t))
		})
	}
}
