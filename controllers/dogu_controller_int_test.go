//go:build k8s_integration
// +build k8s_integration

package controllers

import (
	"context"
	_ "embed"
	"fmt"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
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

	redmineCr.Namespace = "default"
	redmineCr.ResourceVersion = ""

	ctx := context.TODO()
	var exposedService2222LookupKey types.NamespacedName
	var exposedService8888LookupKey types.NamespacedName

	// Upgrade testdata
	namespace := "default"
	ldapFromCr := readDoguCr(t, ldapCrBytes)
	ldapFromCr.ResourceVersion = ""
	ldapFromCr.Namespace = namespace
	ldapFromDoguDescriptor := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	ldapFromDoguLookupKey := types.NamespacedName{Name: ldapFromCr.Name, Namespace: namespace}
	ldapToDoguDescriptor := readDoguDescriptor(t, ldapUpgradeDoguDescriptorBytes)
	ldapToVersion := ldapToDoguDescriptor.Version

	Context("Handle dogu resource", func() {
		It("Setup mocks and test data", func() {
			*ImageRegistryMock = mocks.ImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			*DoguRemoteRegistryMock = cesremotemocks.Registry{}
			DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.48-4").Return(ldapDogu, nil)
			DoguRemoteRegistryMock.Mock.On("GetVersion", "official/redmine", "4.2.3-10").Return(redmineDogu, nil)

			*EtcdDoguRegistry = cesmocks.DoguRegistry{}
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
			installDoguCrd(ctx, ldapCr)

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

			By("Expect created pvc")
			pvc := &corev1.PersistentVolumeClaim{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, pvc)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, pvc.ObjectMeta)
			}, TimeoutInterval, PollingInterval).Should(BeTrue())
			Expect(ldapCr.Name).To(Equal(pvc.Name))
			Expect(ldapCr.Namespace).To(Equal(pvc.Namespace))

			By("Expect exposed service for service port 2222")
			exposedService2222 := &corev1.Service{}
			exposedService2222Name := fmt.Sprintf("%s-exposed-2222", ldapCr.Name)
			exposedService2222LookupKey = types.NamespacedName{Name: exposedService2222Name, Namespace: ldapCr.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService2222LookupKey, exposedService2222)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			By("Expect exposed service for service port 8888")
			exposedService8888 := &corev1.Service{}
			exposedService8888Name := fmt.Sprintf("%s-exposed-8888", ldapCr.Name)
			exposedService8888LookupKey = types.NamespacedName{Name: exposedService8888Name, Namespace: ldapCr.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService8888LookupKey, exposedService8888)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			Expect(exposedService8888.Name).To(Equal(exposedService8888Name))
		})

		It("Set dogu in support mode", func() {
			By("Update dogu resource with support mode true")
			createdDogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ldapDoguLookupKey, createdDogu)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			createdDogu.Spec.SupportMode = true
			updateDoguCrd(ctx, createdDogu)

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
			updateDoguCrd(ctx, createdDogu)

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

		It("Should delete dogu", func() {
			deleteDoguCrd(ctx, ldapCr, ldapDoguLookupKey, true)
			deleteObjectFromCluster(ctx, exposedService8888LookupKey, &corev1.Service{})
			deleteObjectFromCluster(ctx, exposedService2222LookupKey, &corev1.Service{})
		})
	})

	It("Should fail dogu installation as dependency is missing", func() {
		By("Creating redmine dogu resource")
		installDoguCrd(ctx, redmineCr)

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

			if count != 1 {
				return false
			}

			return true
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		By("Delete redmine dogu crd")
		deleteDoguCrd(ctx, redmineCr, redmineCr.GetObjectKey(), false)

		Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(EtcdDoguRegistry.AssertExpectations(mockeryT)).To(BeTrue())
	})

	It("Setup mocks and test data for upgrade", func() {
		// create mocks
		*DoguRemoteRegistryMock = cesremotemocks.Registry{}
		DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.48-4").Once().Return(ldapFromDoguDescriptor, nil)
		DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.49-1").Once().Return(ldapToDoguDescriptor, nil)

		*ImageRegistryMock = mocks.ImageRegistry{}
		ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.48-4").Return(imageConfig, nil)
		ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.49-1").Return(imageConfig, nil)

		*EtcdDoguRegistry = cesmocks.DoguRegistry{}
		EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(false, nil)
		EtcdDoguRegistry.Mock.On("Register", ldapFromDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Enable", ldapFromDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Get", "ldap").Once().Return(ldapFromDoguDescriptor, nil)

		EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(true, nil)
		EtcdDoguRegistry.Mock.On("Register", ldapToDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Enable", ldapToDoguDescriptor).Once().Return(nil)
		EtcdDoguRegistry.Mock.On("Get", "ldap").Return(ldapToDoguDescriptor, nil)
		EtcdDoguRegistry.Mock.On("Unregister", "ldap").Return(nil)
	})

	It("Should upgrade dogu in cluster", func() {
		By("Install ldap dogu resource in version 2.4.48-4")
		installDoguCrd(testCtx, ldapFromCr)

		By("Expect created dogu")
		installedLdapDoguCr := &k8sv1.Dogu{}
		Eventually(func() bool {
			err := k8sClient.Get(testCtx, ldapFromDoguLookupKey, installedLdapDoguCr)
			if err != nil {
				return false
			}

			finalizers := installedLdapDoguCr.Finalizers
			if len(finalizers) == 1 && finalizers[0] == "dogu-finalizer" {
				return true
			}

			return false
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		By("Wait for resources created deployment")
		deployment := new(appsv1.Deployment)
		Eventually(getObjectFromCluster(testCtx, deployment, ldapFromDoguLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())
		Eventually(getObjectFromCluster(testCtx, &corev1.Service{}, ldapFromDoguLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())
		Eventually(getObjectFromCluster(testCtx, &corev1.PersistentVolumeClaim{}, ldapFromDoguLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())

		secretLookupKey := types.NamespacedName{Name: ldapFromDoguLookupKey.Name + "-private", Namespace: ldapFromDoguLookupKey.Namespace}
		Eventually(getObjectFromCluster(testCtx, &corev1.Secret{}, secretLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())

		By("Patch Deployment to contain at least one healthy replica")
		Expect(func() bool {
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			err := k8sClient.Status().Update(testCtx, deployment)
			return err == nil
		}()).To(BeTrue())

		By("Update dogu crd with new version")
		Expect(func() bool {
			installedLdapDoguCr.Spec.Version = ldapToVersion
			err := k8sClient.Update(testCtx, installedLdapDoguCr)
			return err == nil
		}()).To(BeTrue())

		By("Check new image in deployment")
		Eventually(func() bool {
			deploymentAfterUpgrading := new(appsv1.Deployment)
			ok := getObjectFromCluster(testCtx, deploymentAfterUpgrading, ldapFromDoguLookupKey)
			return ok && strings.Contains(deploymentAfterUpgrading.Spec.Template.Spec.Containers[0].Image, ldapToVersion)
		}, TimeoutInterval, PollingInterval).Should(BeTrue())

		deleteDoguCrd(ctx, installedLdapDoguCr, ldapFromDoguLookupKey, true)

		Expect(DoguRemoteRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(ImageRegistryMock.AssertExpectations(mockeryT)).To(BeTrue())
		Expect(EtcdDoguRegistry.AssertExpectations(mockeryT)).To(BeTrue())
	})

})

func isDeploymentInSupportMode(deployment *appsv1.Deployment) bool {
	container := deployment.Spec.Template.Spec.Containers[0]
	envVars := container.Env
	envVarFound := false
	for _, env := range envVars {
		if env.Name == "SUPPORT_MODE" && env.Value == "true" {
			envVarFound = true
		}
	}

	if hasSleepCommand(container) && hasNoProbes(container) && envVarFound {
		return true
	}

	return false
}

func hasSleepCommand(container corev1.Container) bool {
	command := container.Command
	if len(command) != 3 {
		return false
	}
	if command[0] != "/bin/bash" || command[1] != "-c" || command[2] != "--" {
		return false
	}
	args := container.Args
	if len(args) != 1 {
		return false
	}
	if args[0] != "while true; do sleep 5; done;" {
		return false
	}

	return true
}

func hasNoProbes(container corev1.Container) bool {
	if container.StartupProbe == nil && container.LivenessProbe == nil && container.ReadinessProbe == nil {
		return true
	}

	return false
}

func installDoguCrd(ctx context.Context, doguCr *k8sv1.Dogu) {
	Expect(k8sClient.Create(ctx, doguCr)).Should(Succeed())
}

func updateDoguCrd(ctx context.Context, doguCr *k8sv1.Dogu) {
	Expect(k8sClient.Update(ctx, doguCr)).Should(Succeed())
}

func deleteDoguCrd(ctx context.Context, doguCr *k8sv1.Dogu, doguLookupKey types.NamespacedName, deleteAdditional bool) {
	Expect(k8sClient.Delete(ctx, doguCr)).Should(Succeed())

	dogu := &k8sv1.Dogu{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, doguLookupKey, dogu)
		return apierrors.IsNotFound(err)
	}, TimeoutInterval, PollingInterval).Should(BeTrue())

	if !deleteAdditional {
		return
	}

	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &appsv1.Deployment{})
	deleteObjectFromCluster(ctx, doguCr.GetObjectKey(), &corev1.Service{})
	deleteObjectFromCluster(ctx, types.NamespacedName{Name: doguCr.GetPrivateVolumeName(), Namespace: doguCr.Namespace}, &corev1.Secret{})
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
