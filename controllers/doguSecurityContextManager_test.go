package controllers

import (
	"fmt"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguSecurityContextManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		mgrSet := &util.ManagerSet{}
		mockEventRecorder := &mockEventRecorder{}

		// when
		doguSecurityContextManager := NewDoguSecurityContextManager(mgrSet, mockEventRecorder)

		// then
		require.NotNil(t, doguSecurityContextManager)
	})
}

func Test_doguSecurityContextManager_UpdateDeploymentWithSecurityContext(t *testing.T) {
	doguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
		Spec:       k8sv2.DoguSpec{Name: "official/ldap"},
	}
	doguDescriptor := &core.Dogu{Name: "official/ldap"}
	tests := []struct {
		name                string
		localDoguFetcherFn  func(t *testing.T) localDoguFetcher
		resourceUpserterFn  func(t *testing.T) resourceUpserter
		recorderFn          func(t *testing.T) eventRecorder
		securityValidatorFn func(t *testing.T) securityValidator
		wantErr             assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get local dogu descriptor",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mockDoguFetcher := newMockLocalDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(nil, assert.AnError)
				return mockDoguFetcher
			},
			recorderFn: func(t *testing.T) eventRecorder {
				mockEventRecorder := newMockEventRecorder(t)
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Getting local dogu descriptor...")
				return mockEventRecorder
			},
			securityValidatorFn: func(t *testing.T) securityValidator {
				mockSecurityValidator := newMockSecurityValidator(t)
				return mockSecurityValidator
			},
			resourceUpserterFn: func(t *testing.T) resourceUpserter {
				mockResourceUpserter := newMockResourceUpserter(t)
				return mockResourceUpserter
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "failed to get local descriptor for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail when validating security",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mockDoguFetcher := newMockLocalDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(doguDescriptor, nil)
				return mockDoguFetcher
			},
			recorderFn: func(t *testing.T) eventRecorder {
				mockEventRecorder := newMockEventRecorder(t)
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Getting local dogu descriptor...")
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Validating dogu security...")
				return mockEventRecorder
			},
			securityValidatorFn: func(t *testing.T) securityValidator {
				mockSecurityValidator := newMockSecurityValidator(t)
				mockSecurityValidator.EXPECT().ValidateSecurity(doguDescriptor, doguResource).Return(assert.AnError)
				return mockSecurityValidator
			},
			resourceUpserterFn: func(t *testing.T) resourceUpserter {
				mockResourceUpserter := newMockResourceUpserter(t)
				return mockResourceUpserter
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "validation of security context failed for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail to upsert deployment",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mockDoguFetcher := newMockLocalDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(doguDescriptor, nil)
				return mockDoguFetcher
			},
			recorderFn: func(t *testing.T) eventRecorder {
				mockEventRecorder := newMockEventRecorder(t)
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Getting local dogu descriptor...")
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Validating dogu security...")
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Upserting deployment...")
				return mockEventRecorder
			},
			securityValidatorFn: func(t *testing.T) securityValidator {
				mockSecurityValidator := newMockSecurityValidator(t)
				mockSecurityValidator.EXPECT().ValidateSecurity(doguDescriptor, doguResource).Return(nil)
				return mockSecurityValidator
			},
			resourceUpserterFn: func(t *testing.T) resourceUpserter {
				mockResourceUpserter := newMockResourceUpserter(t)
				mockResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, doguDescriptor, mock.AnythingOfType("func(*v1.Deployment)")).Return(nil, assert.AnError)
				return mockResourceUpserter
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "failed to upsert deployment with security context for dogu \"ldap\"", i)
			},
		},
		{
			name: "should succeed",
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				mockDoguFetcher := newMockLocalDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(doguDescriptor, nil)
				return mockDoguFetcher
			},
			recorderFn: func(t *testing.T) eventRecorder {
				mockEventRecorder := newMockEventRecorder(t)
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Getting local dogu descriptor...")
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Validating dogu security...")
				mockEventRecorder.EXPECT().Event(doguResource, "Normal", "SecurityContextChange", "Upserting deployment...")
				return mockEventRecorder
			},
			securityValidatorFn: func(t *testing.T) securityValidator {
				mockSecurityValidator := newMockSecurityValidator(t)
				mockSecurityValidator.EXPECT().ValidateSecurity(doguDescriptor, doguResource).Return(nil)
				return mockSecurityValidator
			},
			resourceUpserterFn: func(t *testing.T) resourceUpserter {
				mockResourceUpserter := newMockResourceUpserter(t)
				mockResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, doguDescriptor, mock.AnythingOfType("func(*v1.Deployment)")).Return(nil, nil)
				return mockResourceUpserter
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doguSecurityContextManager{
				localDoguFetcher:  tt.localDoguFetcherFn(t),
				resourceUpserter:  tt.resourceUpserterFn(t),
				securityValidator: tt.securityValidatorFn(t),
				recorder:          tt.recorderFn(t),
			}
			tt.wantErr(t, d.UpdateDeploymentWithSecurityContext(testCtx, doguResource), fmt.Sprintf("UpdateDeploymentWithSecurityContext(%v, %v)", testCtx, doguResource))
		})
	}
}
