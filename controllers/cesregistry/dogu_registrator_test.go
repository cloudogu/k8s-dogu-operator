package cesregistry

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks"
)

func TestEtcdDoguRegistrator_RegisterNewDogu(t *testing.T) {
	ctx := context.TODO()
	scheme := getTestScheme()

	ldapCr := &corev1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "clusterns"},
		Spec: corev1.DoguSpec{
			Name:    "official/ldap",
			Version: "1.0.0",
		},
	}
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("skip registration because dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(assert.AnError)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to register dogu")
	})

	t.Run("fail get key_provider", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", assert.AnError)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get key provider")
	})

	t.Run("fail to write public key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(assert.AnError)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to write")
	})

	t.Run("fail generate secret", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to generate secret")
	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable dogu")
	})

	t.Run("success with existing private key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)

		privateKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAwp063ZghmURQMUQCy10FQyYvLW7GXvPtKyQv4Ts/0qz9V1S2\ngvT0zvQ6zBiFEDeiebGyA88Uk09ts18f7tt2o9JiZo5wi6MjEstTBmFlvtii9+9j\nkRKlEHaaZI7ethZmBdmxBRfX2lJijj+RVmQd9IexZ1FkoCQS2ChybmAAD/XfjNWR\nGxOykNAoDmPy0wswE39yl9YJjOR5MKJgIAsOf/uNjVud+6iOklPaWsZTxU1rknL8\na9BQzcf/xiciLubdJL5933b+HpLz/OJzonNgZ1aLguWXMFdhhaxOjOAuAi8Kvs1U\nPTslJ5F+mJVGBrl9IEghGSDFARwiyREWZA7WPQIDAQABAoIBAQC8wuArurkr/bSC\ndGL5eRn3jXvI518FDjcF1y2RmnRHFX8MS6BS2ODyMrUs7MNzfWLcAlyVkS91yl6u\n0h8ZAEjMkOzcaGAFMJB+VDQNRj73oww+yzSZq6nqk/8gderSVltSZVlrhTraCXqK\nWmHPl3/uhAawHaQqJ5MXkfOb1wV4c+JQqLOStR0rFwitReKalZp/g6KxfUnyrUox\nAXsm1MU1YRCTgPTA9b8RbsRsqFwCKqABm1w/Jo9qKGXrIBm2av5XTX2cfpYTbm1C\nCG8LEQA7yaByxroBaPyczDAAlypU+wSVM3cW3+ABacDYAKPAHiAZaEsBIDxPpwUg\nEQsz9dpJAoGBAPP8rt6OxEWM/LGnKYGjZ2m4weg8OWjypGbTkBplXIuGkCaSw8GJ\nKhGEre4u5CIaf1Otdcle7H/QsW85F0gKzuIiPCuTuR+Rk9eBEDSi7FaRLPpYV7yD\nhQbRroRYFkU4xYSz81Pj0vG5dCYp3z72co0sw4XwFBjKJkmN8OxFEhOzAoGBAMwy\nOu4CehFHsYVysiBQ2R3AKePv0zEN4q7W6CmFS8UWBTi1+k/MPJXhhozlSBs/w8Ah\nYpmAyNrzBUd1Dx+y18YpiM0/7LykB90l+fN2xJYrRq79qTZsuiJMJeHuloZyg3YT\nxqD7LiGwTOgtG7XKFvJvFyRjpmGc/aytLarB4TZPAoGAcejFp4hV3/bLvxFBEpI8\nZKJqfUcosnOeB5e8TmaGR2nCgQ/CLugf6N/d6DaiMb3XNjTkqegUWDQRstCfqvXI\n0tCS8PFd23w23sUV0M1Ds8LBkfuOsqdggueAJ6+MbjLsHGF7N+5EfLBNpsejv5yF\nrJ16h1yntU8jgvGuylAQ+XsCgYEAvx9gsveUg2oESXCiMscZgNQlIViO5sIlYxp5\ncKt30P+cYYlKwbfbGTpesq/EPuT+9m0JGb5FwVFnpot1XWkKt0qW5e2oSqSJS7/I\n5M1MkXXuEcoQwIUh7woxBvhG4Y57Z2B5MKIJerTGNyZJYmzF76J1GbU/vOuxMBdj\nwAj6H9cCgYEA3kkXC2DlS69Z14CmUVlHbTLpKAEDbBXuiyAqSwRzyQLOn1JE4vL0\nGZZ2DKJ9pYxb0VWaMDJcJm3ppcZPD/N0QqgoBIlEVp60dBgFvHYf7iuCAR8otX20\n69LCVRuY9dbz/I19eb9IT1eX8mhb6i73zjA5Ri9PF7z2epvEU2Lny5g=\n-----END RSA PRIVATE KEY-----\n"
		existingSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"},
			Data:       map[string][]byte{"private.pem": []byte(privateKey)},
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to get existing private key secret", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)

		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects().Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get secret for dogu ldap")
	})

	t.Run("failed to write public key from existing private key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)

		privateKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAwp063ZghmURQMUQCy10FQyYvLW7GXvPtKyQv4Ts/0qz9V1S2\ngvT0zvQ6zBiFEDeiebGyA88Uk09ts18f7tt2o9JiZo5wi6MjEstTBmFlvtii9+9j\nkRKlEHaaZI7ethZmBdmxBRfX2lJijj+RVmQd9IexZ1FkoCQS2ChybmAAD/XfjNWR\nGxOykNAoDmPy0wswE39yl9YJjOR5MKJgIAsOf/uNjVud+6iOklPaWsZTxU1rknL8\na9BQzcf/xiciLubdJL5933b+HpLz/OJzonNgZ1aLguWXMFdhhaxOjOAuAi8Kvs1U\nPTslJ5F+mJVGBrl9IEghGSDFARwiyREWZA7WPQIDAQABAoIBAQC8wuArurkr/bSC\ndGL5eRn3jXvI518FDjcF1y2RmnRHFX8MS6BS2ODyMrUs7MNzfWLcAlyVkS91yl6u\n0h8ZAEjMkOzcaGAFMJB+VDQNRj73oww+yzSZq6nqk/8gderSVltSZVlrhTraCXqK\nWmHPl3/uhAawHaQqJ5MXkfOb1wV4c+JQqLOStR0rFwitReKalZp/g6KxfUnyrUox\nAXsm1MU1YRCTgPTA9b8RbsRsqFwCKqABm1w/Jo9qKGXrIBm2av5XTX2cfpYTbm1C\nCG8LEQA7yaByxroBaPyczDAAlypU+wSVM3cW3+ABacDYAKPAHiAZaEsBIDxPpwUg\nEQsz9dpJAoGBAPP8rt6OxEWM/LGnKYGjZ2m4weg8OWjypGbTkBplXIuGkCaSw8GJ\nKhGEre4u5CIaf1Otdcle7H/QsW85F0gKzuIiPCuTuR+Rk9eBEDSi7FaRLPpYV7yD\nhQbRroRYFkU4xYSz81Pj0vG5dCYp3z72co0sw4XwFBjKJkmN8OxFEhOzAoGBAMwy\nOu4CehFHsYVysiBQ2R3AKePv0zEN4q7W6CmFS8UWBTi1+k/MPJXhhozlSBs/w8Ah\nYpmAyNrzBUd1Dx+y18YpiM0/7LykB90l+fN2xJYrRq79qTZsuiJMJeHuloZyg3YT\nxqD7LiGwTOgtG7XKFvJvFyRjpmGc/aytLarB4TZPAoGAcejFp4hV3/bLvxFBEpI8\nZKJqfUcosnOeB5e8TmaGR2nCgQ/CLugf6N/d6DaiMb3XNjTkqegUWDQRstCfqvXI\n0tCS8PFd23w23sUV0M1Ds8LBkfuOsqdggueAJ6+MbjLsHGF7N+5EfLBNpsejv5yF\nrJ16h1yntU8jgvGuylAQ+XsCgYEAvx9gsveUg2oESXCiMscZgNQlIViO5sIlYxp5\ncKt30P+cYYlKwbfbGTpesq/EPuT+9m0JGb5FwVFnpot1XWkKt0qW5e2oSqSJS7/I\n5M1MkXXuEcoQwIUh7woxBvhG4Y57Z2B5MKIJerTGNyZJYmzF76J1GbU/vOuxMBdj\nwAj6H9cCgYEA3kkXC2DlS69Z14CmUVlHbTLpKAEDbBXuiyAqSwRzyQLOn1JE4vL0\nGZZ2DKJ9pYxb0VWaMDJcJm3ppcZPD/N0QqgoBIlEVp60dBgFvHYf7iuCAR8otX20\n69LCVRuY9dbz/I19eb9IT1eX8mhb6i73zjA5Ri9PF7z2epvEU2Lny5g=\n-----END RSA PRIVATE KEY-----\n"
		existingSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"},
			Data:       map[string][]byte{"private.pem": []byte(privateKey)},
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to write public key from existing private key")
	})
}

func TestEtcdDoguRegistrator_RegisterDoguVersion(t *testing.T) {
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a new dogu version", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("fail because the dogu is not enabled", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not register dogu version: previous version not found")
	})

	t.Run("fail because the dogu cant be registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register dogu ldap")
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	t.Run("successfully unregister a dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(nil)
		doguRegistryMock.On("Unregister", "ldap").Return(nil)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.NoError(t, err)
	})

	t.Run("failed to remove dogu config", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to remove dogu config")
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(nil)
		doguRegistryMock.On("Unregister", "ldap").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to unregister dogu")
	})
}
