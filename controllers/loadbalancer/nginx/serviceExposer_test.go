package nginx

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	namespace = "ecosystem"
)

var (
	tcpServicesLookupKey = types.NamespacedName{Name: "tcp-services", Namespace: namespace}
	udpServicesLookupKey = types.NamespacedName{Name: "udp-services", Namespace: namespace}
)

func TestNewIngressNginxTCPUDPExposer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		mockClient := fake.NewClientBuilder().Build()

		// when
		sut := NewIngressNginxTCPUDPExposer(mockClient)

		// then
		require.NotNil(t, sut)
		assert.Equal(t, mockClient, sut.client)
	})
}

func Test_getConfigMapNameForProtocol(t *testing.T) {
	t.Run("should return the protocol in lower case with -services suffix", func(t *testing.T) {
		// when
		result := getConfigMapNameForProtocol(net.TCP)

		// then
		require.Equal(t, "tcp-services", result)
	})
}

func Test_getServiceEntryKey(t *testing.T) {
	t.Run("should return the host port as string", func(t *testing.T) {
		// when
		result := getServiceEntryKey(core.ExposedPort{Host: 2222})

		// then
		require.Equal(t, "2222", result)
	})
}

func Test_getServiceEntryValue(t *testing.T) {
	t.Run("should return the namespace/servicename:containerport as string", func(t *testing.T) {
		// when
		result := getServiceEntryValue("ecosystem", &core.Dogu{Name: "scm"}, core.ExposedPort{Container: 2222})

		// then
		require.Equal(t, "ecosystem/scm:2222", result)
	})
}

func Test_getServiceEntryValuePrefix(t *testing.T) {
	t.Run("should return the namespace/servicename", func(t *testing.T) {
		// when
		result := getServiceEntryValuePrefix("ecosystem", &core.Dogu{Name: "scm"})

		// then
		require.Equal(t, "ecosystem/scm", result)
	})
}

func Test_getExposedPortsByType(t *testing.T) {
	type args struct {
		dogu     *core.Dogu
		protocol string
	}
	tests := []struct {
		name string
		args args
		want []core.ExposedPort
	}{
		{name: "should return nil slice with no exposed ports", args: args{dogu: &core.Dogu{}, protocol: "tcp"}, want: []core.ExposedPort(nil)},
		{name: "should return nil slice with just udp ports", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "udp"}}}, protocol: "tcp"}, want: []core.ExposedPort(nil)},
		{name: "should return nil slice with just udp ports without http or https", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "udp"}, {Host: 80, Type: "udp"}, {Host: 443, Type: "udp"}}}, protocol: "tcp"}, want: []core.ExposedPort(nil)},
		{name: "should return nil slice with just tcp ports", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "tcp"}}}, protocol: "udp"}, want: []core.ExposedPort(nil)},
		{name: "should return nil slice with just tcp ports without http or https", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "tcp"}, {Host: 80, Type: "tcp"}, {Host: 443, Type: "tcp"}}}, protocol: "udp"}, want: []core.ExposedPort(nil)},
		{name: "should return just tcp ports", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "tcp"}, {Host: 3333, Type: "udp"}}}, protocol: "tcp"}, want: []core.ExposedPort{{Host: 2222, Type: "tcp"}}},
		{name: "should return just udp ports", args: args{dogu: &core.Dogu{ExposedPorts: []core.ExposedPort{{Host: 2222, Type: "tcp"}, {Host: 3333, Type: "udp"}}}, protocol: "udp"}, want: []core.ExposedPort{{Host: 3333, Type: "udp"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getExposedPortsByType(tt.args.dogu, tt.args.protocol), "getExposedPortsByType(%v, %v)", tt.args.dogu, tt.args.protocol)
		})
	}
}

func Test_filterDoguServices(t *testing.T) {
	emptyCm := &corev1.ConfigMap{}
	emptyMap := map[string]string{}
	namespace := "ecosystem"
	scmCm := &corev1.ConfigMap{Data: map[string]string{"2222": "ecosystem/scm:2222"}}
	mixedCm := &corev1.ConfigMap{Data: map[string]string{"2222": "ecosystem/scm:2222", "3333": "ecosystem/ldap:3333"}}
	scmDogu := &core.Dogu{Name: "scm", ExposedPorts: []core.ExposedPort{{Type: "tcp", Host: 2222, Container: 2222}}}
	scmV2Dogu := &core.Dogu{Name: "scm", ExposedPorts: []core.ExposedPort{}}

	type args struct {
		cm        *corev1.ConfigMap
		namespace string
		dogu      *core.Dogu
	}

	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{"should return empty map on empty cm", args{
			cm:        emptyCm,
			namespace: namespace,
			dogu:      scmDogu,
		}, emptyMap},
		{"should return empty map on cm with only dogu ports", args{
			cm:        scmCm,
			namespace: namespace,
			dogu:      scmDogu,
		}, emptyMap},
		{"should leave other ports", args{
			cm:        mixedCm,
			namespace: namespace,
			dogu:      scmDogu,
		}, map[string]string{"3333": "ecosystem/ldap:3333"}},
		{"should remove all ports from dogu in namespace", args{
			cm:        scmCm,
			namespace: namespace,
			dogu:      scmV2Dogu,
		}, emptyMap},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, filterDoguServices(tt.args.cm, tt.args.namespace, tt.args.dogu), "filterDoguServices(%v, %v, %v)", tt.args.cm, tt.args.namespace, tt.args.dogu)
		})
	}
}

func TestIngressNginxTcpUpdExposer_ExposeOrUpdateDoguServices(t *testing.T) {
	t.Run("success with no existent configmaps", func(t *testing.T) {
		// given
		ldap := readLdapDogu(t)
		mockClient := fake.NewClientBuilder().Build()
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.ExposeOrUpdateDoguServices(context.TODO(), namespace, ldap)

		// then
		require.NoError(t, err)

		tcpCm := &corev1.ConfigMap{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "tcp-services"}, tcpCm)
		require.NoError(t, err)
		assert.Equal(t, 2, len(tcpCm.Data))
		assert.Equal(t, "ecosystem/ldap:3333", tcpCm.Data["2222"])
		assert.Equal(t, "ecosystem/ldap:7777", tcpCm.Data["8888"])

		udpCm := &corev1.ConfigMap{}
		err = mockClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "udp-services"}, udpCm)
		require.NoError(t, err)
		assert.Equal(t, 1, len(udpCm.Data))
		assert.Equal(t, "ecosystem/ldap:4444", udpCm.Data["3333"])
	})

	t.Run("should return nil if the dogu has no exposed ports", func(t *testing.T) {
		// given
		sut := ingressNginxTcpUpdExposer{}

		// when
		err := sut.ExposeOrUpdateDoguServices(context.TODO(), namespace, &core.Dogu{})

		// then
		require.Nil(t, err)
	})

	t.Run("should throw an error getting tcp-configmap", func(t *testing.T) {
		// given
		ldap := readLdapDogu(t)
		mockClient := mocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		sut := ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.ExposeOrUpdateDoguServices(context.TODO(), namespace, ldap)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get configmap tcp-services")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should throw an error getting udp-configmap", func(t *testing.T) {
		// given
		ldap := readLdapDogu(t)
		mockClient := mocks.NewK8sClient(t)
		expect := mockClient.EXPECT()
		expect.Get(context.TODO(), udpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		expect.Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(nil)
		expect.Update(context.TODO(), mock.IsType(&corev1.ConfigMap{})).Return(nil)
		sut := ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.ExposeOrUpdateDoguServices(context.TODO(), namespace, ldap)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get configmap udp-services")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_ingressNginxTcpUpdExposer_exposeOrUpdatePortsForProtocol(t *testing.T) {
	t.Run("should return nil if no legacy ports are in configmap and the dogu doesnt contain new ports", func(t *testing.T) {
		// given
		ldap := readLdapDoguOnlyUDP(t)
		tcpCm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "tcp-services", Namespace: namespace}, Data: map[string]string{"2222": "ecosystem/notldap:3333"}}
		mockClient := mocks.NewK8sClient(t)
		expect := mockClient.EXPECT()
		expect.Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			object = tcpCm
			return nil
		})
		sut := ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.exposeOrUpdatePortsForProtocol(context.TODO(), namespace, ldap, net.TCP)

		// then
		require.Nil(t, err)
	})

	t.Run("should return error on update failure", func(t *testing.T) {
		// given
		ldap := readLdapDoguOnlyUDP(t)
		udpCm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "udp-services", Namespace: namespace}, Data: map[string]string{"2222": "ecosystem/notldap:3333"}}
		mockClient := mocks.NewK8sClient(t)
		expect := mockClient.EXPECT()
		expect.Get(context.TODO(), udpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			object = udpCm
			return nil
		})
		expect.Update(context.TODO(), mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		sut := ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.exposeOrUpdatePortsForProtocol(context.TODO(), namespace, ldap, net.UDP)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update configmap")
	})

	t.Run("should return error on creation failure", func(t *testing.T) {
		// given
		ldap := readLdapDoguOnlyUDP(t)
		mockClient := mocks.NewK8sClient(t)
		expect := mockClient.EXPECT()
		expect.Get(context.TODO(), udpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(apierrors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "Configmap"}, "udp-services"))
		expect.Create(context.TODO(), mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		sut := ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.exposeOrUpdatePortsForProtocol(context.TODO(), namespace, ldap, net.UDP)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create configmap udp-services")
	})
}

func Test_ingressNginxTcpUpdExposer_createNginxExposeConfigMapForProtocol(t *testing.T) {
	t.Run("should return nil the dogu contains no matching protocol ports", func(t *testing.T) {
		// given
		sut := &ingressNginxTcpUpdExposer{}

		// when
		cm, err := sut.createNginxExposeConfigMapForProtocol(context.TODO(), "ecosystem", readLdapDoguOnlyUDP(t), net.TCP)

		// then
		require.Nil(t, err)
		require.Nil(t, cm)
	})
}

func Test_ingressNginxTcpUpdExposer_DeleteDoguServices(t *testing.T) {
	t.Run("should return nil if the dogu doesnt contain exposed ports", func(t *testing.T) {
		// given
		sut := &ingressNginxTcpUpdExposer{}

		// when
		err := sut.DeleteDoguServices(context.TODO(), "ecosystem", &core.Dogu{})

		// then
		require.Nil(t, err)
	})

	t.Run("should return error on getting tcp-services configmap failure", func(t *testing.T) {
		// given
		mockClient := mocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.DeleteDoguServices(context.TODO(), "ecosystem", readLdapDogu(t))

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get configmap tcp-services")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on getting udp-services configmap failure", func(t *testing.T) {
		// given
		mockClient := mocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(nil)
		mockClient.EXPECT().Get(context.TODO(), udpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(assert.AnError)
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.DeleteDoguServices(context.TODO(), "ecosystem", readLdapDogu(t))

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get configmap udp-services")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_ingressNginxTcpUpdExposer_deletePortsForProtocol(t *testing.T) {
	t.Run("return nil if configmap is not found", func(t *testing.T) {
		// given
		mockClient := mocks.NewK8sClient(t)
		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).Return(apierrors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "Configmap"}, "tcp-services"))
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.deletePortsForProtocol(context.TODO(), namespace, readLdapDogu(t), net.TCP)

		// then
		require.Nil(t, err)
	})

	t.Run("return nil if configmap has nil data", func(t *testing.T) {
		// given
		mockClient := mocks.NewK8sClient(t)
		emptyCm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "tcp-services", Namespace: namespace}}

		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).RunAndReturn(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
			obj = emptyCm
			return nil
		})
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.deletePortsForProtocol(context.TODO(), namespace, readLdapDogu(t), net.TCP)

		// then
		require.Nil(t, err)
	})

	t.Run("return nil if configmap has no data", func(t *testing.T) {
		// given
		mockClient := mocks.NewK8sClient(t)
		emptyCm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "tcp-services", Namespace: namespace}, Data: map[string]string{}}

		mockClient.EXPECT().Get(context.TODO(), tcpServicesLookupKey, mock.IsType(&corev1.ConfigMap{})).RunAndReturn(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
			obj = emptyCm
			return nil
		})
		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.deletePortsForProtocol(context.TODO(), namespace, readLdapDogu(t), net.TCP)

		// then
		require.Nil(t, err)
	})

	t.Run("should delete all dogu entries from configmap", func(t *testing.T) {
		// given
		tcpCm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "tcp-services", Namespace: namespace}, Data: map[string]string{"2222": "ecosystem/ldap:3333", "1234": "ecosystem/notldap:1234"}}
		mockClient := fake.NewClientBuilder().WithObjects(tcpCm).Build()

		sut := &ingressNginxTcpUpdExposer{client: mockClient}

		// when
		err := sut.deletePortsForProtocol(context.TODO(), namespace, readLdapDogu(t), net.TCP)

		// then
		require.Nil(t, err)

		cm := &corev1.ConfigMap{}
		err = mockClient.Get(context.TODO(), tcpServicesLookupKey, cm)
		require.Nil(t, err)
		assert.Equal(t, 1, len(cm.Data))
	})
}
