package serviceaccount

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_createServiceAccount(t *testing.T) {
	ctx := context.TODO()

	t.Run("success create service account", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"
		params := []string{"param1", "42"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/", r.URL.String())
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, apiKey, r.Header.Get(apiKeyHeader))

			defer r.Body.Close()

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var requestBody createRequest
			err = json.Unmarshal(body, &requestBody)
			require.NoError(t, err)

			assert.Equal(t, consumer, requestBody.Consumer)
			assert.Equal(t, params, requestBody.Params)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"username": "adminUser", "password": "password123"})
		}))

		ac := &apiClient{}
		creds, err := ac.createServiceAccount(ctx, server.URL, apiKey, consumer, params)

		require.NoError(t, err)
		assert.Equal(t, Credentials{"username": "adminUser", "password": "password123"}, creds)
	})

	t.Run("fail on creating request", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"
		params := []string{"param1", "42"}

		ac := &apiClient{}
		_, err := ac.createServiceAccount(nil, "", apiKey, consumer, params)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error creating request:")
	})

	t.Run("fail on sending request", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"
		params := []string{"param1", "42"}

		ac := &apiClient{}
		_, err := ac.createServiceAccount(ctx, "", apiKey, consumer, params)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error while sending request:")
	})

	t.Run("fail on unsuccessful request", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"
		params := []string{"param1", "42"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/", r.URL.String())
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, apiKey, r.Header.Get(apiKeyHeader))

			w.WriteHeader(http.StatusInternalServerError)
		}))

		ac := &apiClient{}
		_, err := ac.createServiceAccount(ctx, server.URL, apiKey, consumer, params)

		require.Error(t, err)
		assert.ErrorContains(t, err, "request was not successful: 500 Internal Server Error")
	})

	t.Run("fail on parsing response", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"
		params := []string{"param1", "42"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/", r.URL.String())
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, apiKey, r.Header.Get(apiKeyHeader))

			w.WriteHeader(http.StatusOK)
		}))

		ac := &apiClient{}
		_, err := ac.createServiceAccount(ctx, server.URL, apiKey, consumer, params)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error parsing credentials from response: ")
	})
}

func Test_deleteServiceAccount(t *testing.T) {
	ctx := context.TODO()

	t.Run("success delete service account", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/grafana", r.URL.String())
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, apiKey, r.Header.Get(apiKeyHeader))

			w.WriteHeader(http.StatusNoContent)
		}))

		ac := &apiClient{}
		err := ac.deleteServiceAccount(ctx, server.URL, apiKey, consumer)

		require.NoError(t, err)
	})

	t.Run("fail on creating url", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"

		ac := &apiClient{}
		err := ac.deleteServiceAccount(ctx, "tt\\:\\not=>???valid", apiKey, consumer)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error creating url:")
	})

	t.Run("fail on error creating request", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"

		ac := &apiClient{}
		err := ac.deleteServiceAccount(nil, "", apiKey, consumer)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error creating request:")
	})

	t.Run("fail on error sending request", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"

		ac := &apiClient{}
		err := ac.deleteServiceAccount(ctx, "", apiKey, consumer)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error while sending request:")
	})

	t.Run("fail on error in response", func(t *testing.T) {
		apiKey := "secretApiKey"
		consumer := "grafana"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/grafana", r.URL.String())
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, apiKey, r.Header.Get(apiKeyHeader))

			w.WriteHeader(http.StatusInternalServerError)
		}))

		ac := &apiClient{}
		err := ac.deleteServiceAccount(ctx, server.URL, apiKey, consumer)

		require.Error(t, err)
		assert.ErrorContains(t, err, "request was not successful: 500 Internal Server Error")
	})
}
