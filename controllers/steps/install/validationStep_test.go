package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewValidationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewValidationStep(
			newMockDoguHealthChecker(t),
			newMockLocalDoguFetcher(t),
			newMockDependencyValidator(t),
			newMockSecurityValidator(t),
			newMockDoguAdditionalMountsValidator(t),
		)

		assert.NotNil(t, step)
	})
}

func TestValidationStep_Run(t *testing.T) {
	type fields struct {
		doguHealthCheckerFn             func(t *testing.T) doguHealthChecker
		localDoguFetcherFn              func(t *testing.T) localDoguFetcher
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
			name: "should fail to get dogu descriptor",
			fields: fields{
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					return newMockDoguHealthChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil, assert.AnError)
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
			want: steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor for %q: %w", "test", assert.AnError)),
		},
		{
			name: "should fail dependency validation",
			fields: fields{
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					return newMockDoguHealthChecker(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil)
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
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.1"}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail dependency health check",
			fields: fields{
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					mck := newMockDoguHealthChecker(t)
					mck.EXPECT().CheckDependenciesRecursive(testCtx, &core.Dogu{Version: "1.0.1"}, "").Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil)
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
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.1"}).Return(nil)
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
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					mck := newMockDoguHealthChecker(t)
					mck.EXPECT().CheckDependenciesRecursive(testCtx, &core.Dogu{Version: "1.0.1"}, "").Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.1"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(assert.AnError)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					return newMockDoguAdditionalMountsValidator(t)
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.1"}).Return(nil)
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
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					mck := newMockDoguHealthChecker(t)
					mck.EXPECT().CheckDependenciesRecursive(testCtx, &core.Dogu{Version: "1.0.1"}, "").Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.1"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					mck := newMockDoguAdditionalMountsValidator(t)
					mck.EXPECT().ValidateAdditionalMounts(testCtx, &core.Dogu{Version: "1.0.1"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(assert.AnError)
					return mck
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.1"}).Return(nil)
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
				doguHealthCheckerFn: func(t *testing.T) doguHealthChecker {
					mck := newMockDoguHealthChecker(t)
					mck.EXPECT().CheckDependenciesRecursive(testCtx, &core.Dogu{Version: "1.0.1"}, "").Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(&core.Dogu{Version: "1.0.1"}, nil)
					return mck
				},
				securityValidatorFn: func(t *testing.T) securityValidator {
					mck := newMockSecurityValidator(t)
					mck.EXPECT().ValidateSecurity(&core.Dogu{Version: "1.0.1"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				doguAdditionalMountsValidatorFn: func(t *testing.T) doguAdditionalMountsValidator {
					mck := newMockDoguAdditionalMountsValidator(t)
					mck.EXPECT().ValidateAdditionalMounts(testCtx, &core.Dogu{Version: "1.0.1"}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
				dependencyValidatorFn: func(t *testing.T) dependencyValidator {
					mck := newMockDependencyValidator(t)
					mck.EXPECT().ValidateDependencies(testCtx, &core.Dogu{Version: "1.0.1"}).Return(nil)
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
				doguHealthChecker:             tt.fields.doguHealthCheckerFn(t),
				localDoguFetcher:              tt.fields.localDoguFetcherFn(t),
				securityValidator:             tt.fields.securityValidatorFn(t),
				doguAdditionalMountsValidator: tt.fields.doguAdditionalMountsValidatorFn(t),
				dependencyValidator:           tt.fields.dependencyValidatorFn(t),
			}
			assert.Equalf(t, tt.want, vs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
