package authregistration

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	authRegClientV1 "github.com/cloudogu/k8s-auth-registration-lib/client/typed/api/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthRegistrationManager struct {
	client authRegistrationClient
}

// NewManager creates an AuthRegistrationManager which can be used to create and remove AuthRegistration resources.
func NewManager(client authRegClientV1.AuthRegistrationInterface) *AuthRegistrationManager {
	return &AuthRegistrationManager{
		client: client,
	}
}

// EnsureAuthRegistration creates/updates the AuthRegistration and syncs sensitive credentials.
func (sm *AuthRegistrationManager) EnsureAuthRegistration(ctx context.Context, dogu *cesappcore.Dogu) error {
	if dogu == nil {
		return fmt.Errorf("dogu must not be nil")
	}

	serviceAccount, found := getCASServiceAccount(dogu)
	if !found {
		return nil
	}

	if sm.client == nil {
		return fmt.Errorf("auth registration client is not configured")
	}

	protocol, logoutURL, err := parseLegacyCASServiceAccountParams(serviceAccount.Params)
	if err != nil {
		return fmt.Errorf("failed to parse CAS service account parameters: %w", err)
	}

	desiredAuthReg := &authRegApiV1.AuthRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: createAuthRegistrationName(dogu.GetSimpleName()),
		},
		Spec: authRegApiV1.AuthRegistrationSpec{
			Protocol:  protocol,
			Consumer:  dogu.GetSimpleName(),
			LogoutURL: logoutURL,
		},
	}

	authRegistrationName := createAuthRegistrationName(dogu.GetSimpleName())

	authReg, err := sm.client.Get(ctx, authRegistrationName, metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			_, createErr := sm.client.Create(ctx, desiredAuthReg, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create AuthRegistration: %w", createErr)
			}

			return nil
		}

		return fmt.Errorf("failed to get AuthRegistration: %w", err)
	}

	if reflect.DeepEqual(authReg.Spec, desiredAuthReg.Spec) {
		return nil
	}

	authReg.Spec = desiredAuthReg.Spec

	_, err = sm.client.Update(ctx, authReg, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update AuthRegistration: %w", err)
	}

	return nil
}

// RemoveAuthRegistration removes the AuthRegistration belonging to the given dogu.
func (sm *AuthRegistrationManager) RemoveAuthRegistration(ctx context.Context, doguName cescommons.SimpleName) error {
	if sm.client == nil {
		return fmt.Errorf("auth registration client is not configured")
	}

	err := sm.client.Delete(ctx, createAuthRegistrationName(doguName.String()), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete AuthRegistration: %w", err)
	}

	return nil
}

func createAuthRegistrationName(doguName string) string {
	return fmt.Sprintf("%s-authregistration", doguName)
}

func getCASServiceAccount(dogu *cesappcore.Dogu) (cesappcore.ServiceAccount, bool) {
	for _, serviceAccount := range dogu.ServiceAccounts {
		if !isDoguServiceAccount(serviceAccount) {
			continue
		}

		if serviceAccount.Type == "cas" {
			return serviceAccount, true
		}
	}

	return cesappcore.ServiceAccount{}, false
}

func isDoguServiceAccount(serviceAccount cesappcore.ServiceAccount) bool {
	return serviceAccount.Kind == "" || serviceAccount.Kind == "dogu"
}

func parseLegacyCASServiceAccountParams(params []string) (authRegApiV1.AuthProtocol, *string, error) {
	// Legacy CAS script usage: create-sa.sh account_type [logout_uri] servicename
	// serviceAccount.Params therefore must be: account_type [logout_uri]
	if len(params) < 1 || len(params) > 2 {
		return "", nil, fmt.Errorf("invalid number of CAS service account params: expected account_type [logout_uri]")
	}

	accountType := strings.TrimSpace(params[0])
	if accountType == "" {
		return "", nil, fmt.Errorf("account_type must not be empty")
	}

	protocol, err := parseProtocol(accountType)
	if err != nil {
		return "", nil, err
	}

	var logoutURL *string
	if len(params) == 2 {
		logoutURI := strings.TrimSpace(params[1])
		if logoutURI != "" {
			logoutURL = &logoutURI
		}
	}

	return protocol, logoutURL, nil
}

func parseProtocol(protocol string) (authRegApiV1.AuthProtocol, error) {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case string(authRegApiV1.AuthProtocolCAS):
		return authRegApiV1.AuthProtocolCAS, nil
	case string(authRegApiV1.AuthProtocolOIDC):
		return authRegApiV1.AuthProtocolOIDC, nil
	case string(authRegApiV1.AuthProtocolOAuth):
		return authRegApiV1.AuthProtocolOAuth, nil
	default:
		return "", fmt.Errorf("unsupported protocol value %q", protocol)
	}
}
