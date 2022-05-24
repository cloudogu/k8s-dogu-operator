package resource_test

import (
	"context"
	_ "embed"
	"github.com/cloudogu/cesapp/v4/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte
var ldapCr = &k8sv1.Dogu{}

func init() {
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		panic(err)
	}
}

func TestNewDoguSecretsWriter(t *testing.T) {
	// given
	fakeClient := fake.NewClientBuilder().Build()
	registryMock := &mocks.Registry{}

	// when
	writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

	// then
	require.NotNil(t, writer)
}

func Test_doguSecretWriter_WriteDoguSecretsToRegistry(t *testing.T) {
	ctx := context.Background()
	ldapCr.Namespace = "test"
	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-secrets", Namespace: "test"}}
	secret.Data = map[string][]byte{"key.test": []byte("value"), "key.test1": []byte("value1")}
	publicKey := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAq/RuSmgG90VfEIY+ok+j\n9Jbx0avXSfIRD3uPd6mH576o1vDjDou08eo3XLTL28LN+AJ5f/HNGKyV6C6Z5X7E\nYV/rayICFHSxS8QimVLk11Du5ht2bniNekVFRPal+HQ6cJXF5GE77Q0tlNvc6cXS\nfU3E1a+KVL+zzvpApSgMV+ZXfjxOjPB7nNJ7n6Ls6DyMVIBmz1Tc+XTh9PK+148F\nzEWjx59MqNKZgk5p8dR9Bvpr1AmqxZp9IpzpqhQ3GoyGW9dZafUK1eQ/pggEDget\n+H/U+pcpUqRsQ1sRyqfkvm5Y5uy3SPtGcK8H45zTtsZYcOEHS9SMyC+pXiSMDRsT\nNQIDAQAB\n-----END PUBLIC KEY-----"

	t.Run("success", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		secret.ResourceVersion = ""
		err := fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		registryMock := &mocks.Registry{}
		doguConfigMock := &mocks.ConfigurationContext{}
		doguConfigMock.On("Get", "public.pem").Return(publicKey, nil)
		doguConfigMock.On("Set", "key/test", mock.Anything).Return(nil)
		doguConfigMock.On("Set", "key/test1", mock.Anything).Return(nil)
		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)
		registryMock.On("DoguConfig", "ldap").Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err = writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.NoError(t, err)
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "ldap-secrets", Namespace: "test"}, &v1.Secret{})
		require.True(t, errors.IsNotFound(err))
		mock.AssertExpectationsForObjects(t, registryMock, doguConfigMock, globalConfigMock)
	})

	t.Run("success because dogu secret is not found", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		registryMock := &mocks.Registry{}
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err := writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, registryMock)
	})

	t.Run("failing get key provider", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		secret.ResourceVersion = ""
		err := fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.On("Get", "key_provider").Return("", assert.AnError)
		registryMock := &mocks.Registry{}
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err = writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get key provider")
		mock.AssertExpectationsForObjects(t, registryMock, globalConfigMock)
	})

	t.Run("fail to get key provider from string", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		secret.ResourceVersion = ""
		err := fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.On("Get", "key_provider").Return("invalid provider", nil)
		registryMock := &mocks.Registry{}
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err = writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create keyprovider")
		mock.AssertExpectationsForObjects(t, registryMock, globalConfigMock)
	})

	t.Run("fail to get public key str", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		secret.ResourceVersion = ""
		err := fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		doguConfigMock := &mocks.ConfigurationContext{}
		doguConfigMock.On("Get", "public.pem").Return("", assert.AnError)
		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.On("Get", "key_provider").Return("pkcs1v15", nil)
		registryMock := &mocks.Registry{}
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", "ldap").Return(doguConfigMock)
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err = writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get public key for dogu")
		mock.AssertExpectationsForObjects(t, registryMock, globalConfigMock, doguConfigMock)
	})

	t.Run("fail to set encrypted value", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()
		secret.ResourceVersion = ""
		err := fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		doguConfigMock := &mocks.ConfigurationContext{}
		doguConfigMock.On("Get", "public.pem").Return(publicKey, nil)
		doguConfigMock.On("Set", mock.Anything, mock.Anything).Return(assert.AnError)
		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.On("Get", "key_provider").Return("pkcs1v15", nil)
		registryMock := &mocks.Registry{}
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", "ldap").Return(doguConfigMock)
		writer := resource.NewDoguSecretsWriter(fakeClient, registryMock)

		// when
		err = writer.WriteDoguSecretsToRegistry(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write key")
		mock.AssertExpectationsForObjects(t, registryMock, globalConfigMock, doguConfigMock)
	})
}
