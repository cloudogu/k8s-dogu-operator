package registry_test

import (
	"encoding/json"
	"github.com/cloudogu/cesapp-lib/core"
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDoguRegistry_GetDogu(t *testing.T) {
	ldapDogu := &core.Dogu{Name: "ldap"}
	ldapDoguResource := &v1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
	}
	ldapBytes, err := json.Marshal(ldapDogu)
	require.NoError(t, err)

	validUser := "user"
	validPw := "pw"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !(ok && u == validUser && p == validPw) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(200)
		_, err := w.Write(ldapBytes)
		if err != nil {
			panic(err)
		}
	}))

	t.Run("Successful get dogu", func(t *testing.T) {
		doguRegistry := registry.NewHTTPDoguRegistry(validUser, validPw, testServer.URL)

		result, err := doguRegistry.GetDogu(ldapDoguResource)
		require.NoError(t, err)

		assert.Equal(t, ldapDogu, result)
	})

	t.Run("Error while doing request", func(t *testing.T) {
		doguRegistry := registry.NewHTTPDoguRegistry(validUser, validPw, "wrongurl")

		_, err := doguRegistry.GetDogu(ldapDoguResource)

		assert.Error(t, err)
	})

	t.Run("Error with status code 401", func(t *testing.T) {
		doguRegistry := registry.NewHTTPDoguRegistry(validUser, "invalid", testServer.URL)

		_, err := doguRegistry.GetDogu(ldapDoguResource)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "status code 401")
	})

	t.Run("Error unmarshal dogu", func(t *testing.T) {
		testServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, err := w.Write([]byte("not a dogu"))
			if err != nil {
				panic(err)
			}
		}))
		doguRegistry := registry.NewHTTPDoguRegistry(validUser, validPw, testServer2.URL)

		_, err := doguRegistry.GetDogu(ldapDoguResource)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "unmarshal")
	})
}
