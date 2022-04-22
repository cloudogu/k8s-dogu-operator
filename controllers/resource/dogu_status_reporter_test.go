package resource_test

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguErrorReporter(t *testing.T) {
	// given
	fakeClient := fake.NewClientBuilder().Build()

	// when
	reporter := resource.NewDoguStatusReporter(fakeClient)

	// then
	require.NotNil(t, reporter)
	assert.Equal(t, fakeClient, reporter.KubernetesClient)
}

type reportableFakeError struct {
	taste string
}

func (r *reportableFakeError) Error() string {
	return "my test error"
}
func (r *reportableFakeError) Report() string {
	return fmt.Sprintf("it tastes like: %s", r.taste)
}

func TestDoguErrorReporter_ReportError(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &k8sv1.Dogu{})

	t.Run("report with an error without a reportable error interface", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		err := reporter.ReportError(context.Background(), doguResource, fmt.Errorf("my error"))

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Len(t, savedDogu.Status.StatusMessages, 0)
	})

	t.Run("report with an error with reportable error interface", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		myError := reportableFakeError{taste: "testing"}
		err := reporter.ReportError(context.Background(), doguResource, &myError)

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Len(t, savedDogu.Status.StatusMessages, 1)
		assert.Equal(t, "it tastes like: testing", savedDogu.Status.StatusMessages[0])
	})

	t.Run("report with an error with multiple wrapped reportable errors", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		myReportableError := &reportableFakeError{taste: "water"}
		myError := fmt.Errorf("failed to do it: %w", myReportableError)
		err := reporter.ReportError(context.Background(), doguResource, myError)

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Len(t, savedDogu.Status.StatusMessages, 1)
		assert.Equal(t, "it tastes like: water", savedDogu.Status.StatusMessages[0])
	})

	t.Run("report with a multierror containing multiple reportable errors", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		var resultError error
		resultError = multierror.Append(resultError, &reportableFakeError{taste: "water"})
		resultError = multierror.Append(resultError, &reportableFakeError{taste: "earth"})
		resultError = multierror.Append(resultError, &reportableFakeError{taste: "fire"})
		resultError = multierror.Append(resultError, &reportableFakeError{taste: "air"})
		err := reporter.ReportError(context.Background(), doguResource, resultError)

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Len(t, savedDogu.Status.StatusMessages, 4)
		assert.Equal(t, "it tastes like: water", savedDogu.Status.StatusMessages[0])
		assert.Equal(t, "it tastes like: earth", savedDogu.Status.StatusMessages[1])
		assert.Equal(t, "it tastes like: fire", savedDogu.Status.StatusMessages[2])
		assert.Equal(t, "it tastes like: air", savedDogu.Status.StatusMessages[3])
	})

	t.Run("fail report on updating the status", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		err := reporter.ReportError(context.Background(), doguResource, fmt.Errorf("error"))

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update dogu status")
	})
}

func TestDoguErrorReporter_ReportMessage(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &k8sv1.Dogu{})

	t.Run("report a single message", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		err := reporter.ReportMessage(context.Background(), doguResource, "this is my message")

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Equal(t, "this is my message", savedDogu.Status.StatusMessages[0])
	})

	t.Run("report multiple message", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		err := reporter.ReportMessage(context.Background(), doguResource, "this is my message")
		require.NoError(t, err)
		err = reporter.ReportMessage(context.Background(), doguResource, "this is my message 2")
		require.NoError(t, err)

		// then
		require.NoError(t, err)

		var savedDogu k8sv1.Dogu
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "testdogu", Namespace: "testnamespace"}, &savedDogu)
		require.NoError(t, err)

		assert.Equal(t, "this is my message", savedDogu.Status.StatusMessages[0])
		assert.Equal(t, "this is my message 2", savedDogu.Status.StatusMessages[1])
	})

	t.Run("fail report on updating the status", func(t *testing.T) {
		// given
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "testdogu", Namespace: "testnamespace"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reporter := resource.NewDoguStatusReporter(fakeClient)

		// when
		err := reporter.ReportMessage(context.Background(), doguResource, "this is my message")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update dogu status")
	})
}
