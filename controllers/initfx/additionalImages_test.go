package initfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var testCtx = context.Background()

func TestGetAdditionalImages(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		want              resource.AdditionalImages
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get config map",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				mck := newMockConfigMapInterface(t)
				mck.EXPECT().Get(mock.Anything, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{}).Return(nil, assert.AnError)
				return mck
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "should fail to image for key",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				mck := newMockConfigMapInterface(t)
				mck.EXPECT().Get(mock.Anything, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{}).Return(&v2.ConfigMap{
					Data: map[string]string{
						config.ChownInitImageConfigmapNameKey: "test",
					},
				}, nil)
				return mck
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "should fail to verify image tag",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				mck := newMockConfigMapInterface(t)
				mck.EXPECT().Get(mock.Anything, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{}).Return(&v2.ConfigMap{
					Data: map[string]string{
						config.ChownInitImageConfigmapNameKey:                     "test",
						config.ExporterImageConfigmapNameKey:                      "test",
						config.AdditionalMountsInitContainerImageConfigmapNameKey: "test",
					},
				}, nil)
				return mck
			},
			want:    resource.AdditionalImages{"additionalMountsInitContainerImage": "test", "chownInitImage": "test", "exporterImage": "test"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientMock := tt.configMapClientFn(t)
			got, err := GetAdditionalImages(clientMock)
			if !tt.wantErr(t, err, fmt.Sprintf("GetAdditionalImages(%v)", clientMock)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetAdditionalImages(%v)", clientMock)
		})
	}
}
