package authregistration

import (
	"context"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

func TestAuthRegistrationManager_EnsureAuthRegistration(t *testing.T) {
	ctx := context.Background()

	t.Run("should fail if dogu is nil", func(t *testing.T) {
		manager := &AuthRegistrationManager{}

		err := manager.EnsureAuthRegistration(ctx, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu must not be nil")
	})

	t.Run("should skip if dogu has no CAS service account", func(t *testing.T) {
		client := &fakeAuthRegistrationClient{}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "postgresql"},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
		assert.Equal(t, 0, client.getCalls)
		assert.Equal(t, 0, client.createCalls)
		assert.Equal(t, 0, client.updateCalls)
	})

	t.Run("should fail if CAS service account exists but client is not configured", func(t *testing.T) {
		manager := &AuthRegistrationManager{}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "auth registration client is not configured")
	})

	t.Run("should fail if protocol parameter is invalid", func(t *testing.T) {
		manager := &AuthRegistrationManager{client: &fakeAuthRegistrationClient{}}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"invalid"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse CAS service account parameters")
		assert.ErrorContains(t, err, `unsupported protocol value "invalid"`)
	})

	t.Run("should return get error", func(t *testing.T) {
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, assert.AnError
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 1, client.getCalls)
	})

	t.Run("should create auth registration if it does not exist", func(t *testing.T) {
		expectedName := createAuthRegistrationName("redmine")
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, newNotFoundErr(expectedName)
			},
			createFn: func(_ context.Context, authRegistration *authRegApiV1.AuthRegistration, _ metav1.CreateOptions) (*authRegApiV1.AuthRegistration, error) {
				require.Equal(t, expectedName, authRegistration.Name)
				assert.Equal(t, authRegApiV1.AuthProtocolCAS, authRegistration.Spec.Protocol)
				assert.Equal(t, "redmine", authRegistration.Spec.Consumer)
				assert.Nil(t, authRegistration.Spec.LogoutURL)
				return authRegistration, nil
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
		assert.Equal(t, 1, client.getCalls)
		assert.Equal(t, 1, client.createCalls)
		assert.Equal(t, 0, client.updateCalls)
	})

	t.Run("should create auth registration with positional params", func(t *testing.T) {
		expectedName := createAuthRegistrationName("redmine")
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, newNotFoundErr(expectedName)
			},
			createFn: func(_ context.Context, authRegistration *authRegApiV1.AuthRegistration, _ metav1.CreateOptions) (*authRegApiV1.AuthRegistration, error) {
				require.Equal(t, expectedName, authRegistration.Name)
				assert.Equal(t, authRegApiV1.AuthProtocolOIDC, authRegistration.Spec.Protocol)
				assert.Equal(t, "redmine", authRegistration.Spec.Consumer)
				require.NotNil(t, authRegistration.Spec.LogoutURL)
				assert.Equal(t, "https://dogu.example/logout", *authRegistration.Spec.LogoutURL)
				return authRegistration, nil
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"OIDC", "https://dogu.example/logout"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
		assert.Equal(t, 1, client.getCalls)
		assert.Equal(t, 1, client.createCalls)
	})

	t.Run("should return error for key value style params", func(t *testing.T) {
		expectedName := createAuthRegistrationName("redmine")
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, newNotFoundErr(expectedName)
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"protocol=OAUTH", "logoutURL=https://dogu.example/logout", "service=service-a"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of CAS service account params")
		assert.Equal(t, 0, client.getCalls)
		assert.Equal(t, 0, client.createCalls)
	})

	t.Run("should return create error", func(t *testing.T) {
		expectedName := createAuthRegistrationName("redmine")
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, newNotFoundErr(expectedName)
			},
			createFn: func(_ context.Context, _ *authRegApiV1.AuthRegistration, _ metav1.CreateOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, assert.AnError
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should not update when current spec already matches", func(t *testing.T) {
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return &authRegApiV1.AuthRegistration{
					Spec: authRegApiV1.AuthRegistrationSpec{
						Protocol: authRegApiV1.AuthProtocolCAS,
						Consumer: "redmine",
					},
				}, nil
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
		assert.Equal(t, 1, client.getCalls)
		assert.Equal(t, 0, client.updateCalls)
	})

	t.Run("should update auth registration when spec differs", func(t *testing.T) {
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return &authRegApiV1.AuthRegistration{
					Spec: authRegApiV1.AuthRegistrationSpec{
						Protocol: authRegApiV1.AuthProtocolCAS,
						Consumer: "old-consumer",
					},
				}, nil
			},
			updateFn: func(_ context.Context, authRegistration *authRegApiV1.AuthRegistration, _ metav1.UpdateOptions) (*authRegApiV1.AuthRegistration, error) {
				assert.Equal(t, authRegApiV1.AuthProtocolCAS, authRegistration.Spec.Protocol)
				assert.Equal(t, "redmine", authRegistration.Spec.Consumer)
				return authRegistration, nil
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
		assert.Equal(t, 1, client.getCalls)
		assert.Equal(t, 1, client.updateCalls)
	})

	t.Run("should return update error", func(t *testing.T) {
		client := &fakeAuthRegistrationClient{
			getFn: func(_ context.Context, _ string, _ metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
				return &authRegApiV1.AuthRegistration{
					Spec: authRegApiV1.AuthRegistrationSpec{
						Protocol: authRegApiV1.AuthProtocolCAS,
						Consumer: "old-consumer",
					},
				}, nil
			},
			updateFn: func(_ context.Context, _ *authRegApiV1.AuthRegistration, _ metav1.UpdateOptions) (*authRegApiV1.AuthRegistration, error) {
				return nil, assert.AnError
			},
		}
		manager := &AuthRegistrationManager{client: client}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "cas", Params: []string{"cas"}},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func newNotFoundErr(name string) error {
	return k8sErr.NewNotFound(schema.GroupResource{Group: "api.k8s.cloudogu.com", Resource: "authregistrations"}, name)
}

type fakeAuthRegistrationClient struct {
	getCalls    int
	createCalls int
	updateCalls int
	deleteCalls int

	getFn    func(ctx context.Context, name string, opts metav1.GetOptions) (*authRegApiV1.AuthRegistration, error)
	createFn func(ctx context.Context, authRegistration *authRegApiV1.AuthRegistration, opts metav1.CreateOptions) (*authRegApiV1.AuthRegistration, error)
	updateFn func(ctx context.Context, authRegistration *authRegApiV1.AuthRegistration, opts metav1.UpdateOptions) (*authRegApiV1.AuthRegistration, error)
	deleteFn func(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

func (f *fakeAuthRegistrationClient) Create(ctx context.Context, authRegistration *authRegApiV1.AuthRegistration, opts metav1.CreateOptions) (*authRegApiV1.AuthRegistration, error) {
	f.createCalls++
	if f.createFn == nil {
		panic("unexpected call to Create")
	}
	return f.createFn(ctx, authRegistration, opts)
}

func (f *fakeAuthRegistrationClient) Update(ctx context.Context, authRegistration *authRegApiV1.AuthRegistration, opts metav1.UpdateOptions) (*authRegApiV1.AuthRegistration, error) {
	f.updateCalls++
	if f.updateFn == nil {
		panic("unexpected call to Update")
	}
	return f.updateFn(ctx, authRegistration, opts)
}

func (f *fakeAuthRegistrationClient) UpdateStatus(ctx context.Context, _ *authRegApiV1.AuthRegistration, _ metav1.UpdateOptions) (*authRegApiV1.AuthRegistration, error) {
	panic("unexpected call to UpdateStatus")
}

func (f *fakeAuthRegistrationClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	f.deleteCalls++
	if f.deleteFn == nil {
		panic("unexpected call to Delete")
	}
	return f.deleteFn(ctx, name, opts)
}

func (f *fakeAuthRegistrationClient) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	panic("unexpected call to DeleteCollection")
}

func (f *fakeAuthRegistrationClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*authRegApiV1.AuthRegistration, error) {
	f.getCalls++
	if f.getFn == nil {
		panic("unexpected call to Get")
	}
	return f.getFn(ctx, name, opts)
}

func (f *fakeAuthRegistrationClient) List(_ context.Context, _ metav1.ListOptions) (*authRegApiV1.AuthRegistrationList, error) {
	panic("unexpected call to List")
}

func (f *fakeAuthRegistrationClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	panic("unexpected call to Watch")
}

func (f *fakeAuthRegistrationClient) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (*authRegApiV1.AuthRegistration, error) {
	panic("unexpected call to Patch")
}

func TestCreateAuthRegistrationName(t *testing.T) {
	assert.Equal(t, "redmine-authregistration", createAuthRegistrationName("redmine"))
}

func TestParseLegacyCASServiceAccountParams(t *testing.T) {
	t.Run("should parse account type only", func(t *testing.T) {
		protocol, logoutURL, err := parseLegacyCASServiceAccountParams([]string{"cas"})

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolCAS, protocol)
		assert.Nil(t, logoutURL)
	})

	t.Run("should parse account type and logout uri", func(t *testing.T) {
		protocol, logoutURL, err := parseLegacyCASServiceAccountParams([]string{"oidc", "https://dogu.example/logout"})

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolOIDC, protocol)
		require.NotNil(t, logoutURL)
		assert.Equal(t, "https://dogu.example/logout", *logoutURL)
	})

	t.Run("should return error for invalid number of params", func(t *testing.T) {
		_, _, err := parseLegacyCASServiceAccountParams(nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of CAS service account params")
	})

	t.Run("should return error for invalid protocol", func(t *testing.T) {
		_, _, err := parseLegacyCASServiceAccountParams([]string{"invalid"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported protocol value")
	})
}
