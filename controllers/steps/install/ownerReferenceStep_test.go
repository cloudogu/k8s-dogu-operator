package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewOwnerReferenceStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewOwnerReferenceStep(newMockOwnerReferenceSetter(t), nil)

		assert.NotNil(t, step)
	})
}

func TestOwnerReferenceStep_Run(t *testing.T) {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "Dogu",
	}, &v2.Dogu{})

	tests := []struct {
		name                   string
		ownerReferenceSetterFn func(t *testing.T) ownerReferenceSetter
		scheme                 *runtime.Scheme
		doguResource           *v2.Dogu
		want                   steps.StepResult
	}{
		{
			name:   "should fail to set owner reference",
			scheme: scheme,
			ownerReferenceSetterFn: func(t *testing.T) ownerReferenceSetter {
				mck := newMockOwnerReferenceSetter(t)
				mck.EXPECT().SetOwnerReference(testCtx, cescommons.SimpleName("test"), []metav1.OwnerReference{
					{
						Name:               "test",
						Kind:               "Dogu",
						APIVersion:         "k8s.cloudogu.com/v2",
						UID:                "uid",
						Controller:         &[]bool{true}[0],
						BlockOwnerDeletion: &[]bool{true}[0],
					},
				}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "uid",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Dogu",
					APIVersion: "v1",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name:   "should succeed to set owner reference",
			scheme: scheme,
			ownerReferenceSetterFn: func(t *testing.T) ownerReferenceSetter {
				mck := newMockOwnerReferenceSetter(t)
				mck.EXPECT().SetOwnerReference(testCtx, cescommons.SimpleName("test"), []metav1.OwnerReference{
					{
						Name:               "test",
						Kind:               "Dogu",
						APIVersion:         "k8s.cloudogu.com/v2",
						UID:                "uid",
						Controller:         &[]bool{true}[0],
						BlockOwnerDeletion: &[]bool{true}[0],
					},
				}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "uid",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Dogu",
					APIVersion: "v1",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dcs := &OwnerReferenceStep{
				ownerReferenceSetter: tt.ownerReferenceSetterFn(t),
				doguScheme:           tt.scheme,
			}
			assert.Equalf(t, tt.want, dcs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
