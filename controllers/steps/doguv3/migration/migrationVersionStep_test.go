package migration

import (
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// versionedDogu returns a dogu that has v2 version 2.4.48-3 installed and requests chart version 1.0.0.
func versionedDogu() *v3beta1.Dogu {
	dogu := testDogu()
	dogu.Spec.Version = "1.0.0"
	dogu.Status.InstalledVersion = "2.4.48-3"
	return dogu
}

func TestMigrationVersionStep_Run(t *testing.T) {
	t.Run("continues if the appVersion matches the v2 version without dogu revision", func(t *testing.T) {
		dogu := versionedDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
		chartReader := NewMockChartAppVersionReader(t)
		chartReader.EXPECT().AppVersion(testCtx, dogu).Return("2.4.48", nil)

		result := NewMigrationVersionStep(detector, chartReader, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("continues if the v2 version has no dogu revision", func(t *testing.T) {
		dogu := versionedDogu()
		dogu.Status.InstalledVersion = "2.4.48"
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
		chartReader := NewMockChartAppVersionReader(t)
		chartReader.EXPECT().AppVersion(testCtx, dogu).Return("2.4.48", nil)

		result := NewMigrationVersionStep(detector, chartReader, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	for _, state := range []dogumigration.State{dogumigration.NotRequired, dogumigration.InProgress} {
		t.Run("continues without check if the migration state is not required", func(t *testing.T) {
			dogu := versionedDogu()
			detector := NewMockMigrationDetector(t)
			detector.EXPECT().Detect(testCtx, dogu).Return(state, nil)

			result := NewMigrationVersionStep(detector, NewMockChartAppVersionReader(t), NewMockEventRecorder(t)).Run(testCtx, dogu)

			assert.Equal(t, stepsv3.Continue(), result)
		})
	}

	t.Run("requeues if the migration state cannot be detected", func(t *testing.T) {
		dogu := versionedDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.NotRequired, errors.New("boom"))

		result := NewMigrationVersionStep(detector, NewMockChartAppVersionReader(t), NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.ErrorContains(t, result.Err, "boom")
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})

	t.Run("requeues if the appVersion cannot be read", func(t *testing.T) {
		dogu := versionedDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
		chartReader := NewMockChartAppVersionReader(t)
		chartReader.EXPECT().AppVersion(testCtx, dogu).Return("", errors.New("boom"))

		result := NewMigrationVersionStep(detector, chartReader, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.ErrorContains(t, result.Err, "boom")
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})

	abortTests := []struct {
		name       string
		v2Version  string
		appVersion string
		wantMsg    string
	}{
		{
			name:       "different appVersion",
			v2Version:  "2.4.48-3",
			appVersion: "2.4.49",
			wantMsg:    `The migration to v3 is blocked: the appVersion "2.4.49" of chart version "1.0.0" does not match the version "2.4.48-3" of the installed v2 dogu`,
		},
		{
			name:       "appVersion containing the dogu revision",
			v2Version:  "2.4.48-3",
			appVersion: "2.4.48-3",
			wantMsg:    `The migration to v3 is blocked: the appVersion "2.4.48-3" of chart version "1.0.0" does not match the version "2.4.48-3" of the installed v2 dogu`,
		},
		{
			name:       "missing appVersion",
			v2Version:  "2.4.48-3",
			appVersion: "",
			wantMsg:    `The migration to v3 is blocked: the chart version "1.0.0" does not set an appVersion`,
		},
	}
	for _, tt := range abortTests {
		t.Run("aborts on "+tt.name, func(t *testing.T) {
			dogu := versionedDogu()
			dogu.Status.InstalledVersion = tt.v2Version
			detector := NewMockMigrationDetector(t)
			detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
			chartReader := NewMockChartAppVersionReader(t)
			chartReader.EXPECT().AppVersion(testCtx, dogu).Return(tt.appVersion, nil)
			recorder := NewMockEventRecorder(t)
			recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, ReasonMigrationVersionMismatch, tt.wantMsg)

			result := NewMigrationVersionStep(detector, chartReader, recorder).Run(testCtx, dogu)

			assert.Equal(t, stepsv3.Abort(ReasonMigrationVersionMismatch, tt.wantMsg), result)
		})
	}

	t.Run("aborts without reading the chart if the v2 version is unknown", func(t *testing.T) {
		dogu := versionedDogu()
		dogu.Status.InstalledVersion = ""
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
		wantMsg := "The migration to v3 is blocked: the installed version of the v2 dogu is unknown"
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, ReasonMigrationVersionMismatch, wantMsg)

		result := NewMigrationVersionStep(detector, NewMockChartAppVersionReader(t), recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Abort(ReasonMigrationVersionMismatch, wantMsg), result)
	})
}
