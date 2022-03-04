package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"time"
)

var _ = Describe("Dogu Controller", func() {

	const timeout = time.Second * 10
	const interval = time.Second * 1
	const doguName = "testdogu"
	const namespace = "default"

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

			By("Creating dogu resource")
			Expect(k8sClient.Create(context.Background(), newDogu)).Should(Succeed())

			By("Expect created dogu")
			doguLookupKey := types.NamespacedName{Name: doguName, Namespace: namespace}
			createdDogu := &k8sv1.Dogu{}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), doguLookupKey, createdDogu)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Expect created deployment")
			deployments := &appsv1.DeploymentList{}

			Eventually(func() bool {
				err := k8sClient.List(context.Background(), deployments)
				if err != nil {
					return false
				}
				return len(deployments.Items) == 1
			}, timeout, interval).Should(BeTrue())

			By("Expect created service")
			services := &corev1.ServiceList{}

			Eventually(func() bool {
				err := k8sClient.List(context.Background(), services)
				if err != nil {
					return false
				}
				return len(services.Items) == 1
			}, timeout, interval).Should(BeTrue())
		})
	})

})
