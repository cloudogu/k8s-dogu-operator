package controllers

import (
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	"testing"
)

func TestNewDoguSecurityContextManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		mgrSet := &util.ManagerSet{}

		// when
		doguSecurityContextManager := NewDoguSecurityContextManager(mgrSet)

		// then
		require.NotNil(t, doguSecurityContextManager)
	})
}

func Test_doguSecurityContextManager_UpdateDeploymentWithSecurityContext(t *testing.T) {
	// TODO all test cases have to be completed
	tests := []struct {
		name                  string
		resourceDoguFetcherFn func(t *testing.T) resourceDoguFetcher
		resourceUpserterFn    func(t *testing.T) resource.ResourceUpserter
		doguResource          *k8sv2.Dogu
		wantErr               assert.ErrorAssertionFunc
	}{
		{
			name: "failed to fetch dogu",
			resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
				doguResource := &k8sv2.Dogu{
					Spec: k8sv2.DoguSpec{
						Name: "ldap",
					},
				}
				mockDoguFetcher := newMockResourceDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchWithResource(testCtx, doguResource).Return(nil, nil, errors.New("failed to get development dogu map:"))
				return mockDoguFetcher
			},
			resourceUpserterFn: func(t *testing.T) resource.ResourceUpserter {
				mockResourceUpserter := newMockResourceUpserter(t)
				return mockResourceUpserter
			},
			doguResource: &k8sv2.Dogu{
				Spec: k8sv2.DoguSpec{
					Name: "ldap",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to fetch dogu ldap", i)
			},
		},

		{
			name: "failed to upsert dogu deployment",
			resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
				doguResource := &k8sv2.Dogu{
					Spec: k8sv2.DoguSpec{
						Name: "ldap",
					},
				}
				dogu := &core.Dogu{
					Name: "ldap",
				}
				mockDoguFetcher := newMockResourceDoguFetcher(t)
				mockDoguFetcher.EXPECT().FetchWithResource(testCtx, doguResource).Return(dogu, nil, nil)
				return mockDoguFetcher
			},
			resourceUpserterFn: func(t *testing.T) resource.ResourceUpserter {
				/*
					TODO
					doguResource := &k8sv2.Dogu{
						Spec: k8sv2.DoguSpec{
							Name: "ldap",
						},
					}
					dogu := &core.Dogu{
						Name: "ldap",
					}
					mockResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.AnythingOfType("func(*v1.Deployment)")).Return(nil, errors.New("failed to upsert deployment with security context"))
				*/
				doguResource := &k8sv2.Dogu{
					Spec: k8sv2.DoguSpec{
						Name: "ldap",
					},
				}
				dogu := &core.Dogu{
					Name: "ldap",
				}
				mockResourceUpserter := newMockResourceUpserter(t)

				fn := (func(*v1.Deployment))(nil)
				mockResourceUpserter.On("UpsertDoguDeployment", testCtx, doguResource, dogu, fn).Return(nil, errors.New("failed to upsert deployment with security context"))
				return mockResourceUpserter
			},
			doguResource: &k8sv2.Dogu{
				Spec: k8sv2.DoguSpec{
					Name: "ldap",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to fetch dogu ldap", i)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := doguSecurityContextManager{
				resourceDoguFetcher: tt.resourceDoguFetcherFn(t),
				resourceUpserter:    tt.resourceUpserterFn(t),
			}
			tt.wantErr(t, d.UpdateDeploymentWithSecurityContext(testCtx, tt.doguResource), fmt.Sprintf("UpdateDeploymentWithSecurityContext(%v, %v)", testCtx, tt.doguResource))
		})
	}
}
