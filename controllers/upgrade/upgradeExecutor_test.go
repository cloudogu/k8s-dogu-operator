package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/internal"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks/external"

	"github.com/cloudogu/cesapp-lib/core"
	regmock "github.com/cloudogu/cesapp-lib/registry/mocks"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const redmineUpgradeVersion = "4.2.3-11"

var testCtx = context.TODO()
var testRestConfig = &rest.Config{}

var (
	copyCmd1              = exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved")
	mkdirCmd              = exec.NewShellCommand("/bin/mkdir", "-p", "/")
	copyCmd2              = exec.NewShellCommand("/bin/cp", "/tmp/dogu-reserved/pre-upgrade.sh", "/pre-upgrade.sh")
	preUpgradeCmd         = exec.NewShellCommand("/pre-upgrade.sh", "4.2.3-10", "4.2.3-11")
	postUpgradeCmd        = exec.NewShellCommand("/post-upgrade.sh", "4.2.3-10", "4.2.3-11")
	mockCmdOutput         = bytes.NewBufferString("")
	preUpgradeFileCopyErr = errors.New("cp: can't create '/pre-upgrade.sh': File exists': command terminated with exit code 1")
)

func TestNewUpgradeExecutor(t *testing.T) {
	t.Run("should return a valid object", func(t *testing.T) {
		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		imageRegMock := mocks.NewImageRegistry(t)
		saCreator := mocks.NewServiceAccountCreator(t)
		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		doguRegistry := new(regmock.DoguRegistry)
		mockRegistry := new(regmock.Registry)
		mockRegistry.On("DoguRegistry").Return(doguRegistry, nil)
		eventRecorder := external.NewEventRecorder(t)
		commandExecutor := mocks.NewCommandExecutor(t)

		// when
		actual := NewUpgradeExecutor(myClient, testRestConfig, commandExecutor, eventRecorder, imageRegMock, applier, k8sFileEx, saCreator, mockRegistry)

		// then
		require.NotNil(t, actual)
		assert.IsType(t, &upgradeExecutor{}, actual)
	})
}

func Test_upgradeExecutor_Upgrade(t *testing.T) {
	typeNormal := corev1.EventTypeNormal
	upgradeEvent := EventReason

	toDoguResource := readTestDataRedmineCr(t)
	toDoguResource.Spec.Version = redmineUpgradeVersion

	redmineOldPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "redmine-old-x1y2z3", Labels: toDoguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	redmineOldPod.ObjectMeta.Labels[k8sv1.DoguLabelVersion] = "4.2.3-10"
	redmineUpgradePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "redmine-new-q3w4e5", Labels: toDoguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}

	t.Run("should succeed", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		customK8sResource := map[string]string{"my-custom-resource.yml": "kind: Namespace"}

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(nil)
		execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
		execPod.On("Delete", testCtx).Once().Return(nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForDogu", testCtx, toDoguResource, postUpgradeCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil)

		k8sFileEx := mocks.NewFileExtractor(t)
		k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(customK8sResource, nil)
		applier := mocks.NewCollectApplier(t)
		applier.On("CollectApply", testCtx, customK8sResource, toDoguResource).Return(nil, nil)

		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguDeployment", testCtx, toDoguResource, toDogu, mock.AnythingOfType("func(*v1.Deployment)")).Once().Return(nil, nil)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)
		upserter.On("UpsertDoguPVCs", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying/Updating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional post-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Reverting to original startup probe values...").Once()

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
			doguCommandExecutor:   mockExecutor,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.NoError(t, err)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during revert startup probe", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "namespace-causing-failure")
		dependencyDeployment := createTestDeployment("dependency-dogu", "namespace-causing-failure")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		customK8sResource := map[string]string{"my-custom-resource.yml": "kind: Namespace"}

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(nil)
		execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
		execPod.On("Delete", testCtx).Once().Return(nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForDogu", testCtx, toDoguResource, postUpgradeCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil)

		k8sFileEx := mocks.NewFileExtractor(t)
		k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(customK8sResource, nil)
		applier := mocks.NewCollectApplier(t)
		applier.On("CollectApply", testCtx, customK8sResource, toDoguResource).Return(nil, nil)

		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguDeployment", testCtx, toDoguResource, toDogu, mock.AnythingOfType("func(*v1.Deployment)")).Once().Return(nil, nil)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)
		upserter.On("UpsertDoguPVCs", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying/Updating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional post-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Reverting to original startup probe values...").Once()

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
			doguCommandExecutor:   mockExecutor,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "deployments.apps \"redmine\" not found")
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during K8s resource application", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(nil)
		execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
		execPod.On("Delete", testCtx).Once().Return(nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil)

		k8sFileEx := mocks.NewFileExtractor(t)
		k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(nil, assert.AnError)
		applier := mocks.NewCollectApplier(t)

		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once()

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			eventRecorder:         eventRecorder,
			resourceUpserter:      upserter,
			execPodFactory:        execPodFactory,
			doguCommandExecutor:   mockExecutor,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail during resource update", func(t *testing.T) {
		t.Run("fail on upserting service", func(t *testing.T) {
			// given
			fromDogu := readTestDataDogu(t, redmineBytes)
			toDogu := readTestDataDogu(t, redmineBytes)
			toDogu.Version = redmineUpgradeVersion
			toDogu.Dependencies = []core.Dependency{{
				Type: core.DependencyTypeDogu,
				Name: "dependencyDogu",
			}}

			dependentDeployment := createTestDeployment("redmine", "")
			dependencyDeployment := createTestDeployment("dependency-dogu", "")

			myClient := fake.NewClientBuilder().
				WithScheme(getTestScheme()).
				WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
				Build()

			registrator := mocks.NewDoguRegistrator(t)
			registrator.On("RegisterDoguVersion", toDogu).Return(nil)
			saCreator := mocks.NewServiceAccountCreator(t)
			saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
			imageRegMock := mocks.NewImageRegistry(t)
			image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
			imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

			mockExecutor := mocks.NewCommandExecutor(t)
			k8sFileEx := mocks.NewFileExtractor(t)
			applier := mocks.NewCollectApplier(t)
			upserter := mocks.NewResourceUpserter(t)
			upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, assert.AnError)

			eventRecorder := external.NewEventRecorder(t)
			eventRecorder.
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once()

			execPodFactory := mocks.NewExecPodFactory(t)

			sut := &upgradeExecutor{
				client:                myClient,
				imageRegistry:         imageRegMock,
				collectApplier:        applier,
				k8sFileExtractor:      k8sFileEx,
				serviceAccountCreator: saCreator,
				doguRegistrator:       registrator,
				resourceUpserter:      upserter,
				eventRecorder:         eventRecorder,
				execPodFactory:        execPodFactory,
				doguCommandExecutor:   mockExecutor,
			}

			// when
			err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			// mocks will be asserted during t.CleanUp
		})
		t.Run("fail on upserting exposed services", func(t *testing.T) {
			// given
			fromDogu := readTestDataDogu(t, redmineBytes)
			toDogu := readTestDataDogu(t, redmineBytes)
			toDogu.Version = redmineUpgradeVersion
			toDogu.Dependencies = []core.Dependency{{
				Type: core.DependencyTypeDogu,
				Name: "dependencyDogu",
			}}

			dependentDeployment := createTestDeployment("redmine", "")
			dependencyDeployment := createTestDeployment("dependency-dogu", "")

			myClient := fake.NewClientBuilder().
				WithScheme(getTestScheme()).
				WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
				Build()

			registrator := mocks.NewDoguRegistrator(t)
			registrator.On("RegisterDoguVersion", toDogu).Return(nil)
			saCreator := mocks.NewServiceAccountCreator(t)
			saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
			imageRegMock := mocks.NewImageRegistry(t)
			image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
			imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

			mockExecutor := mocks.NewCommandExecutor(t)
			k8sFileEx := mocks.NewFileExtractor(t)
			applier := mocks.NewCollectApplier(t)
			upserter := mocks.NewResourceUpserter(t)
			upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
			upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, assert.AnError)

			eventRecorder := external.NewEventRecorder(t)
			eventRecorder.
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once()

			execPodFactory := mocks.NewExecPodFactory(t)

			sut := &upgradeExecutor{
				client:                myClient,
				imageRegistry:         imageRegMock,
				collectApplier:        applier,
				k8sFileExtractor:      k8sFileEx,
				serviceAccountCreator: saCreator,
				doguRegistrator:       registrator,
				resourceUpserter:      upserter,
				eventRecorder:         eventRecorder,
				execPodFactory:        execPodFactory,
				doguCommandExecutor:   mockExecutor,
			}

			// when
			err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			// mocks will be asserted during t.CleanUp
		})
		t.Run("fail on upserting deployment", func(t *testing.T) {
			// given
			fromDogu := readTestDataDogu(t, redmineBytes)
			toDogu := readTestDataDogu(t, redmineBytes)
			toDogu.Version = redmineUpgradeVersion
			toDogu.Dependencies = []core.Dependency{{
				Type: core.DependencyTypeDogu,
				Name: "dependencyDogu",
			}}

			dependentDeployment := createTestDeployment("redmine", "")
			dependencyDeployment := createTestDeployment("dependency-dogu", "")

			myClient := fake.NewClientBuilder().
				WithScheme(getTestScheme()).
				WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
				Build()

			registrator := mocks.NewDoguRegistrator(t)
			registrator.On("RegisterDoguVersion", toDogu).Return(nil)
			saCreator := mocks.NewServiceAccountCreator(t)
			saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
			imageRegMock := mocks.NewImageRegistry(t)
			image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
			imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

			execPod := mocks.NewExecPod(t)
			execPod.On("Create", testCtx).Once().Return(nil)
			execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
			execPod.On("Delete", testCtx).Once().Return(nil)

			mockExecutor := mocks.NewCommandExecutor(t)
			mockExecutor.
				On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
				On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
				On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil)

			k8sFileEx := mocks.NewFileExtractor(t)
			k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(nil, nil)
			applier := mocks.NewCollectApplier(t)
			var emptyCustomK8sResource map[string]string
			applier.On("CollectApply", testCtx, emptyCustomK8sResource, toDoguResource).Return(nil, nil)
			upserter := mocks.NewResourceUpserter(t)
			upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
			upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)
			upserter.On("UpsertDoguDeployment", testCtx, toDoguResource, toDogu, mock.AnythingOfType("func(*v1.Deployment)")).Once().Return(nil, assert.AnError)

			eventRecorder := external.NewEventRecorder(t)
			eventRecorder.
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once()

			execPodFactory := mocks.NewExecPodFactory(t)
			execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

			sut := &upgradeExecutor{
				client:                myClient,
				imageRegistry:         imageRegMock,
				collectApplier:        applier,
				k8sFileExtractor:      k8sFileEx,
				serviceAccountCreator: saCreator,
				doguRegistrator:       registrator,
				resourceUpserter:      upserter,
				eventRecorder:         eventRecorder,
				execPodFactory:        execPodFactory,
				doguCommandExecutor:   mockExecutor,
			}

			// when
			err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			// mocks will be asserted during t.CleanUp
		})
		t.Run("fail on upserting pvc", func(t *testing.T) {
			// given
			fromDogu := readTestDataDogu(t, redmineBytes)
			toDogu := readTestDataDogu(t, redmineBytes)
			toDogu.Version = redmineUpgradeVersion
			toDogu.Dependencies = []core.Dependency{{
				Type: core.DependencyTypeDogu,
				Name: "dependencyDogu",
			}}

			dependentDeployment := createTestDeployment("redmine", "")
			dependencyDeployment := createTestDeployment("dependency-dogu", "")

			myClient := fake.NewClientBuilder().
				WithScheme(getTestScheme()).
				WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
				Build()

			registrator := mocks.NewDoguRegistrator(t)
			registrator.On("RegisterDoguVersion", toDogu).Return(nil)
			saCreator := mocks.NewServiceAccountCreator(t)
			saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
			imageRegMock := mocks.NewImageRegistry(t)
			image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
			imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

			execPod := mocks.NewExecPod(t)
			execPod.On("Create", testCtx).Once().Return(nil)
			execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
			execPod.On("Delete", testCtx).Once().Return(nil)

			mockExecutor := mocks.NewCommandExecutor(t)
			mockExecutor.
				On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
				On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
				On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil)

			k8sFileEx := mocks.NewFileExtractor(t)
			k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(nil, nil)
			applier := mocks.NewCollectApplier(t)
			var emptyCustomK8sResource map[string]string
			applier.On("CollectApply", testCtx, emptyCustomK8sResource, toDoguResource).Return(nil, nil)
			upserter := mocks.NewResourceUpserter(t)
			upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
			upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)
			upserter.On("UpsertDoguDeployment", testCtx, toDoguResource, toDogu, mock.AnythingOfType("func(*v1.Deployment)")).Once().Return(nil, nil)
			upserter.On("UpsertDoguPVCs", testCtx, toDoguResource, toDogu).Once().Return(nil, assert.AnError)

			eventRecorder := external.NewEventRecorder(t)
			eventRecorder.
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
				On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once()

			execPodFactory := mocks.NewExecPodFactory(t)
			execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

			sut := &upgradeExecutor{
				client:                myClient,
				imageRegistry:         imageRegMock,
				collectApplier:        applier,
				k8sFileExtractor:      k8sFileEx,
				serviceAccountCreator: saCreator,
				doguRegistrator:       registrator,
				resourceUpserter:      upserter,
				eventRecorder:         eventRecorder,
				execPodFactory:        execPodFactory,
				doguCommandExecutor:   mockExecutor,
			}

			// when
			err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			// mocks will be asserted during t.CleanUp
		})

	})
	t.Run("should fail during post-upgrade execution", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment, redmineOldPod, redmineUpgradePod).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		customK8sResource := map[string]string{"my-custom-resource.yml": "kind: Namespace"}

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(nil)
		execPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)
		execPod.On("Delete", testCtx).Once().Return(nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForDogu", testCtx, toDoguResource, postUpgradeCmd, internal.ContainersStarted).Once().Return(bytes.NewBufferString("ouch"), assert.AnError)

		k8sFileEx := mocks.NewFileExtractor(t)
		k8sFileEx.On("ExtractK8sResourcesFromContainer", testCtx, execPod).Return(customK8sResource, nil)
		applier := mocks.NewCollectApplier(t)
		applier.On("CollectApply", testCtx, customK8sResource, toDoguResource).Return(nil, nil)

		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguDeployment", testCtx, toDoguResource, toDogu, mock.AnythingOfType("func(*v1.Deployment)")).Once().Return(nil, nil)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)
		upserter.On("UpsertDoguPVCs", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying/Updating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional post-upgrade scripts...").Once()

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
			doguCommandExecutor:   mockExecutor,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/post-upgrade.sh 4.2.3-10 4.2.3-11': output: 'ouch'")
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during pre-upgrade script application", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(nil)
		execPod.On("Exec", testCtx, copyCmd1).Once().Return("oh noez", assert.AnError)
		execPod.On("Delete", testCtx).Once().Return(nil)

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once()

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/bin/cp")
		assert.ErrorContains(t, err, "oh noez")
	})
	t.Run("should fail during pre-upgrade script application", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(nil, fmt.Errorf("could not create execPod: %w", assert.AnError))

		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once()

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "could not create execPod")
	})
	t.Run("should fail during K8s resource extraction", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)

		execPod := mocks.NewExecPod(t)
		execPod.On("Create", testCtx).Once().Return(assert.AnError)

		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)
		upserter.On("UpsertDoguService", testCtx, toDoguResource, image).Once().Return(nil, nil)
		upserter.On("UpsertDoguExposedServices", testCtx, toDoguResource, toDogu).Once().Return(nil, nil)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Updating dogu resources in the cluster...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Extracting optional custom K8s resources...").Once()

		execPodFactory := mocks.NewExecPodFactory(t)
		execPodFactory.On("NewExecPod", internal.VolumeModeUpgrade, toDoguResource, toDogu).Return(execPod, nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
			execPodFactory:        execPodFactory,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during image pull", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(nil, assert.AnError)
		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Pulling new image %s:%s...", "registry.cloudogu.com/official/redmine", "4.2.3-11").Once()

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during SA creation", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(assert.AnError)
		imageRegMock := mocks.NewImageRegistry(t)
		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering optional service accounts...").Once()

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail for etcd error", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		dependentDeployment := createTestDeployment("redmine", "")
		dependencyDeployment := createTestDeployment("dependency-dogu", "")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(assert.AnError)
		saCreator := mocks.NewServiceAccountCreator(t)
		imageRegMock := mocks.NewImageRegistry(t)
		k8sFileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		eventRecorder := external.NewEventRecorder(t)
		eventRecorder.On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Registering upgraded version %s in local dogu registry...", "4.2.3-11")

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			k8sFileExtractor:      k8sFileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
			eventRecorder:         eventRecorder,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// mocks will be asserted during t.CleanUp
	})
}

func Test_registerUpgradedDoguVersion(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		doguRegistryMock := new(regmock.DoguRegistry)
		registryMock := new(regmock.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(true, nil)
		doguRegistryMock.On("Register", toDogu).Return(nil)
		doguRegistryMock.On("Enable", toDogu).Return(nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.NoError(t, err)
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
	t.Run("should fail", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion

		doguRegistryMock := new(regmock.DoguRegistry)
		registryMock := new(regmock.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(false, nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register upgrade: could not register dogu version: previous version not found")
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
}

func Test_registerNewServiceAccount(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(nil)

		// when
		err := registerNewServiceAccount(testCtx, saCreator, toDogu)

		// then
		require.NoError(t, err)
		saCreator.AssertExpectations(t)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDogu).Return(assert.AnError)

		// when
		err := registerNewServiceAccount(testCtx, saCreator, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register service accounts: assert.AnError")
		saCreator.AssertExpectations(t)
	})
}

func Test_upgradeExecutor_pullUpgradeImage(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		imagePuller := mocks.NewImageRegistry(t)
		doguImage := toDogu.Image + ":" + toDogu.Version

		imagePuller.On("PullImageConfig", testCtx, doguImage).Return(&imagev1.ConfigFile{}, nil)

		// when
		image, err := pullUpgradeImage(testCtx, imagePuller, toDogu)

		// then
		require.NoError(t, err)
		require.NotNil(t, image)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		imagePuller := mocks.NewImageRegistry(t)
		doguImage := toDogu.Image + ":" + toDogu.Version
		var noConfigFile *imagev1.ConfigFile

		imagePuller.On("PullImageConfig", testCtx, doguImage).Return(noConfigFile, assert.AnError)

		// when
		_, err := pullUpgradeImage(testCtx, imagePuller, toDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to pull upgrade image: assert.AnError")
	})
}

func Test_extractCustomK8sResources(t *testing.T) {
	t.Run("should return custom K8s resources", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, mock.Anything).Return(fakeResources, nil)

		// when
		resources, err := extractCustomK8sResources(testCtx, extractor, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, fakeResources, resources)
	})
	t.Run("should return no custom K8s resources", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		var emptyResourcesAreValidToo map[string]string
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, mock.Anything).Return(emptyResourcesAreValidToo, nil)

		// when
		resources, err := extractCustomK8sResources(testCtx, extractor, nil)

		// then
		require.NoError(t, err)
		assert.Nil(t, resources)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		var nilMap map[string]string
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, mock.Anything).Return(nilMap, assert.AnError)

		// when
		_, err := extractCustomK8sResources(testCtx, extractor, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to extract custom K8s resources: assert.AnError")
	})
}

func Test_applyCustomK8sResources(t *testing.T) {
	t.Run("should apply K8s resources", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		collectApplier := mocks.NewCollectApplier(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		collectApplier.On("CollectApply", mock.Anything, fakeResources, toDoguCr).Return(nil)

		// when
		err := applyCustomK8sResources(testCtx, collectApplier, toDoguCr, fakeResources)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		collectApplier := mocks.NewCollectApplier(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		collectApplier.On("CollectApply", mock.Anything, fakeResources, toDoguCr).Return(assert.AnError)

		// when
		err := applyCustomK8sResources(testCtx, collectApplier, toDoguCr, fakeResources)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to apply custom K8s resources: assert.AnError")
	})
}

func Test_upgradeExecutor_applyPreUpgradeScripts(t *testing.T) {

	doguResource := readTestDataRedmineCr(t)
	redmineOldPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "redmine-old-x1y2z3", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	scriptInterpreterOutput := bytes.NewBufferString("#!/bin/bash\n")

	t.Run("should be successful if no pre-upgrade exposed command", func(t *testing.T) {
		// given
		toDoguResource := &k8sv1.Dogu{}
		mockExecPod := mocks.NewExecPod(t)

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.ExposedCommands = []core.ExposedCommand{}

		upgradeExecutor := upgradeExecutor{}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail if copy from pod to pod fails because of a non-retryable reason", func(t *testing.T) {
		// given
		toDoguResource := &k8sv1.Dogu{}
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		mockExecPod := mocks.NewExecPod(t)
		copy1 := exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved")
		mockExecPod.On("Exec", testCtx, copy1).Once().Return("oopsie woopsie", assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{eventRecorder: eventRecorder}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/bin/cp /pre-upgrade.sh /tmp/dogu-reserved' in execpod, stdout: 'oopsie woopsie'")
	})
	t.Run("should fail if upgrade dir creation fails", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(bytes.NewBufferString("oops"), assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/bin/mkdir -p /': output: 'oops'")
	})
	t.Run("should fail if copy to original dir fails", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(bytes.NewBufferString("oops"), assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/bin/cp /tmp/dogu-reserved/pre-upgrade.sh /pre-upgrade.sh': output: 'oops'")
	})
	t.Run("should fail during pre-upgrade execution", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(bytes.NewBufferString("uhoh"), assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/pre-upgrade.sh 4.2.3-10 4.2.3-11': output: 'uhoh'")
	})
	t.Run("should fail if copy from pod to pod fails and re-try fails as well", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		copy1 := exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved")
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copy1).Once().Return("", nil)

		copy2 := exec.NewShellCommand("/bin/cp", "/tmp/dogu-reserved/pre-upgrade.sh", "/pre-upgrade.sh")
		detectInterpreter := exec.NewShellCommand("/bin/sh", "-c", "/bin/grep '#!' /tmp/dogu-reserved/pre-upgrade.sh")
		shellPipeExec := exec.NewShellCommand("/bin/sh", "-c", `cd $(dirname /pre-upgrade.sh) && (cat /tmp/dogu-reserved/pre-upgrade.sh | /bin/bash -s "4.2.3-10" "4.2.3-11")`)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copy2, internal.ContainersStarted).Once().Return(mockCmdOutput, preUpgradeFileCopyErr).
			On("ExecCommandForPod", testCtx, redmineOldPod, detectInterpreter, internal.PodReady).Once().Return(scriptInterpreterOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, shellPipeExec, internal.PodReady).Once().Return(mockCmdOutput, assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Running pre-upgrade script with shell pipe strategy instead...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to re-execute pre-upgrade script '/pre-upgrade.sh'")
	})
	t.Run("should fail if copy from pod to pod fails and re-try fails because script interpreter detection", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		copy1 := exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved")
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copy1).Once().Return("", nil)

		copy2 := exec.NewShellCommand("/bin/cp", "/tmp/dogu-reserved/pre-upgrade.sh", "/pre-upgrade.sh")
		detectInterpreter := exec.NewShellCommand("/bin/sh", "-c", "/bin/grep '#!' /tmp/dogu-reserved/pre-upgrade.sh")

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copy2, internal.ContainersStarted).Once().Return(mockCmdOutput, preUpgradeFileCopyErr).
			On("ExecCommandForPod", testCtx, redmineOldPod, detectInterpreter, internal.PodReady).Once().Return(mockCmdOutput, assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Running pre-upgrade script with shell pipe strategy instead...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to re-execute pre-upgrade script '/pre-upgrade.sh'")
	})
	t.Run("should succeed after failed script copy but on shell pipe retry", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		copy1 := exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved")
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copy1).Once().Return("", nil)

		copy2 := exec.NewShellCommand("/bin/cp", "/tmp/dogu-reserved/pre-upgrade.sh", "/pre-upgrade.sh")
		detectInterpreter := exec.NewShellCommand("/bin/sh", "-c", "/bin/grep '#!' /tmp/dogu-reserved/pre-upgrade.sh")
		shellPipeExec := exec.NewShellCommand("/bin/sh", "-c", `cd $(dirname /pre-upgrade.sh) && (cat /tmp/dogu-reserved/pre-upgrade.sh | /bin/bash -s "4.2.3-10" "4.2.3-11")`)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copy2, internal.ContainersStarted).Once().Return(mockCmdOutput, preUpgradeFileCopyErr).
			On("ExecCommandForPod", testCtx, redmineOldPod, detectInterpreter, internal.PodReady).Once().Return(scriptInterpreterOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, shellPipeExec, internal.PodReady).Once().Return(mockCmdOutput, nil)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Running pre-upgrade script with shell pipe strategy instead...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed after successful script copy", func(t *testing.T) {
		// given
		cli := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineOldPod).
			Build()
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Exec", testCtx, copyCmd1).Once().Return("", nil)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.
			On("ExecCommandForPod", testCtx, redmineOldPod, mkdirCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, copyCmd2, internal.ContainersStarted).Once().Return(mockCmdOutput, nil).
			On("ExecCommandForPod", testCtx, redmineOldPod, preUpgradeCmd, internal.PodReady).Once().Return(mockCmdOutput, nil)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Copying optional pre-upgrade scripts...").Once().
			On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional pre-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{
			client:              cli,
			eventRecorder:       eventRecorder,
			doguCommandExecutor: mockExecutor,
		}

		// when
		err := upgradeExecutor.applyPreUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu, mockExecPod)

		// then
		require.NoError(t, err)
	})
}

func Test_upgradeExecutor_applyPostUpgradeScript(t *testing.T) {
	t.Run("should succeed if no post-upgrade exposed command", func(t *testing.T) {
		// given
		toDoguResource := &k8sv1.Dogu{}

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.ExposedCommands = []core.ExposedCommand{}

		sut := upgradeExecutor{}

		// when
		err := sut.applyPostUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail on executing post-upgrade script", func(t *testing.T) {
		// given
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.On("ExecCommandForDogu", testCtx, toDoguResource, postUpgradeCmd, internal.ContainersStarted).Once().Return(bytes.NewBufferString("oof"), assert.AnError)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional post-upgrade scripts...").Once()

		sut := upgradeExecutor{doguCommandExecutor: mockExecutor, eventRecorder: eventRecorder}

		// when
		err := sut.applyPostUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute '/post-upgrade.sh 4.2.3-10 4.2.3-11': output: 'oof'")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		mockExecutor := mocks.NewCommandExecutor(t)
		mockExecutor.On("ExecCommandForDogu", testCtx, toDoguResource, postUpgradeCmd, internal.ContainersStarted).Once().Return(mockCmdOutput, nil)

		eventRecorder := external.NewEventRecorder(t)
		typeNormal := corev1.EventTypeNormal
		upgradeEvent := EventReason
		eventRecorder.On("Eventf", toDoguResource, typeNormal, upgradeEvent, "Applying optional post-upgrade scripts...").Once()

		upgradeExecutor := upgradeExecutor{eventRecorder: eventRecorder, doguCommandExecutor: mockExecutor}

		// when
		err := upgradeExecutor.applyPostUpgradeScript(testCtx, toDoguResource, fromDogu, toDogu)

		// then
		require.NoError(t, err)
	})
}

func createTestDeployment(doguName string, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 1},
	}
}

func Test_getMapKeysAsString(t *testing.T) {
	t.Run("should return beautiful list", func(t *testing.T) {
		// given
		inputList := map[string]string{
			"test.json":    "bytes and bytes",
			"another.json": "even more bytes and bytes",
		}

		// when
		output := util.GetMapKeysAsString(inputList)

		// then
		assert.Contains(t, output, "test.json")
		assert.Contains(t, output, "another.json")
	})
}

func Test_increaseStartupProbeTimeoutForUpdate(t *testing.T) {
	t.Run("should edit existing startup probe threshold for same container name", func(t *testing.T) {
		// given
		expectedDeployment := createTestDeploymentWithStartupProbe("ldap", 1080)
		patchDeployment := createTestDeploymentWithStartupProbe("ldap", 3)

		// when
		increaseStartupProbeTimeoutForUpdate("ldap", patchDeployment)

		// then
		require.NotNil(t, patchDeployment)
		require.NotEqual(t, 3, patchDeployment.Spec.Template.Spec.Containers[0].StartupProbe.FailureThreshold)
		require.Equal(t, expectedDeployment, patchDeployment)
	})

	t.Run("do nothing if no container matches", func(t *testing.T) {
		// given
		patchDeployment := createTestDeploymentWithStartupProbe("ldap", 3)

		// when
		increaseStartupProbeTimeoutForUpdate("ldap-side-bmw", patchDeployment)

		// then
		require.NotNil(t, patchDeployment)
		require.Equal(t, 1, len(patchDeployment.Spec.Template.Spec.Containers))
	})
}

func Test_revertStartupProbeAfterUpdate(t *testing.T) {
	t.Run("should fail because deployment cannot be found", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDoguResource := readTestDataRedmineCr(t)
		myClient := fake.NewClientBuilder().Build()

		// when
		err := revertStartupProbeAfterUpdate(testCtx, toDoguResource, toDogu, myClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "deployments.apps \"redmine\" not found")
	})
	t.Run("should successfully update startup probes", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDoguResource := readTestDataRedmineCr(t)
		deployment := createTestDeployment("redmine", "")
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name: toDoguResource.Name,
			StartupProbe: &corev1.Probe{
				FailureThreshold: int32(100),
			},
		}}
		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, deployment).
			Build()

		// when
		err := revertStartupProbeAfterUpdate(testCtx, toDoguResource, toDogu, myClient)

		// then
		require.NoError(t, err)
		actualDeployment := &appsv1.Deployment{}
		err = myClient.Get(testCtx, toDoguResource.GetObjectKey(), actualDeployment)
		require.NoError(t, err)
		assert.Equal(t, int32(180), actualDeployment.Spec.Template.Spec.Containers[0].StartupProbe.FailureThreshold)
	})
}

func createTestDeploymentWithStartupProbe(containerName string, threshold int32) *appsv1.Deployment {
	container := corev1.Container{
		Name: containerName,
		StartupProbe: &corev1.Probe{
			FailureThreshold: threshold,
		},
	}
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}
}

func Test_deleteExecPod(t *testing.T) {
	t.Run("should throw event if delete fails", func(t *testing.T) {
		// given
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.
			On("Delete", testCtx).Once().Return(assert.AnError).
			On("PodName").Once().Return("test-pod")

		mockRecorder := external.NewEventRecorder(t)
		mockRecorder.On("Eventf", &k8sv1.Dogu{}, corev1.EventTypeNormal, EventReason, "Failed to delete execPod %s: %w", "test-pod", assert.AnError).Once()

		// when
		deleteExecPod(testCtx, mockExecPod, mockRecorder, &k8sv1.Dogu{})

		// then
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockExecPod := mocks.NewExecPod(t)
		mockExecPod.On("Delete", testCtx).Once().Return(nil)

		// when
		deleteExecPod(testCtx, mockExecPod, nil, nil)

		// then
		// mocks will be asserted during t.CleanUp
	})
}

func Test_isPreUpgradeExecErrRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"true for full error", errors.New("dogu upgrade official/scm:2.40.0-1 failed: pre-upgrade failed :failed to execute '/bin/cp /tmp/dogu-reserved/pre-upgrade.sh /pre-upgrade.sh': output: '<nil>': resource does not meet requirements for exec: scm-55dbffc4f8-cjtf9, source error: error streaming command to pod; out: '': errOut: 'cp: can't create '/pre-upgrade.sh': File exists': command terminated with exit code 1"), true},
		{"true for partial error", preUpgradeFileCopyErr, true},
		{"false for similar error", errors.New("cp: can't do things with '/pre-upgrade.sh': I forgot how to compute': command terminated with exit code 1"), false},
		{"false for unrelated error", assert.AnError, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, isPreUpgradeExecErrRetryable(tt.err), "isPreUpgradeExecErrRetryable(%v)", tt.err)
		})
	}
}

func Test_splitScriptInterpreter(t *testing.T) {
	type args struct {
		interpreterOut string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{"return bash path", args{"#!/bin/bash"}, "/bin/bash", assert.NoError},
		{"return dash env", args{"#!/bin/env dash"}, "/bin/env dash", assert.NoError},
		{"removes newline", args{"#!/bin/bash\n"}, "/bin/bash", assert.NoError},
		{"error on missing shebang", args{""}, "", assert.Error},
		{"error on weird comment", args{"#!!! Please note !!!"}, "", assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getScriptInterpreterFromOutput(tt.args.interpreterOut)
			if !tt.wantErr(t, err, fmt.Sprintf("getScriptInterpreterFromOutput(%v)", tt.args.interpreterOut)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getScriptInterpreterFromOutput(%v)", tt.args.interpreterOut)
		})
	}
}
