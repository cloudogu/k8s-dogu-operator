package upgrade

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestNewRetroactiveServiceAccountStep(t *testing.T) {
	doguEvents := make(chan event.TypedGenericEvent[*doguv2.Dogu])
	doguClient := newMockDoguInterface(t)
	localDoguFetcher := newMockLocalDoguFetcher(t)
	step := NewRetroactiveServiceAccountStep(doguEvents, doguClient, localDoguFetcher)

	require.NotNil(t, step)
	assert.NotNil(t, step.doguEvents)
	assert.Same(t, doguClient, step.doguClient)
	assert.Same(t, localDoguFetcher, step.localDoguFetcher)
}

func TestRetroactiveServiceAccountStep_Run(t *testing.T) {
	type fields struct {
		doguClientFn       func(t *testing.T) doguInterface
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
	}
	tests := []struct {
		name           string
		fields         fields
		resource       *doguv2.Dogu
		expectedEvents []event.TypedGenericEvent[*doguv2.Dogu]
		want           steps.StepResult
	}{
		{
			name: "fail to list dogus",
			fields: fields{
				doguClientFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					return mck
				},
			},
			resource:       &doguv2.Dogu{},
			expectedEvents: nil,
			want:           steps.StepResult{Err: fmt.Errorf("list dogus for retroactive service accounts: %w", assert.AnError)},
		},
		{
			name: "fail to fetch multiple dogu descriptors",
			fields: fields{
				doguClientFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					doguList := &doguv2.DoguList{Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "dogu1"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "dogu2"}},
					}}
					mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu1"}}).
						Return(nil, assert.AnError)
					mck.EXPECT().FetchForResource(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu2"}}).
						Return(nil, assert.AnError)
					return mck
				},
			},
			resource:       &doguv2.Dogu{},
			expectedEvents: nil,
			want:           steps.StepResult{Err: fmt.Errorf("retrieve retroactive service accounts: %w", errors.Join(assert.AnError, assert.AnError))},
		},
		{
			name: "success",
			fields: fields{
				doguClientFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					doguList := &doguv2.DoguList{Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "dogu1"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "dogu2"}},
					}}
					mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu1"}}).
						Return(&core.Dogu{ServiceAccounts: []core.ServiceAccount{
							{Kind: "component", Type: "prometheus"},
							{Kind: "", Type: "not-dogu0"},
							{Kind: "dogu", Type: "not-dogu0"},
							{Kind: "dogu", Type: "dogu0"},
						}}, nil)
					mck.EXPECT().FetchForResource(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu2"}}).
						Return(&core.Dogu{ServiceAccounts: []core.ServiceAccount{
							{Kind: "", Type: "dogu0"},
							{Kind: "dogu", Type: "dogu0"},
						}}, nil)
					return mck
				},
			},
			resource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu0"}},
			expectedEvents: []event.TypedGenericEvent[*doguv2.Dogu]{
				{Object: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu1"}}},
				{Object: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu2"}}},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doguEvents := make(chan event.TypedGenericEvent[*doguv2.Dogu])
			defer close(doguEvents)

			r := &RetroactiveServiceAccountStep{
				doguEvents:       doguEvents,
				doguClient:       tt.fields.doguClientFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
			}

			go func() {
				var actualEvents []event.TypedGenericEvent[*doguv2.Dogu]
				for doguEvent := range doguEvents {
					actualEvents = append(actualEvents, doguEvent)
				}

				assert.ElementsMatch(t, tt.expectedEvents, actualEvents)
			}()

			assert.Equal(t, tt.want, r.Run(testCtx, tt.resource))
		})
	}
}
