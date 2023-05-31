package resource

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguExposedPortHandler(t *testing.T) {
	// given
	mockClient := fake.NewClientBuilder().Build()

	// when
	sut := NewDoguExposedPortHandler(mockClient)

	// then
	require.NotNil(t, sut)
}

func Test_doguExposedPortHandler_CreateOrUpdateCesLoadbalancerService(t *testing.T) {
	t.Run("should return empty service and nil if dogu has no exposed ports", func(t *testing.T) {
		// given
		sut := &doguExposedPortHandler{}

		// when
		service, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), &k8sv1.Dogu{}, &core.Dogu{})

		// then
		require.Nil(t, err)
		assert.Equal(t, service, &v1.Service{})
	})

	t.Run("should return error on error getting service error", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, &v1.Service{}).Return(assert.AnError)
		sut := &doguExposedPortHandler{client: mockClient}

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service ces-loadbalancer")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return an error on tcp/udp exposure if no loadbalancer is available", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().ExposeOrUpdateDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(assert.AnError)
		mockClient := fake.NewClientBuilder().Build()
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to expose dogu services")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should create a new loadbalancer service if none is in the cluster", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		expectedLoadBalancer := readNginxIngressOnlyExpectedLoadBalancer(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().ExposeOrUpdateDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		mockClient := fake.NewClientBuilder().Build()
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.NoError(t, err)

		serviceFromCluster := &v1.Service{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, serviceFromCluster)
		require.NoError(t, err)

		assertServicePorts(t, expectedLoadBalancer, serviceFromCluster)
		assert.Equal(t, v1.ServiceTypeLoadBalancer, serviceFromCluster.Spec.Type)
		assert.Equal(t, 0, len(serviceFromCluster.ObjectMeta.OwnerReferences))
	})

	t.Run("should return an error on service creation error", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			&v1.Service{}).Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, "ces-loadbalancer"))
		sut := &doguExposedPortHandler{client: mockClient}
		mockClient.EXPECT().Create(context.TODO(), mock.IsType(&v1.Service{})).Return(assert.AnError)

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create ces-loadbalancer service")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return an error on exposing tcp/udp service error if a loadbalancer is available", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)

		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type:  v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{{Name: "scm-2222", Port: 2222, TargetPort: intstr.IntOrString{IntVal: 2222}, Protocol: v1.ProtocolTCP}},
			},
		}

		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().ExposeOrUpdateDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(assert.AnError)

		mockClient := fake.NewClientBuilder().WithObjects(existingLB).Build()
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to expose dogu services")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should update the existing loadbalancer", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		expectedLoadBalancer := readNginxIngressSCMExpectedLoadBalancer(t)

		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type:  v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{{Name: "scm-2222", Port: 2222, TargetPort: intstr.IntOrString{IntVal: 2222}, Protocol: v1.ProtocolTCP}},
			},
		}

		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().ExposeOrUpdateDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)

		mockClient := fake.NewClientBuilder().WithObjects(existingLB).Build()
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.NoError(t, err)

		serviceFromCluster := &v1.Service{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, serviceFromCluster)
		require.NoError(t, err)

		assertServicePorts(t, expectedLoadBalancer, serviceFromCluster)
		assert.Equal(t, v1.ServiceTypeLoadBalancer, serviceFromCluster.Spec.Type)
		assert.Equal(t, 0, len(serviceFromCluster.ObjectMeta.OwnerReferences))
	})

	t.Run("should return an error on service update error", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)

		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type:  v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{{Name: "scm-2222", Port: 2222, TargetPort: intstr.IntOrString{IntVal: 2222}, Protocol: v1.ProtocolTCP}},
			},
		}

		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().ExposeOrUpdateDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)

		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			&v1.Service{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			object = existingLB
			return nil
		})
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}
		mockClient.EXPECT().Update(context.TODO(), mock.IsType(&v1.Service{})).Return(assert.AnError)

		// when
		_, err := sut.CreateOrUpdateCesLoadbalancerService(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update ces-loadbalancer service")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func assertServicePorts(t *testing.T, expected *v1.Service, service *v1.Service) {
	assert.Equal(t, len(expected.Spec.Ports), len(service.Spec.Ports))

	for _, servicePort := range service.Spec.Ports {
		found := false
		for _, expectedServicePort := range expected.Spec.Ports {
			if areServicePortsEqual(expectedServicePort, servicePort) {
				found = true
				break
			}
		}
		if !found {
			t.Fail()
		}
	}
}

func areServicePortsEqual(x v1.ServicePort, y v1.ServicePort) bool {
	return x.Port == y.Port && x.Name == y.Name && x.TargetPort == y.TargetPort && x.Protocol == y.Protocol
}

func Test_doguExposedPortHandler_RemoveExposedPorts(t *testing.T) {
	t.Run("should do nothing if the dogu has no exposed ports", func(t *testing.T) {
		// given
		sut := &doguExposedPortHandler{}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), &k8sv1.Dogu{}, &core.Dogu{})

		// then
		require.Nil(t, err)
	})

	t.Run("should return error on tcp/udp exposure error", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(assert.AnError)
		sut := &doguExposedPortHandler{serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete entries from expose configmap")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should do nothing if no loadbalancer service exists", func(t *testing.T) {
		// given
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		mockClient := fake.NewClientBuilder().Build()
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Nil(t, err)
	})

	t.Run("should return an error on service get error", func(t *testing.T) {
		// given
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, &v1.Service{}).Return(assert.AnError)
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service ces-loadbalancer")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should update the loadbalancer service ports if others are existent", func(t *testing.T) {
		// given
		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{{Name: "scm-2222", Port: 2222, TargetPort: intstr.IntOrString{IntVal: 2222}, Protocol: v1.ProtocolTCP},
					{Name: "nginx-ingress-80", Port: 80, TargetPort: intstr.IntOrString{IntVal: 80}, Protocol: v1.ProtocolTCP},
					{Name: "nginx-ingress-443", Port: 443, TargetPort: intstr.IntOrString{IntVal: 443}, Protocol: v1.ProtocolTCP}},
			},
		}
		expectedLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type:  v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{{Name: "scm-2222", Port: 2222, TargetPort: intstr.IntOrString{IntVal: 2222}, Protocol: v1.ProtocolTCP}},
			}}

		mockClient := fake.NewClientBuilder().WithObjects(existingLB).Build()
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Nil(t, err)

		serviceFromCluster := &v1.Service{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, serviceFromCluster)
		require.NoError(t, err)

		assertServicePorts(t, expectedLB, serviceFromCluster)
	})

	t.Run("should delete the service if the dogu ports are the only ones", func(t *testing.T) {
		// given
		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{
					{Name: "nginx-ingress-80", Port: 80, TargetPort: intstr.IntOrString{IntVal: 80}, Protocol: v1.ProtocolTCP},
					{Name: "nginx-ingress-443", Port: 443, TargetPort: intstr.IntOrString{IntVal: 443}, Protocol: v1.ProtocolTCP}},
			},
		}

		mockClient := fake.NewClientBuilder().WithObjects(existingLB).Build()
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Nil(t, err)

		serviceFromCluster := &v1.Service{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"}, serviceFromCluster)
		require.Error(t, err)
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("should return an error on service deletion error", func(t *testing.T) {
		// given
		existingLB := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{
					{Name: "nginx-ingress-80", Port: 80, TargetPort: intstr.IntOrString{IntVal: 80}, Protocol: v1.ProtocolTCP},
					{Name: "nginx-ingress-443", Port: 443, TargetPort: intstr.IntOrString{IntVal: 443}, Protocol: v1.ProtocolTCP}},
			},
		}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), types.NamespacedName{Name: "ces-loadbalancer", Namespace: "ecosystem"},
			&v1.Service{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			object = existingLB
			return nil
		})
		mockClient.EXPECT().Delete(context.TODO(), mock.IsType(&v1.Service{})).Return(assert.AnError)
		nginxIngressCR := readNginxIngressDoguResource(t)
		nginxIngressDogu := readNginxIngressDogu(t)
		serviceExposer := mocks.NewTcpUpdServiceExposer(t)
		serviceExposer.EXPECT().DeleteDoguServices(context.TODO(), nginxIngressCR.Namespace, nginxIngressDogu).Return(nil)
		sut := &doguExposedPortHandler{client: mockClient, serviceExposer: serviceExposer}

		// when
		err := sut.RemoveExposedPorts(context.TODO(), nginxIngressCR, nginxIngressDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete service ces-loadbalancer")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
