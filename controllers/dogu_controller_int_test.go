//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	"fmt"
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
	"time"
)

var _ = Describe("Dogu Controller", func() {
	const timeoutInterval = time.Second * 10
	const pollingInterval = time.Second * 1
	ldapCr.Namespace = "default"
	ldapCr.ResourceVersion = ""
	doguName := ldapCr.Name
	namespace := ldapCr.Namespace
	ctx := context.TODO()
	doguLookupKey := types.NamespacedName{Name: doguName, Namespace: namespace}

	Context("Handle dogu resource", func() {
		It("Should install dogu in cluster", func() {

			ImageRegistryMock = mocks.ImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			DoguRegistryMock = mocks.DoguRegistry{}
			DoguRegistryMock.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)

			By("Creating dogu resource")
			Expect(k8sClient.Create(ctx, ldapCr)).Should(Succeed())

			By("Expect created dogu")
			createdDogu := &k8sv1.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, createdDogu)
				if err != nil {
					return false
				}
				finalizers := createdDogu.Finalizers
				if len(finalizers) == 1 && finalizers[0] == "dogu-finalizer" {
					return true
				}
				return false
			}, timeoutInterval, pollingInterval).Should(BeTrue())

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, deployment)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, deployment.ObjectMeta)
			}, timeoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(deployment.Name))
			Expect(namespace).To(Equal(deployment.Namespace))

			By("Expect created service")
			service := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, service)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, service.ObjectMeta)
			}, timeoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(service.Name))
			Expect(namespace).To(Equal(service.Namespace))

			By("Expect created secret")
			secret := &corev1.Secret{}
			secretLookupKey := types.NamespacedName{Name: doguName + "-private", Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, secret)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, secret.ObjectMeta)
			}, timeoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName + "-private").To(Equal(secret.Name))
			Expect(namespace).To(Equal(secret.Namespace))

			By("Expect created pvc")
			pvc := &corev1.PersistentVolumeClaim{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, pvc)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, pvc.ObjectMeta)
			}, timeoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(pvc.Name))
			Expect(namespace).To(Equal(pvc.Namespace))

			By("Expect exposed service for service port 2222")
			exposedService2222 := &corev1.Service{}
			exposedService2222Name := fmt.Sprintf("%s-exposed-2222", doguName)
			exposedService2222LookupKey := types.NamespacedName{Name: exposedService2222Name, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService2222LookupKey, exposedService2222)
				if err != nil {
					return false
				}
				return true
			}, pollingInterval, timeoutInterval).Should(BeTrue())

			By("Expect exposed service for service port 8888")
			exposedService8888 := &corev1.Service{}
			exposedService8888Name := fmt.Sprintf("%s-exposed-2222", doguName)
			exposedService8888LookupKey := types.NamespacedName{Name: exposedService8888Name, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedService8888LookupKey, exposedService8888)
				if err != nil {
					return false
				}
				return true
			}, pollingInterval, timeoutInterval).Should(BeTrue())

			Expect(exposedService8888.Name).To(Equal(exposedService8888Name))

			By("Delete Dogu")
			Expect(k8sClient.Delete(ctx, ldapCr)).Should(Succeed())

			dogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, dogu)
				return apierrors.IsNotFound(err)
			}, timeoutInterval, pollingInterval).Should(BeTrue())
		})
	})
})

// VerifyOwner checks if the objectmetadata has a specific owner. This Method should be used to verify that a dogu is
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
