//go:build k8s_integration
// +build k8s_integration

package controllers

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

type mockeryGinkgoLogger struct {
}

func (c mockeryGinkgoLogger) Logf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(GinkgoWriter, strings.ReplaceAll(strings.ReplaceAll(format, "PASS", "\nPASS"), "FAIL", "\nFAIL"), args...)
}

func (c mockeryGinkgoLogger) Errorf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(GinkgoWriter, format, args...)
}

func (c mockeryGinkgoLogger) FailNow() {
	println("fail now")
}

var _ = Describe("Dogu Upgrade Tests", func() {
	t := &testing.T{}
	mockeryT := &mockeryGinkgoLogger{}

	// Install testdata
	ldapCr := readDoguCr(t, ldapCrBytes)
	redmineCr := readDoguCr(t, redmineCrBytes)
	imageConfig := readImageConfig(t, imageConfigBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	redmineDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)

	ldapCr.Namespace = "default"
	ldapCr.ResourceVersion = ""
	ldapDoguLookupKey := types.NamespacedName{Name: ldapCr.Name, Namespace: ldapCr.Namespace}
	cesLoadbalancerLookupKey := types.NamespacedName{Name: "ces-loadbalancer", Namespace: "default"}
	tcpExposedPortsLookupKey := types.NamespacedName{Name: "tcp-services", Namespace: "default"}

	redmineCr.Namespace = "default"
	redmineCr.ResourceVersion = ""

	ctx := context.TODO()

	// Upgrade testdata
	upgradeNamespace := "upgrade"
	upgradeLdapToDoguDescriptor := readDoguDescriptor(t, ldapUpgradeDoguDescriptorBytes)
	upgradeLdapToDoguDescriptor.Name = upgradeNamespace + "/ldap"
	ldapToVersion := upgradeLdapToDoguDescriptor.Version

	upgradeLdapFromCr := readDoguCr(t, ldapCrBytes)
	upgradeLdapFromCr.ResourceVersion = ""
	upgradeLdapFromCr.Namespace = upgradeNamespace

	upgradeLdapFromDoguDescriptor := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	upgradeLdapFromDoguDescriptor.Name = upgradeNamespace + "/ldap"
	upgradeLdapFromDoguLookupKey := types.NamespacedName{Name: upgradeLdapFromCr.Name, Namespace: upgradeNamespace}

	Context("Handle dogu resource", func() {
		It("Setup mocks and test data", func() {
			*ImageRegistryMock = mocks.ImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			*DoguRemoteRegistryMock = extMocks.RemoteRegistry{}
			DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.48-4").Return(ldapDogu, nil)
			DoguRemoteRegistryMock.Mock.On("GetVersion", "official/redmine", "4.2.3-10").Return(redmineDogu, nil)

			*EtcdDoguRegistry = extMocks.DoguRegistry{}
			EtcdDoguRegistry.Mock.On("Get", "postgresql").Return(nil, fmt.Errorf("not installed"))
			EtcdDoguRegistry.Mock.On("Get", "cas").Return(nil, fmt.Errorf("not installed"))
			EtcdDoguRegistry.Mock.On("Get", "postfix").Return(nil, fmt.Errorf("not installed"))
			EtcdDoguRegistry.Mock.On("Get", "nginx-ingress").Return(nil, fmt.Errorf("not installed"))
			EtcdDoguRegistry.Mock.On("Get", "nginx-static").Return(nil, fmt.Errorf("not installed"))
			EtcdDoguRegistry.Mock.On("Get", "ldap").Return(ldapDogu, nil)
			EtcdDoguRegistry.Mock.On("Get", "redmine").Return(redmineDogu, nil)
			EtcdDoguRegistry.Mock.On("Register", mock.Anything).Return(nil)
			EtcdDoguRegistry.Mock.On("Unregister", mock.Anything).Return(nil)
			EtcdDoguRegistry.Mock.On("Enable", mock.Anything).Return(nil)
			EtcdDoguRegistry.Mock.On("IsEnabled", mock.Anything).Return(false, nil)
		})

		It("Should install dogu in cluster", func() {
			By("Creating dogu resource")
			installDoguCr(ctx, ldapCr)

			By("Expect created dogu")
			createdDogu := &k8sv1.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				if err != nil {
					return false
				}
				finalizers := createdDogu.Finalizers
				if len(finalizers) == 1 && finalizers[0] == "dogu-finalizer" {
					return true
				}
				return false
			}, TimeoutInterval, PollingInterval).Should(BeTrue())

			setExecPodRunning(ctx, "ldap")

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, deployment)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, deployment.ObjectMeta)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(deployment.Name))
			Expect(ldapCr.Namespace).To(Equal(deployment.Namespace))

			By("Expect created service")
			service := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, service)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, service.ObjectMeta)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(service.Name))
			Expect(ldapCr.Namespace).To(Equal(service.Namespace))

			By("Expect created loadbalancer service")
			lbService := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cesLoadbalancerLookupKey, lbService)
				if err != nil {
					return false
				}

				found := false
				for _, doguPort := range ldapDogu.ExposedPorts {
					for _, port := range lbService.Spec.Ports {
						if port.Port == int32(doguPort.Host) && port.TargetPort.IntVal == int32(doguPort.Container) &&
							strings.EqualFold(string(port.Protocol), doguPort.Type) &&
							port.Name == fmt.Sprintf("%s-%d", ldapDogu.GetSimpleName(), doguPort.Host) {
							found = true
							break
						}
					}
					if !found {
						return false
					}
				}

				return true
			}, TimeoutInterval, PollingInterval).Should(BeTrue())

			By("Expect created tcp/udp configmap")
			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, tcpExposedPortsLookupKey, cm)
				if err != nil && cm.Data == nil && len(cm.Data) != 2 {
					return false
				}

				if cm.Data["2222"] != "default/ldap:2222" && cm.Data["8888"] == "default/ldap:8888" {
					return false
				}

				return true
			}, TimeoutInterval, PollingInterval).Should(BeTrue())

			By("Expect created secret")
			secret := &corev1.Secret{}
			secretLookupKey := types.NamespacedName{Name: ldapCr.Name + "-private", Namespace: ldapCr.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, secret)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, secret.ObjectMeta)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name + "-private").To(Equal(secret.Name))
			Expect(ldapCr.Namespace).To(Equal(secret.Namespace))

			By("Expect created dogu pvc")
			doguPvc := &corev1.PersistentVolumeClaim{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, doguPvc)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, doguPvc.ObjectMeta)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(doguPvc.Name))
			Expect(ldapCr.Namespace).To(Equal(doguPvc.Namespace))
			Expect(resource.MustParse("2Gi")).To(Equal(*doguPvc.Spec.Resources.Requests.Storage()))
		})

		It("Update dogus additional ingress annotations", func() {
			By("Update dogu resource with ingress annotations")
			createdDogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			if createdDogu.Spec.AdditionalIngressAnnotations == nil {
				createdDogu.Spec.AdditionalIngressAnnotations = map[string]string{}
			}
			createdDogu.Spec.AdditionalIngressAnnotations["new"] = "new"
			updateDoguCr(ctx, createdDogu)

			By("Expect service with additional ingress annotations")
			service := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, service)
				if err != nil {
					return false
				}

				s, exists := service.ObjectMeta.Annotations["k8s-dogu-operator.cloudogu.com/additional-ingress-annotations"]
				if exists && s == "{\"new\": \"new\"}" {
					return true
				}

				return false
			})
		})

		It("Set dogu in support mode", func() {
			By("Update dogu resource with support mode true")
			createdDogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			createdDogu.Spec.SupportMode = true
			updateDoguCr(ctx, createdDogu)

			By("Expect deployment in support mode")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, deployment)
				if err != nil {
					return false
				}
				if isDeploymentInSupportMode(deployment) {
					return true
				}
				return false
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
		})

		It("Should unset dogu support mode", func() {
			By("Update dogu resource with support mode false")
			createdDogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			createdDogu.Spec.SupportMode = false
			updateDoguCr(ctx, createdDogu)

			By("Expect deployment in normal mode")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, deployment)
				if err != nil {
					return false
				}
				return !isDeploymentInSupportMode(deployment)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
		})

		// This test does not work because the client in this environment can't even update the storage request of a pvc.
		// It can be used in planned future environments with real clusters.
		//
		// It("Should resize dogu volume", func() {
		// 	By("Update dogu resource with dataVolumeSize")
		// 	createdDogu := &k8sv1.Dogu{}
		// 	Eventually(func() bool {
		// 		err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
		// 		return err == nil
		// 	}, PollingInterval, TimeoutInterval).Should(BeTrue())
		//
		// 	newVolumeSize := "10Gi"
		// 	newVolumeQuantity := resource.MustParse(newVolumeSize)
		// 	createdDogu.Spec.Resources.DataVolumeSize = newVolumeSize
		// 	updateDoguCr(ctx, createdDogu)
		//
		// 	By("Expect expanded pvc")
		// 	pvc := &corev1.PersistentVolumeClaim{}
		//
		// 	Eventually(func() bool {
		// 		err := k8sClient.Get(ctx, ldapDoguLookupKey, pvc)
		// 		if err != nil {
		// 			return false
		// 		}
		//
		// 		// Does not work with the actual test environment can be use with a real cluster.
		// 		// hasSizeInStatus := pvc.Status.Capacity.Storage().Equal(newVolumeQuantity)
		//
		//
		// 		hasSizeInSpec := pvc.Spec.Resources.Requests.Storage().Equal(newVolumeQuantity)
		// 		if hasSizeInSpec {
		// 			return true
		// 		}
		//
		// 		return false
		// 	}, TimeoutInterval, PollingInterval).Should(BeTrue())
		// })

		It("Should delete dogu", func() {
			deleteDoguCr(ctx, ldapCr, true)

			By("LoadBalancer service should be deleted")
			lbService := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cesLoadbalancerLookupKey, lbService)
				return apierrors.IsNotFound(err)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())

			By("Expected deleted entries in tcp/udp configmap")
			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, tcpExposedPortsLookupKey, cm)
				if err != nil && len(cm.Data) != 0 {
					return false
				}

				return true
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
		})
	})

	It("Should fail dogu installation as dependency is missing", func() {
		By("Creating redmine dogu resource")
		installDoguCr(ctx, redmineCr)

		By("Check for failed installation and check events of dogu resource")
		createdDogu := &k8sv1.Dogu{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, redmineCr.GetObjectKey(), createdDogu)
			if err != nil {
				return false
			}
			if createdDogu.Status.Status != k8sv1.DoguStatusNotInstalled {
				return false
			}

			eventList := &corev1.EventList{}
			err = k8sClient.List(ctx, eventList, &client.ListOptions{})
			if err != nil {
				return false
			}

			count := 0
			for _, item := range eventList.Items {
				if item.InvolvedObject.Name == createdDogu.Name && item.Reason == ErrorOnInstallEventReason {
					count++
				}
			}

			return count == 1
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		By("Delete redmine dogu crd")
		deleteDoguCr(ctx, redmineCr, false)

		Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(EtcdDoguRegistry.AssertExpectations(mockeryT)).To(BeTrue())
	})

	It("Setup mocks and test data for upgrade", func() {
		// create mocks
		*DoguRemoteRegistryMock = extMocks.RemoteRegistry{}
		DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.48-4").Once().Return(upgradeLdapFromDoguDescriptor, nil)
		DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.49-1").Once().Return(upgradeLdapToDoguDescriptor, nil)

		*ImageRegistryMock = mocks.ImageRegistry{}
		ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.48-4").Return(imageConfig, nil)
		ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.49-1").Return(imageConfig, nil)

		*EtcdDoguRegistry = extMocks.DoguRegistry{}
		EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(false, nil)
		EtcdDoguRegistry.Mock.On("Register", upgradeLdapFromDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Enable", upgradeLdapFromDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Get", "ldap").Return(upgradeLdapFromDoguDescriptor, nil)

		EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(true, nil)
		EtcdDoguRegistry.Mock.On("Register", upgradeLdapToDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Enable", upgradeLdapToDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Unregister", "ldap").Return(nil)

		CommandExecutor.
			On("ExecCommandForPod", mock.Anything, mock.Anything, exec.NewShellCommand("/bin/cp", "/pre-upgrade.sh", "/tmp/dogu-reserved"), cloudogu.ContainersStarted).Once().Return(&bytes.Buffer{}, nil).
			On("ExecCommandForPod", mock.Anything, mock.Anything, exec.NewShellCommand("/tmp/dogu-reserved/pre-upgrade.sh", "2.4.48-4", "2.4.49-1"), cloudogu.PodReady).Once().Return(&bytes.Buffer{}, nil).
			On("ExecCommandForDogu", mock.Anything, mock.Anything, exec.NewShellCommand("/post-upgrade.sh", "2.4.48-4", "2.4.49-1"), cloudogu.ContainersStarted).Once().Run(func(args mock.Arguments) {
			defer GinkgoRecover()
			assertNewDeploymentVersionWithStartupProbe(upgradeLdapFromDoguLookupKey, ldapToVersion, 1080)
			assertRessourceStatus(upgradeLdapFromDoguLookupKey, "upgrading")
		}).Return(&bytes.Buffer{}, nil)
	})

	It("Should upgrade dogu in cluster", func() {
		namespace := &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: upgradeNamespace, Namespace: upgradeNamespace}}
		_ = k8sClient.Create(ctx, namespace)

		By("Install ldap dogu resource in version 2.4.48-4")
		installDoguCr(testCtx, upgradeLdapFromCr)

		By("Expect created dogu")
		installedLdapDoguCr := &k8sv1.Dogu{}
		Eventually(func() bool {
			err := k8sClient.Get(testCtx, upgradeLdapFromDoguLookupKey, installedLdapDoguCr)
			if err != nil {
				return false
			}

			finalizers := installedLdapDoguCr.Finalizers
			if len(finalizers) == 1 && finalizers[0] == "dogu-finalizer" {
				return true
			}

			return false
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		setExecPodRunning(ctx, "ldap")

		By("Wait for resources created deployment")
		deployment := new(appsv1.Deployment)
		Eventually(func() bool { return getObjectFromCluster(testCtx, deployment, upgradeLdapFromDoguLookupKey) }, TimeoutInterval, PollingInterval).Should(BeTrue())
		Eventually(func() bool { return getObjectFromCluster(testCtx, &corev1.Service{}, upgradeLdapFromDoguLookupKey) }, TimeoutInterval, PollingInterval).Should(BeTrue())
		Eventually(func() bool {
			return getObjectFromCluster(testCtx, &corev1.PersistentVolumeClaim{}, upgradeLdapFromDoguLookupKey)
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		secretLookupKey := types.NamespacedName{Name: upgradeLdapFromDoguLookupKey.Name + "-private", Namespace: upgradeLdapFromDoguLookupKey.Namespace}
		Eventually(func() bool { return getObjectFromCluster(testCtx, &corev1.Secret{}, secretLookupKey) }, TimeoutInterval, PollingInterval).Should(BeTrue())

		assertRessourceStatus(upgradeLdapFromDoguLookupKey, "installed")

		By("Patch Deployment to contain at least one healthy replica")
		Expect(func() bool {
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			err := k8sClient.Status().Update(testCtx, deployment)
			return err == nil
		}()).To(BeTrue())

		By("Update dogu resource with new version")
		Expect(func() bool {
			return getObjectFromCluster(ctx, installedLdapDoguCr, upgradeLdapFromDoguLookupKey)
		}()).To(BeTrue())

		upgradedLdapDoguCr := installedLdapDoguCr
		oldPodLabels := upgradedLdapDoguCr.GetPodLabels()
		Expect(func() bool {
			upgradedLdapDoguCr.Spec.Version = ldapToVersion
			err := k8sClient.Update(testCtx, upgradedLdapDoguCr)
			return err == nil
		}()).To(BeTrue())

		// key take away: We must take all unmocked pod interactions in our own hands because here is no deployment controller
		setExecPodRunning(ctx, "ldap")
		createDoguPod(ctx, upgradedLdapDoguCr, oldPodLabels)

		assertNewDeploymentVersionWithStartupProbe(upgradeLdapFromDoguLookupKey, ldapToVersion, 180)

		assertRessourceStatus(upgradeLdapFromDoguLookupKey, "installed")

		deleteDoguCr(ctx, upgradedLdapDoguCr, true)

		Expect(CommandExecutor.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(EtcdDoguRegistry.AssertExpectations(mockeryT)).To(BeTrue())
	})
})

func assertRessourceStatus(ressourceLookupKey types.NamespacedName, expectedStatus string) {
	By("Verify dogu ressource is " + expectedStatus)
	Eventually(func() string {
		actualResource := &k8sv1.Dogu{}
		ok := getObjectFromCluster(testCtx, actualResource, ressourceLookupKey)
		if ok {
			return actualResource.Status.Status
		}
		return "resource not found"
	}, TimeoutInterval, PollingInterval).Should(Equal(expectedStatus))
}

func assertNewDeploymentVersionWithStartupProbe(doguLookupKey types.NamespacedName, doguVersion string, expectedStartupProbe int) {
	By("Check new image in deployment")
	deploymentAfterUpgrading := new(appsv1.Deployment)
	Eventually(func() string {
		ok := getObjectFromCluster(testCtx, deploymentAfterUpgrading, doguLookupKey)
		if ok {
			return deploymentAfterUpgrading.Spec.Template.Spec.Containers[0].Image
		}
		return "resource not found"
	}, TimeoutInterval, PollingInterval).Should(ContainSubstring(doguVersion))

	By("Check startup probe failure threshold in deployment")
	Expect(int32(expectedStartupProbe)).To(Equal(deploymentAfterUpgrading.Spec.Template.Spec.Containers[0].StartupProbe.FailureThreshold))
}

// setExecPodRunning can be necessary because the environment has no controllers to really start the pods,
// therefore the dogu controller waits until timeout.
func setExecPodRunning(ctx context.Context, doguName string) {
	By("Simulate ExecPod is running")
	podList := &corev1.PodList{}

	Eventually(func() bool {
		err := k8sClient.List(ctx, podList)
		if err != nil {
			return false
		}
		for _, pod := range podList.Items {
			if strings.Contains(pod.Name, doguName+"-execpod") && pod.Status.Phase != corev1.PodRunning {
				pod.Status.Phase = corev1.PodRunning
				err := k8sClient.Status().Update(ctx, &pod)

				return err == nil
			}
		}
		return false
	}, TimeoutInterval, PollingInterval).Should(BeTrue())
}

// createDoguPod can be necessary because the environment has no controllers to really create the pods,
// therefore the dogu controller waits until timeout.
func createDoguPod(ctx context.Context, doguCr *k8sv1.Dogu, podLabels k8sv1.CesMatchingLabels) {
	By("Simulate dogu pod creation by deployment controller")
	doguPod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      doguCr.Name,
			Namespace: doguCr.Namespace,
			Labels:    podLabels,
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "asdf", Image: "ldap-image"}}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}},
		},
	}

	Expect(k8sClient.Create(ctx, doguPod)).Should(Succeed())
}

func installDoguCr(ctx context.Context, doguCr *k8sv1.Dogu) {
	doguClient := k8sClientSet.Dogus(doguCr.Namespace)
	_, err := doguClient.Create(ctx, doguCr, v1.CreateOptions{})
	Expect(err).Should(Succeed())
}

func updateDoguCr(ctx context.Context, doguCr *k8sv1.Dogu) {
	doguClient := k8sClientSet.Dogus(doguCr.Namespace)
	_, err := doguClient.Update(ctx, doguCr, v1.UpdateOptions{})
	Expect(err).Should(Succeed())
}

func deleteDoguCr(ctx context.Context, doguCr *k8sv1.Dogu, deleteAdditional bool) {
	doguClient := k8sClientSet.Dogus(doguCr.Namespace)
	err := doguClient.Delete(ctx, doguCr.Name, v1.DeleteOptions{})
	Expect(err).Should(Succeed())

	Eventually(func() bool {
		_, err := doguClient.Get(ctx, doguCr.Name, v1.GetOptions{})
		return apierrors.IsNotFound(err)
	}, TimeoutInterval, PollingInterval).Should(BeTrue())

	if !deleteAdditional {
		return
	}

	// For now, this is obsolete because our pseudocluster cannot delete stuff.
	// We will keep it here anyway, for when we migrate these tests to a real cluster.
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &appsv1.Deployment{})
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &corev1.Service{})
	deleteObjectFromCluster(ctx, types.NamespacedName{Name: doguCr.GetPrivateKeySecretName(), Namespace: doguCr.Namespace}, &corev1.Secret{})
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &corev1.PersistentVolumeClaim{})
}

func deleteObjectFromCluster(ctx context.Context, objectKey client.ObjectKey, deleteType client.Object) {
	Eventually(func() bool {
		ok := getObjectFromCluster(ctx, deleteType, objectKey)

		if !ok {
			return false
		}

		err := k8sClient.Delete(ctx, deleteType)
		return err == nil
	}, TimeoutInterval, PollingInterval).Should(BeTrue())
}

func getObjectFromCluster(ctx context.Context, objectType client.Object, lookupKey types.NamespacedName) bool {
	err := k8sClient.Get(ctx, lookupKey, objectType)
	return err == nil
}

// verifyOwner checks if the objectmetadata has a specific owner. This method should be used to verify that a dogu is
// the owner of every related resource. This replaces an integration test for the deletion of dogu related resources.
// In a real cluster resources without an owner will be garbage collected. In this environment the resources still exist
// after dogu deletion
func verifyOwner(name string, obj v1.ObjectMeta) bool {
	ownerRefs := obj.OwnerReferences
	if len(ownerRefs) == 1 && ownerRefs[0].Name == name {
		return true
	}

	return false
}
