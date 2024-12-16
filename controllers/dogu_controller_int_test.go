//go:build k8s_integration

package controllers

import (
	"context"
	_ "embed"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
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
	ldapQualifiedName, _ := cescommons.NewQualifiedName("official", "ldap")
	redmineCr := readDoguCr(t, redmineCrBytes)
	redmineQualifiedName, _ := cescommons.NewQualifiedName("official", "redmine")
	imageConfig := readImageConfig(t, imageConfigBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	redmineDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)

	ldapCr.Namespace = testNamespace
	ldapCr.ResourceVersion = ""
	ldapDoguLookupKey := types.NamespacedName{Name: ldapCr.Name, Namespace: ldapCr.Namespace}
	cesLoadbalancerLookupKey := types.NamespacedName{Name: "ces-loadbalancer", Namespace: testNamespace}
	tcpExposedPortsLookupKey := types.NamespacedName{Name: "tcp-services", Namespace: testNamespace}

	redmineCr.Namespace = testNamespace
	redmineCr.ResourceVersion = ""

	ctx := context.TODO()

	// Upgrade testdata
	upgradeLdapToDoguDescriptor := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	ldapToVersion := "2.4.49-1"
	upgradeLdapToDoguDescriptor.Version = ldapToVersion

	Context("Handle dogu resource", func() {
		It("Setup mocks and test data", func() {
			*DoguInterfaceMock = mockDoguInterface{}
			DoguInterfaceMock.EXPECT().UpdateStatusWithRetry(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(
				func(ctx context.Context, dogu *k8sv2.Dogu, modifyStatusFn func(k8sv2.DoguStatus) k8sv2.DoguStatus, opts metav1.UpdateOptions) {
					modifyStatusFn(dogu.Status)
				}).Once()
			*ImageRegistryMock = mockImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil).Once()
			*DoguRemoteRegistryMock = mockRemoteDoguDescriptorRepository{}
			ldapVersion, _ := core.ParseVersion("2.4.48-4")
			ldapQualifiedVersion, _ := cescommons.NewQualifiedVersion(ldapQualifiedName, ldapVersion)
			DoguRemoteRegistryMock.EXPECT().Get(mock.Anything, ldapQualifiedVersion).Return(ldapDogu, nil).Once()
			redmineVersion, _ := core.ParseVersion("4.2.3-10")
			redmineQualifiedVersion, _ := cescommons.NewQualifiedVersion(redmineQualifiedName, redmineVersion)
			DoguRemoteRegistryMock.EXPECT().Get(mock.Anything, redmineQualifiedVersion).Return(redmineDogu, nil).Once()
		})

		It("Should install dogu in cluster", func() {
			By("Creating namespace")
			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Namespace: testNamespace}}
			err := k8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Create dogu health state config map")
			healthCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "k8s-dogu-operator-dogu-health",
					Namespace: testNamespace,
				},
			}
			_, err = k8sClientSet.CoreV1().ConfigMaps(testNamespace).Create(ctx, healthCM, metav1.CreateOptions{})
			if err != nil {
				panic(err)
			}

			By("Creating dogu resource")
			installDoguCr(ctx, ldapCr)

			By("Expect created dogu")
			createdDogu := &k8sv2.Dogu{}

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
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

			setExecPodRunning(ctx, "ldap")

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, deployment)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, deployment.ObjectMeta)
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
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
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(service.Name))
			Expect(ldapCr.Namespace).To(Equal(service.Namespace))

			By("Expect created dogu pvc")
			doguPvc := &corev1.PersistentVolumeClaim{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, doguPvc)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, doguPvc.ObjectMeta)
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(doguPvc.Name))
			Expect(ldapCr.Namespace).To(Equal(doguPvc.Namespace))
			Expect(resource.MustParse("2Gi")).To(Equal(*doguPvc.Spec.Resources.Requests.Storage()))

			By("Expect dogu status to be installed")
			dogu := &k8sv2.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, dogu)
				if err != nil {
					return false
				}
				status := dogu.Status.Status
				return status == k8sv2.DoguStatusInstalled
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

			setDeploymentAvailable(ctx, ldapDoguLookupKey.Name)
			checkDoguAvailable(ctx, ldapDoguLookupKey.Name)
		})

		It("Update dogus additional ingress annotations", func() {
			By("Update dogu resource with ingress annotations")
			createdDogu := &k8sv2.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

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
				if exists && s == "{\"new\":\"new\"}" {
					return true
				}

				return false
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
		})

		It("Set dogu in support mode", func() {
			By("Update dogu resource with support mode true")
			createdDogu := &k8sv2.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

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
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
		})

		It("Should unset dogu support mode", func() {
			By("Update dogu resource with support mode false")
			createdDogu := &k8sv2.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

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
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
		})

		// This test does not work because the client in this environment can't even update the storage request of a pvc.
		// It can be used in planned future environments with real clusters.
		//
		// It("Should resize dogu volume", func() {
		// 	By("Update dogu resource with dataVolumeSize")
		// 	createdDogu := &k8sv2.Dogu{}
		// 	Eventually(func() bool {
		// 		err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
		// 		return err == nil
		// 	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
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
		// 	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
		// })

		It("Setup mocks and test data for upgrade", func() {
			// create mocks
			*DoguRemoteRegistryMock = mockRemoteDoguDescriptorRepository{}
			ldapVersion, _ := core.ParseVersion("2.4.49-1")
			ldapQualifiedVersion, _ := cescommons.NewQualifiedVersion(ldapQualifiedName, ldapVersion)
			DoguRemoteRegistryMock.EXPECT().Get(mock.Anything, ldapQualifiedVersion).Once().Return(upgradeLdapToDoguDescriptor, nil)

			*ImageRegistryMock = mockImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.49-1").Return(imageConfig, nil).Once()
		})

		It("Should upgrade dogu in cluster", func() {
			setDeploymentAvailable(ctx, ldapDoguLookupKey.Name)
			checkDoguAvailable(ctx, ldapDoguLookupKey.Name)

			createdDogu := &k8sv2.Dogu{}

			By("Update dogu resource with new version")
			Expect(func() bool {
				return getObjectFromCluster(ctx, createdDogu, ldapDoguLookupKey)
			}()).To(BeTrue())

			upgradedLdapDoguCr := createdDogu
			oldPodLabels := upgradedLdapDoguCr.GetPodLabels()
			Expect(func() bool {
				upgradedLdapDoguCr.Spec.Version = ldapToVersion
				err := k8sClient.Update(testCtx, upgradedLdapDoguCr)
				return err == nil
			}()).To(BeTrue())

			// key take away: We must take all unmocked pod interactions in our own hands because here is no deployment controller
			setExecPodRunning(ctx, "ldap")
			createDoguPod(ctx, upgradedLdapDoguCr, oldPodLabels)

			assertNewDeploymentVersionWithStartupProbe(ldapDoguLookupKey, ldapToVersion, 180)

			assertRessourceStatus(ldapDoguLookupKey, "installed")

			Expect(CommandExecutorMock.AssertExpectations(mockeryT)).To(BeTrue())
			Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
			Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		})

		It("Setup mocks and test data for delete", func() {
			// create mocks
			*DoguRemoteRegistryMock = mockRemoteDoguDescriptorRepository{}
			*ImageRegistryMock = mockImageRegistry{}
		})

		It("Should delete dogu", func() {
			deleteDoguCr(ctx, ldapCr, true)

			By("LoadBalancer service should be deleted")
			lbService := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cesLoadbalancerLookupKey, lbService)
				return apierrors.IsNotFound(err)
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

			By("Expected deleted entries in tcp/udp configmap")
			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, tcpExposedPortsLookupKey, cm)
				if err != nil && len(cm.Data) != 0 {
					return false
				}

				return true
			}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

			Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
			Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		})
	})

	// Fails sporadically. A cluster test did not show this behavior.
	// The test needs to be analyzed in more detail as to why there is a problem here.
	//
	//It("Should fail dogu installation as dependency is missing", func() {
	//	By("Creating redmine dogu resource")
	//	installDoguCr(ctx, redmineCr)
	//
	//	By("Check for failed installation and check events of dogu resource")
	//	createdDogu := &k8sv2.Dogu{}
	//
	//	Eventually(func() bool {
	//		err := k8sClient.Get(ctx, redmineCr.GetObjectKey(), createdDogu)
	//		if err != nil {
	//			return false
	//		}
	//		if createdDogu.Status.Status != k8sv2.DoguStatusNotInstalled {
	//			return false
	//		}
	//
	//		eventList := &corev1.EventList{}
	//		err = k8sClient.List(ctx, eventList, &client.ListOptions{})
	//		if err != nil {
	//			return false
	//		}
	//
	//		count := 0
	//		for _, item := range eventList.Items {
	//			if item.InvolvedObject.Name == createdDogu.Name && item.Reason == ErrorOnInstallEventReason {
	//				count++
	//			}
	//		}
	//
	//		return count == 1
	//	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
	//
	//	By("Delete redmine dogu crd")
	//	deleteDoguCr(ctx, redmineCr, false)
	//
	//	Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
	//	Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
	//	Expect(EtcdDoguRegistry.AssertExpectations(mockeryT)).To(BeTrue())
	//})
})

func assertRessourceStatus(ressourceLookupKey types.NamespacedName, expectedStatus string) {
	By("Verify dogu ressource is " + expectedStatus)
	Eventually(func() string {
		actualResource := &k8sv2.Dogu{}
		ok := getObjectFromCluster(testCtx, actualResource, ressourceLookupKey)
		if ok {
			return actualResource.Status.Status
		}
		return "resource not found"
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(Equal(expectedStatus))
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
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(ContainSubstring(doguVersion))

	By("Check startup probe failure threshold in deployment")
	Expect(int32(expectedStartupProbe)).To(Equal(deploymentAfterUpgrading.Spec.Template.Spec.Containers[0].StartupProbe.FailureThreshold))
}

// setExecPodRunning can be necessary because the environment has no controllers to really start the pods,
// therefore the dogu controller waits until timeout.
func setExecPodRunning(ctx context.Context, doguName string) {
	By("Simulate execPod is running")
	podList := &corev1.PodList{}

	Eventually(func() bool {
		err := k8sClient.List(ctx, podList)
		Expect(err).ToNot(HaveOccurred())
		for _, pod := range podList.Items {
			if strings.Contains(pod.Name, doguName+"-execpod") && pod.Status.Phase != corev1.PodRunning {
				pod.Status.Phase = corev1.PodRunning
				err := k8sClient.Status().Update(ctx, &pod)

				return err == nil
			}
		}
		return false
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
}

func setDeploymentAvailable(ctx context.Context, doguName string) {
	By("Set Deployment to be available")
	Eventually(func() error {
		deployment, err := k8sClientSet.AppsV1().Deployments(testNamespace).Get(ctx, doguName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		var replicas int32 = 1
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}
		deployment.Status.Replicas = replicas
		deployment.Status.UpdatedReplicas = replicas
		deployment.Status.ReadyReplicas = replicas
		deployment.Status.AvailableReplicas = replicas

		_, err = k8sClientSet.AppsV1().Deployments(testNamespace).UpdateStatus(ctx, deployment, metav1.UpdateOptions{})
		return err
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).ShouldNot(HaveOccurred())
}

func checkDoguAvailable(ctx context.Context, doguName string) {
	By("Expect dogu to be available")
	Eventually(func() bool {
		dogu, err := ecosystemClientSet.Dogus(testNamespace).Get(ctx, doguName, metav1.GetOptions{})
		if err != nil {
			return false
		}

		status := dogu.Status.Health
		return status == k8sv2.AvailableHealthStatus
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
}

// createDoguPod can be necessary because the environment has no controllers to really create the pods,
// therefore the dogu controller waits until timeout.
func createDoguPod(ctx context.Context, doguCr *k8sv2.Dogu, podLabels k8sv2.CesMatchingLabels) {
	By("Simulate dogu pod creation by deployment controller")
	doguPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
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

func installDoguCr(ctx context.Context, doguCr *k8sv2.Dogu) {
	doguClient := ecosystemClientSet.Dogus(doguCr.Namespace)
	_, err := doguClient.Create(ctx, doguCr, metav1.CreateOptions{})
	Expect(err).Should(Succeed())
}

func updateDoguCr(ctx context.Context, doguCr *k8sv2.Dogu) {
	doguClient := ecosystemClientSet.Dogus(doguCr.Namespace)
	_, err := doguClient.Update(ctx, doguCr, metav1.UpdateOptions{})
	Expect(err).Should(Succeed())
}

func deleteDoguCr(ctx context.Context, doguCr *k8sv2.Dogu, deleteAdditional bool) {
	doguClient := ecosystemClientSet.Dogus(doguCr.Namespace)
	err := doguClient.Delete(ctx, doguCr.Name, metav1.DeleteOptions{})
	Expect(err).Should(Succeed())

	Eventually(func() bool {
		_, err := doguClient.Get(ctx, doguCr.Name, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())

	if !deleteAdditional {
		return
	}

	// For now, this is obsolete because our pseudocluster cannot delete stuff.
	// We will keep it here anyway, for when we migrate these tests to a real cluster.
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &appsv1.Deployment{})
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &corev1.Service{})
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
	}).WithTimeout(TimeoutInterval).WithPolling(PollingInterval).Should(BeTrue())
}

func getObjectFromCluster(ctx context.Context, objectType client.Object, lookupKey types.NamespacedName) bool {
	err := k8sClient.Get(ctx, lookupKey, objectType)
	return err == nil
}

// verifyOwner checks if the objectmetadata has a specific owner. This method should be used to verify that a dogu is
// the owner of every related resource. This replaces an integration test for the deletion of dogu related resources.
// In a real cluster resources without an owner will be garbage collected. In this environment the resources still exist
// after dogu deletion
func verifyOwner(name string, obj metav1.ObjectMeta) bool {
	ownerRefs := obj.OwnerReferences
	if len(ownerRefs) == 1 && ownerRefs[0].Name == name {
		return true
	}

	return false
}
