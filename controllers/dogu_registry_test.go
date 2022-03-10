package controllers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDoguRegistry_GetDogu(t *testing.T) {
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
		doguRegistry := NewHTTPDoguRegistry(validUser, validPw, testServer.URL)

		result, err := doguRegistry.GetDogu(doguCr)
		require.NoError(t, err)

		assert.Equal(t, ldapDogu, result)
	})

	t.Run("Error while doing request", func(t *testing.T) {
		doguRegistry := NewHTTPDoguRegistry(validUser, validPw, "wrongurl")

		_, err := doguRegistry.GetDogu(doguCr)

		assert.Error(t, err)
	})

	t.Run("Error with status code 401", func(t *testing.T) {
		doguRegistry := NewHTTPDoguRegistry(validUser, "invalid", testServer.URL)

		_, err := doguRegistry.GetDogu(doguCr)

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
		doguRegistry := NewHTTPDoguRegistry(validUser, validPw, testServer2.URL)

		_, err := doguRegistry.GetDogu(doguCr)

		assert.Contains(t, err.Error(), "unmarshal")
	})
}
