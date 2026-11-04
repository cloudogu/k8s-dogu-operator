package controllers

import (
	"context"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func volumeLabelTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newVolumeLabelFakeClient(funcs interceptor.Funcs, objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(volumeLabelTestScheme()).
		WithObjects(objects...).
		WithInterceptorFuncs(funcs).
		Build()
}

func volumeLabelTestStatefulSet(name string, labels map[string]string, templates ...string) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{Name: name, Namespace: testNamespace, Labels: labels}
	for _, template := range templates {
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{Name: template})
	}
	return sts
}

func volumeLabelTestPVC(name string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{Name: name, Namespace: testNamespace, Labels: labels}
}

func persistedVolumeLabels(t *testing.T, c client.Client, name string) map[string]string {
	t.Helper()
	persisted := &corev1.PersistentVolumeClaim{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, persisted))
	return persisted.Labels
}

func failOnPatch(t *testing.T) interceptor.Funcs {
	return interceptor.Funcs{
		Patch: func(ctx context.Context, _ client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			t.Errorf("unexpected patch of %q", obj.GetName())
			return nil
		},
	}
}

func TestNewDoguVolumeLabelReconciler(t *testing.T) {
	t.Run("should register the controller with the manager if dogu v3 is enabled", func(t *testing.T) {
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(volumeLabelTestScheme())
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(nil)
		managerMock.EXPECT().GetCache().Return(nil)
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		sut, err := NewDoguVolumeLabelReconciler(nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should not register the controller with the manager if dogu v3 is disabled", func(t *testing.T) {
		// the manager mock fails the test on any call
		sut, err := NewDoguVolumeLabelReconciler(nil, newMockCtrlManager(t), &opConfig.OperatorConfig{DoguV3Enabled: false})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should return error if the controller cannot be registered", func(t *testing.T) {
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(volumeLabelTestScheme())
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(assert.AnError)
		managerMock.EXPECT().GetCache().Return(nil).Maybe()
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		_, err := NewDoguVolumeLabelReconciler(nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to setup dogu-v3-volume-labels controller")
	})
}

func TestDoguVolumeLabelReconciler_Reconcile(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "ldap"}

	t.Run("should add both dogu labels to a claim template volume", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{},
			volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
			volumeLabelTestPVC("data-ldap-0", nil),
		)
		sut := newDoguVolumeLabelReconciler(c)

		result, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		expected := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap"}
		assert.Equal(t, expected, persistedVolumeLabels(t, c, "data-ldap-0"))
	})

	t.Run("should overwrite a wrong dogu label value and keep other labels", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{},
			volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
			volumeLabelTestPVC("data-ldap-0", map[string]string{v3beta1.DoguLabelName: "cas", "app": "ces"}),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		expected := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap", "app": "ces"}
		assert.Equal(t, expected, persistedVolumeLabels(t, c, "data-ldap-0"))
	})

	t.Run("should label a volume carrying the backup scope", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{},
			volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
			volumeLabelTestPVC("data-ldap-0", map[string]string{"k8s.cloudogu.com/backup-scope": "misc"}),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		expected := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap", "k8s.cloudogu.com/backup-scope": "misc"}
		assert.Equal(t, expected, persistedVolumeLabels(t, c, "data-ldap-0"))
	})

	t.Run("should ignore a matching statefulset without dogu label", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{},
			// VolumeName would match both StatefulSets
			volumeLabelTestStatefulSet("b-c", map[string]string{"app": "ces"}, "a"),
			volumeLabelTestStatefulSet("c", doguLabels, "a-b"),
			volumeLabelTestPVC("a-b-c-0", nil),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "a-b-c-0"})

		require.NoError(t, err)
		// Labels does not include the here unwanted `app: ces`-label
		expected := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap"}
		assert.Equal(t, expected, persistedVolumeLabels(t, c, "a-b-c-0"))
	})

	for _, name := range []string{"data-ldap-01", "data-ldap-x", "data-ldap-", "other-ldap-0", "data-cas-0", "data-ldap-0-1"} {
		t.Run("should not match volume "+name, func(t *testing.T) {
			c := newVolumeLabelFakeClient(failOnPatch(t),
				volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
				volumeLabelTestPVC(name, nil),
			)
			sut := newDoguVolumeLabelReconciler(c)

			_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: name})

			require.NoError(t, err)
			assert.Nil(t, persistedVolumeLabels(t, c, name))
		})
	}

	t.Run("should return error if the volume cannot be fetched", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			Get: func(ctx context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			},
		})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get volume")
	})

	t.Run("should return error if the statefulsets cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, _ client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			},
		}, volumeLabelTestPVC("data-ldap-0", nil))
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list dogu statefulsets")
	})

	t.Run("should return error if the volume cannot be patched", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			Patch: func(ctx context.Context, _ client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				return assert.AnError
			},
		}, volumeLabelTestStatefulSet("ldap", doguLabels, "data"), volumeLabelTestPVC("data-ldap-0", nil))
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to patch dogu labels")
	})

	t.Run("should do nothing if the volume does not exist", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{}, volumeLabelTestStatefulSet("ldap", doguLabels, "data"))
		sut := newDoguVolumeLabelReconciler(c)

		// Request with non-existent volume-name
		result, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("should leave a volume unchanged that matches claim templates of two dogus", func(t *testing.T) {
		// would throw an error if patch is called
		c := newVolumeLabelFakeClient(failOnPatch(t),
			volumeLabelTestStatefulSet("b-c", doguLabels, "a"),
			volumeLabelTestStatefulSet("c", map[string]string{v3beta1.DoguLabelName: "cas"}, "a-b"),
			volumeLabelTestPVC("a-b-c-0", nil),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "a-b-c-0"})

		require.NoError(t, err)
		assert.Nil(t, persistedVolumeLabels(t, c, "a-b-c-0"))
	})

	t.Run("should leave a volume unchanged that is being deleted", func(t *testing.T) {
		deleting := volumeLabelTestPVC("data-ldap-0", nil)
		deleting.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		deleting.Finalizers = []string{"kubernetes.io/pvc-protection"}
		// would throw an error if patch is called
		c := newVolumeLabelFakeClient(failOnPatch(t), volumeLabelTestStatefulSet("ldap", doguLabels, "data"), deleting)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		assert.Nil(t, persistedVolumeLabels(t, c, "data-ldap-0"))
	})

	t.Run("should not write a volume whose dogu labels are already correct", func(t *testing.T) {
		expected := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap"}
		// would throw an error if patch is called
		c := newVolumeLabelFakeClient(failOnPatch(t),
			volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
			volumeLabelTestPVC("data-ldap-0", expected),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
	})

	t.Run("should leave a volume unchanged that matches no claim template", func(t *testing.T) {
		// would throw an error if patch is called
		c := newVolumeLabelFakeClient(failOnPatch(t),
			volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
			volumeLabelTestPVC("foreign", map[string]string{"app": "foreign"}),
		)
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "foreign"})

		require.NoError(t, err)
	})
}

func TestNewDoguVolumeRequestMapper(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "ldap"}

	t.Run("should map a statefulset to the existing volumes of its claim templates", func(t *testing.T) {
		statefulSet := volumeLabelTestStatefulSet("ldap", doguLabels, "data", "logs")
		c := newVolumeLabelFakeClient(interceptor.Funcs{},
			volumeLabelTestPVC("data-ldap-0", nil),
			volumeLabelTestPVC("data-ldap-1", nil),
			volumeLabelTestPVC("logs-ldap-0", nil),
			volumeLabelTestPVC("data-cas-0", nil),
			volumeLabelTestPVC("data-ldap-01", nil),
		)

		requests := newDoguVolumeRequestMapper(c)(context.Background(), statefulSet)

		assert.ElementsMatch(t, []reconcile.Request{
			{Namespace: testNamespace, Name: "data-ldap-0"},
			{Namespace: testNamespace, Name: "data-ldap-1"},
			{Namespace: testNamespace, Name: "logs-ldap-0"},
		}, requests)
	})

	t.Run("should map a statefulset without claim templates to nothing", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{}, volumeLabelTestPVC("data-ldap-0", nil))

		requests := newDoguVolumeRequestMapper(c)(context.Background(), volumeLabelTestStatefulSet("ldap", doguLabels))

		assert.Empty(t, requests)
	})

	t.Run("should map to nothing if the volumes cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, _ client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			},
		})

		requests := newDoguVolumeRequestMapper(c)(context.Background(), volumeLabelTestStatefulSet("ldap", doguLabels, "data"))

		assert.Empty(t, requests)
	})
}
