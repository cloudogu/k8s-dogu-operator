//go:build k8s_integration
// +build k8s_integration

package controllers

import (
	"context"
	_ "embed"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Dogu Upgrade Tests", func() {
	t := &testing.T{}
	namespace := "default"

	// create mocks
	DoguRemoteRegistryMock = cesremotemocks.Registry{}
	ImageRegistryMock = mocks.ImageRegistry{}
	t.Cleanup(func() { ImageRegistryMock.AssertExpectations(t) })
	EtcdDoguRegistry = cesmocks.DoguRegistry{}
	t.Cleanup(func() { EtcdDoguRegistry.AssertExpectations(t) })

	// configure image configuration mock
	imageConfig := readImageConfig(t, imageConfigBytes)

	// configure mocks for installed ldap version
	ldapFromCr := readDoguCr(t, ldapCrBytes)
	ldapFromCr.ResourceVersion = ""
	ldapFromCr.Namespace = namespace
	ldapFromDoguDescriptor := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	ldapFromDoguLookupKey := types.NamespacedName{Name: ldapFromCr.Name, Namespace: namespace}
	DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.48-4").Once().Return(ldapFromDoguDescriptor, nil)
	EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(false, nil)
	EtcdDoguRegistry.Mock.On("Register", ldapFromDoguDescriptor).Once().Return(nil)
	EtcdDoguRegistry.Mock.On("Enable", ldapFromDoguDescriptor).Once().Return(nil)
	ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.48-4").Return(imageConfig, nil)
	EtcdDoguRegistry.Mock.On("Get", "ldap").Once().Return(ldapFromDoguDescriptor, nil)

	// configure mocks for upgraded ldap version
	ldapToDoguDescriptor := readDoguDescriptor(t, ldapUpgradeDoguDescriptorBytes)
	ldapToVersion := ldapToDoguDescriptor.Version
	DoguRemoteRegistryMock.Mock.On("GetVersion", "official/ldap", "2.4.49-1").Once().Return(ldapToDoguDescriptor, nil)
	EtcdDoguRegistry.Mock.On("IsEnabled", "ldap").Once().Return(true, nil)
	EtcdDoguRegistry.Mock.On("Register", ldapToDoguDescriptor).Once().Return(nil)
	EtcdDoguRegistry.Mock.On("Enable", ldapToDoguDescriptor).Once().Return(nil)
	ImageRegistryMock.Mock.On("PullImageConfig", mock.Anything, "registry.cloudogu.com/official/ldap:2.4.49-1").Return(imageConfig, nil)
	EtcdDoguRegistry.Mock.On("Get", "ldap").Once().Return(ldapToDoguDescriptor, nil)

	Context("DoguUpgradeManager", func() {
		testCtx := context.TODO()

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
			Eventually(getObjectFromCluster(testCtx, &v1.Service{}, ldapFromDoguLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())
			Eventually(getObjectFromCluster(testCtx, &v1.PersistentVolumeClaim{}, ldapFromDoguLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())

			secretLookupKey := types.NamespacedName{Name: ldapFromDoguLookupKey.Name + "-private", Namespace: ldapFromDoguLookupKey.Namespace}
			Eventually(getObjectFromCluster(testCtx, &v1.Secret{}, secretLookupKey), TimeoutInterval, PollingInterval).Should(BeTrue())

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

			Expect(DoguRemoteRegistryMock.AssertExpectations(t)).To(BeTrue())
			Expect(ImageRegistryMock.AssertExpectations(t)).To(BeTrue())
			Expect(EtcdDoguRegistry.AssertExpectations(t)).To(BeTrue())
		})
	})
})

func getObjectFromCluster(ctx context.Context, objectType client.Object, lookupKey types.NamespacedName) bool {
	err := k8sClient.Get(ctx, lookupKey, objectType)
	return err == nil
}
