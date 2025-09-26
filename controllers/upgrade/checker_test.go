package upgrade

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker(newMockLocalDoguFetcher(t))
	assert.NotEmpty(t, c)
}

func Test_checker_IsUpgrade(t *testing.T) {
	tests := []struct {
		name               string
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		doguResource       *doguv2.Dogu
		want               bool
		wantErr            assert.ErrorAssertionFunc
	}{
		{
			name: "fail to fetch installed",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mck := newMockLocalDoguFetcher(t)
				mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to fetch dogu when checking for upgrade", i)
			},
		},
		{
			name: "fail to parse dogu resource version",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mck := newMockLocalDoguFetcher(t)
				mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, nil)
				return mck
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: doguv2.DoguSpec{Version: "invalid"}},
			want:         false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to parse desired dogu version", i)
			},
		},
		{
			name: "fail to parse dogu descriptor version",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mck := newMockLocalDoguFetcher(t)
				mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "invalid"}, nil)
				return mck
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: doguv2.DoguSpec{Name: "test", Version: "1.2.3-4"}},
			want:         false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to parse installed dogu version", i)
			},
		},
		{
			name: "should upgrade if desired version is newer than installed version",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mck := newMockLocalDoguFetcher(t)
				mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.2.3-4"}, nil)
				return mck
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: doguv2.DoguSpec{Name: "test", Version: "1.2.3-5"}},
			want:         true,
			wantErr:      assert.NoError,
		},
		{
			name: "should not upgrade if desired version is older than installed version",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mck := newMockLocalDoguFetcher(t)
				mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.2.3-5"}, nil)
				return mck
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: doguv2.DoguSpec{Name: "test", Version: "1.2.3-4"}},
			want:         false,
			wantErr:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &checker{
				localDoguFetcher: tt.localDoguFetcherFn(t),
			}
			got, err := c.IsUpgrade(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("IsUpgrade(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsUpgrade(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
