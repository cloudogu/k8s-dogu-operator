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
	"testing"
)

var _ = Describe("Dogu Controller", func() {
	t := &testing.T{}
	ldapCr := readTestDataLdapCr(t)
	redmineCr := readTestDataRedmineCr(t)
	imageConfig := readTestDataImageConfig(t)
	ldapDogu := readTestDataLdapDogu(t)
	redmineDogu := readTestDataRedmineDogu(t)

	ldapCr.Namespace = "default"
	ldapCr.ResourceVersion = ""
	ldapDoguLookupKey := types.NamespacedName{Name: ldapCr.Name, Namespace: ldapCr.Namespace}

	redmineCr.Namespace = "default"
	redmineCr.ResourceVersion = ""
	redmineDoguLookupKey := types.NamespacedName{Name: redmineCr.Name, Namespace: redmineCr.Namespace}

	ctx := context.TODO()
	ImageRegistryMock = mocks.ImageRegistry{}
	ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
	DoguRemoteRegistryMock = cesremotemocks.Registry{}
	DoguRemoteRegistryMock.Mock.On("Get", "official/ldap").Return(ldapDogu, nil)
	DoguRemoteRegistryMock.Mock.On("Get", "official/redmine").Return(redmineDogu, nil)

	EtcdDoguRegistry = cesmocks.DoguRegistry{}
	EtcdDoguRegistry.Mock.On("Get", "postgresql").Return(nil, fmt.Errorf("not installed"))
	EtcdDoguRegistry.Mock.On("Get", "cas").Return(nil, fmt.Errorf("not installed"))
	EtcdDoguRegistry.Mock.On("Get", "nginx").Return(nil, fmt.Errorf("not installed"))
	EtcdDoguRegistry.Mock.On("Get", "postfix").Return(nil, fmt.Errorf("not installed"))
	EtcdDoguRegistry.Mock.On("Get", "ldap").Return(ldapDogu, nil)
	EtcdDoguRegistry.Mock.On("Get", "redmine").Return(redmineDogu, nil)
	EtcdDoguRegistry.Mock.On("Register", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("Unregister", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("Enable", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("IsEnabled", mock.Anything).Return(false, nil)

	Context("Handle dogu resource", func() {
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
			exposedService2222LookupKey := types.NamespacedName{Name: exposedService2222Name, Namespace: ldapCr.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService2222LookupKey, exposedService2222)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			By("Expect exposed service for service port 8888")
			exposedService8888 := &corev1.Service{}
			exposedService8888Name := fmt.Sprintf("%s-exposed-2222", ldapCr.Name)
			exposedService8888LookupKey := types.NamespacedName{Name: exposedService8888Name, Namespace: ldapCr.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService8888LookupKey, exposedService8888)
				return err == nil
			}, PollingInterval, TimeoutInterval).Should(BeTrue())

			Expect(exposedService8888.Name).To(Equal(exposedService8888Name))
		})

		It("Should delete dogu", func() {
			By("Delete Dogu")
			deleteDoguCrd(ctx, ldapCr, ldapDoguLookupKey)
		})

		It("Should fail dogu installation as dependency is missing", func() {
			By("Creating redmine dogu resource")
			installDoguCrd(ctx, redmineCr)

			By("Check for failed installation and check messages of dogu resource")
			createdDogu := &k8sv1.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, redmineDoguLookupKey, createdDogu)
				if err != nil {
					return false
				}
				if createdDogu.Status.Status != k8sv1.DoguStatusNotInstalled {
					return false
				}
				statusMessages := createdDogu.Status.StatusMessages
				if len(statusMessages) != 4 {
					return false
				}
				statusMessage := "failed to resolve dependency: {dogu postgresql }"
				if !containsStatusMessage(statusMessages, statusMessage) {
					return false
				}
				statusMessage = "failed to resolve dependency: {dogu cas }"
				if !containsStatusMessage(statusMessages, statusMessage) {
					return false
				}
				statusMessage = "failed to resolve dependency: {dogu postfix }"
				if !containsStatusMessage(statusMessages, statusMessage) {
					return false
				}
				return true
			}, TimeoutInterval, PollingInterval).Should(BeTrue())

			By("Delete redmine dogu crd")
			deleteDoguCrd(ctx, redmineCr, redmineDoguLookupKey)
		})
	})
})

func containsStatusMessage(messages []string, statusMessage string) bool {
	for _, msg := range messages {
		if msg == statusMessage {
			return true
		}
	}

	return false
}

func installDoguCrd(ctx context.Context, doguCr *k8sv1.Dogu) {
	Expect(k8sClient.Create(ctx, doguCr)).Should(Succeed())
}

func deleteDoguCrd(ctx context.Context, doguCr *k8sv1.Dogu, doguLookupKey types.NamespacedName) {
	Expect(k8sClient.Delete(ctx, doguCr)).Should(Succeed())

	dogu := &k8sv1.Dogu{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, doguLookupKey, dogu)
		return apierrors.IsNotFound(err)
	}, TimeoutInterval, PollingInterval).Should(BeTrue())
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
