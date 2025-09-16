package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewValidationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewValidationStep(
			newMockPremisesChecker(t),
			newMockLocalDoguFetcher(t),
			newMockResourceDoguFetcher(t),
			newMockDependencyValidator(t),
			newMockSecurityValidator(t),
			newMockDoguAdditionalMountsValidator(t),
		)

		assert.NotNil(t, step)
	})
}

func TestValidationStep_Run(t *testing.T) {
	type fields struct {
		premisesCheckerFn               func(t *testing.T) premisesChecker
		localDoguFetcherFn              func(t *testing.T) localDoguFetcher
		resourceDoguFetcherFn           func(t *testing.T) resourceDoguFetcher
		securityValidatorFn             func(t *testing.T) securityValidator
		doguAdditionalMountsValidatorFn func(t *testing.T) doguAdditionalMountsValidator
		dependencyValidatorFn           func(t *testing.T) dependencyValidator
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get remote dogu descriptor",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					return newMockPremisesChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil, nil, assert.AnError)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					return newMockSecurityValidator(t)
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					return newMockDependencyValidator(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should fail premise check for upgrade",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					mck := newMockPremisesChecker(t)
					mck.EXPECT().Check(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					},
						&core.Dogu{Version: "1.0.1"},
						&core.Dogu{Version: "1.0.0"},
					).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					return newMockSecurityValidator(t)
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					return newMockDependencyValidator(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed a premise check: %w", assert.AnError)),
		},
		{
			name: "should succeed premise check for upgrade",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					mck := newMockPremisesChecker(t)
					mck.EXPECT().Check(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					},
						&core.Dogu{Version: "1.0.1"},
						&core.Dogu{Version: "1.0.0"},
					).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					return newMockSecurityValidator(t)
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					return newMockDependencyValidator(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail dependency validation",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					return newMockPremisesChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					return newMockSecurityValidator(t)
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.0"}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail security validation",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					return newMockPremisesChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.0"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(assert.AnError)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.0"}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail additional mounts validation",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					return newMockPremisesChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.0"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					mck := newMockDoguAdditionalMountsValidator(t)
					mck.EXPECT().ValidateAdditionalMounts(testCtx, &core.Dogu{Version: "1.0.0"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(assert.AnError)
					return mck
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.0"}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should successfully run validation step",
			fields: fields{
				premisesCheckerFn: func(t *testing.T) premisesChecker {
					return newMockPremisesChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.0"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					mck := newMockDoguAdditionalMountsValidator(t)
					mck.EXPECT().ValidateAdditionalMounts(testCtx, &core.Dogu{Version: "1.0.0"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.0"}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &ValidationStep{
				premisesChecker:               tt.fields.premisesCheckerFn(t),
				localDoguFetcher:              tt.fields.localDoguFetcherFn(t),
				resourceDoguFetcher:           tt.fields.resourceDoguFetcherFn(t),
				securityValidator:             tt.fields.securityValidatorFn(t),
				doguAdditionalMountsValidator: tt.fields.doguAdditionalMountsValidatorFn(t),
				dependencyValidator:           tt.fields.dependencyValidatorFn(t),
			}
			assert.Equalf(t, tt.want, vs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
