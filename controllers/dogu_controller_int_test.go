//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	"github.com/cloudogu/cesapp/v4/core"
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

	const interval = time.Second * 10
	const timeout = time.Second * 1
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
			}, interval, timeout).Should(BeTrue())

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, deployment)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, deployment.ObjectMeta)
			}, interval, timeout).Should(BeTrue())
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
			}, interval, timeout).Should(BeTrue())
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
			}, interval, timeout).Should(BeTrue())
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
			}, interval, timeout).Should(BeTrue())
			Expect(doguName).To(Equal(pvc.Name))
			Expect(namespace).To(Equal(pvc.Namespace))

			By("Expect exposed service")
			exposedService := &corev1.Service{}
			exposedServiceName := fmt.Sprintf("%s-exposed", doguName)
			exposedServiceLookupKey := types.NamespacedName{Name: exposedServiceName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, exposedServiceLookupKey, exposedService)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(exposedService.Name).To(Equal(exposedServiceName))

			By("Delete Dogu")
			Expect(k8sClient.Delete(ctx, ldapCr)).Should(Succeed())

			dogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, dogu)
				return apierrors.IsNotFound(err)
			}, interval, timeout).Should(BeTrue())
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
