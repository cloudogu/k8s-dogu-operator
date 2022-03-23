//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Dogu Controller", func() {

	const timeout = time.Second * 10
	const interval = time.Second * 1
	const doguName = "testdogu"
	const namespace = "default"
	ctx := context.TODO()
	doguLookupKey := types.NamespacedName{Name: doguName, Namespace: namespace}

	Context("Handle new dogu resource", func() {

		It("Should install dogu in cluster", func() {
			newDogu := &k8sv1.Dogu{
				TypeMeta: metav1.TypeMeta{
					Kind: "Dogu",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      doguName,
					Namespace: namespace,
				},
				Spec: k8sv1.DoguSpec{Name: doguName, Version: "1.0.0"},
			}

			ImageRegistryMock = mocks.ImageRegistry{}
			ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			DoguRegistryMock = mocks.DoguRegistry{}
			doguJson := &core.Dogu{
				Image:   "image",
				Version: "1.0.0",
				ExposedPorts: []core.ExposedPort{{
					Type:      "tcp",
					Container: 2222,
					Host:      4444,
				}},
			}
			DoguRegistryMock.Mock.On("GetDogu", mock.Anything).Return(doguJson, nil)
			DoguRegistratorMock.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			By("Creating dogu resource")
			Expect(k8sClient.Create(ctx, newDogu)).Should(Succeed())

			By("Expect created dogu")
			createdDogu := &k8sv1.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, createdDogu)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Expect created deployment")
			deployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, deployment)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(doguName).To(Equal(deployment.Name))
			Expect(namespace).To(Equal(deployment.Namespace))

			By("Expect created service")
			service := &corev1.Service{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, doguLookupKey, service)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

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
		})
	})
})
