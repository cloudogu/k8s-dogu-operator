//go:build k8s_integration
// +build k8s_integration

package controllers

import (
	"context"
	_ "embed"
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
	"sigs.k8s.io/yaml"
	"time"
)

//go:embed testdata/ldap-descriptor-cm.yaml
var ldapDescriptorCmBytes []byte
var ldapDescriptorCm = &corev1.ConfigMap{}

func init() {
	err := yaml.Unmarshal(ldapDescriptorCmBytes, ldapDescriptorCm)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Dogu Controller", func() {

	const timoutInterval = time.Second * 10
	const pollingInterval = time.Second * 1
	ldapDescriptorCm.Namespace = "default"
	ldapCr.Namespace = "default"
	ldapCr.ResourceVersion = ""
	doguName := ldapCr.Name
	namespace := ldapCr.Namespace
	ctx := context.TODO()
	doguLookupKey := types.NamespacedName{Name: doguName, Namespace: namespace}
	ImageRegistryMock = mocks.ImageRegistry{}
	ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
	DoguRegistryMock = mocks.DoguRegistry{}
	DoguRegistryMock.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)

	var createdResources []client.Object

	deleteResourceAfterTest := func(o client.Object) {
		createdResources = append(createdResources, o)
	}

	BeforeEach(func() {
		createdResources = nil
	})

	AfterEach(func() {
		for _, resource := range createdResources {
			err := k8sClient.Delete(ctx, resource)
			Expect(err).To(Succeed())
		}
	})

	Context("Handle dogu resource", func() {
		It("Should install dogu in cluster", func() {
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
			}, timoutInterval, pollingInterval).Should(BeTrue())

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}
			deleteResourceAfterTest(deployment)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, deployment)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, deployment.ObjectMeta)
			}, timoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(deployment.Name))
			Expect(namespace).To(Equal(deployment.Namespace))

			By("Expect created service")
			service := &corev1.Service{}
			deleteResourceAfterTest(service)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, service)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, service.ObjectMeta)
			}, timoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(service.Name))
			Expect(namespace).To(Equal(service.Namespace))

			By("Expect created secret")
			secret := &corev1.Secret{}
			deleteResourceAfterTest(secret)
			secretLookupKey := types.NamespacedName{Name: doguName + "-private", Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, secret)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, secret.ObjectMeta)
			}, timoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName + "-private").To(Equal(secret.Name))
			Expect(namespace).To(Equal(secret.Namespace))

			By("Expect created pvc")
			pvc := &corev1.PersistentVolumeClaim{}
			deleteResourceAfterTest(pvc)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, pvc)
				if err != nil {
					return false
				}
				return verifyOwner(createdDogu.Name, pvc.ObjectMeta)
			}, timoutInterval, pollingInterval).Should(BeTrue())
			Expect(doguName).To(Equal(pvc.Name))
			Expect(namespace).To(Equal(pvc.Namespace))
		})

		It("Should delete dogu", func() {
			By("Delete Dogu")
			Expect(k8sClient.Delete(ctx, ldapCr)).Should(Succeed())

			dogu := &k8sv1.Dogu{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, dogu)
				return apierrors.IsNotFound(err)
			}, timoutInterval, pollingInterval).Should(BeTrue())
		})
	})
})

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
