package ecoSystem

import (
	"context"
	"encoding/json"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_doguClient_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/testdogu", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "testdogu", Namespace: "test"}}
			doguBytes, err := json.Marshal(dogu)
			require.NoError(t, err)
			_, err = writer.Write(doguBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.Get(context.TODO(), "testdogu", v1.GetOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodGet, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			doguList := k8sv1.DoguList{}
			dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "testdogu", Namespace: "test"}}
			doguList.Items = append(doguList.Items, *dogu)
			doguBytes, err := json.Marshal(doguList)
			require.NoError(t, err)
			_, err = writer.Write(doguBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.List(context.TODO(), v1.ListOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdDogu := &k8sv1.Dogu{}
			require.NoError(t, json.Unmarshal(bytes, createdDogu))
			assert.Equal(t, "tocreate", createdDogu.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.Create(context.TODO(), dogu, v1.CreateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/tocreate", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdDogu := &k8sv1.Dogu{}
			require.NoError(t, json.Unmarshal(bytes, createdDogu))
			assert.Equal(t, "tocreate", createdDogu.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.Update(context.TODO(), dogu, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/tocreate/status", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdDogu := &k8sv1.Dogu{}
			require.NoError(t, json.Unmarshal(bytes, createdDogu))
			assert.Equal(t, "tocreate", createdDogu.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.UpdateStatus(context.TODO(), dogu, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/testdogu", request.URL.Path)

			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		err = dClient.Delete(context.TODO(), "testdogu", v1.DeleteOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_DeleteCollection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus", request.URL.Path)
			assert.Equal(t, "labelSelector=test", request.URL.RawQuery)
			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		err = dClient.DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_Patch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPatch, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/testdogu", request.URL.Path)
			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("test"), bytes)
			result, err := json.Marshal(k8sv1.Dogu{})
			require.NoError(t, err)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(result)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		patchData := []byte("test")

		// when
		_, err = dClient.Patch(context.TODO(), "testdogu", types.JSONPatchType, patchData, v1.PatchOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_Watch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)
			assert.Equal(t, "labelSelector=test&watch=true", request.URL.RawQuery)

			writer.Header().Add("content-type", "application/json")
			_, err := writer.Write([]byte("egal"))
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.Watch(context.TODO(), v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_UpdateSpecWithRetry(t *testing.T) {
	t.Run("should retry on conflict error", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "toUpdate", Namespace: "test"}, Spec: k8sv1.DoguSpec{Version: "1.0.0"}}

		firstPut := true

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// First update return conflict error
			if request.Method == http.MethodPut && firstPut {
				firstPut = false
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate", request.URL.Path)

				bytes, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				updatedDogu := &k8sv1.Dogu{}
				require.NoError(t, json.Unmarshal(bytes, updatedDogu))
				assert.Equal(t, "toUpdate", updatedDogu.Name)
				assert.Equal(t, "1.0.2", updatedDogu.Spec.Version)

				writer.Header().Add("content-type", "application/json")
				conflict := errors.NewConflict(schema.GroupResource{}, "toUpdate", assert.AnError)

				marshal, err := json.Marshal(conflict)
				require.NoError(t, err)

				writer.WriteHeader(409)
				_, err = writer.Write(marshal)
				require.NoError(t, err)
				return
			}

			// Get
			if request.Method == http.MethodGet {
				assert.Equal(t, "GET", request.Method)
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate", request.URL.Path)
				assert.Equal(t, http.NoBody, request.Body)

				writer.Header().Add("content-type", "application/json")
				doguRestart := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "toUpdate", Namespace: "test"}, Spec: k8sv1.DoguSpec{Version: "1.0.1"}}
				doguBytes, err := json.Marshal(doguRestart)
				require.NoError(t, err)
				writer.WriteHeader(200)
				_, err = writer.Write(doguBytes)
				require.NoError(t, err)
				return
			}

			// Retry
			if request.Method == http.MethodPut && !firstPut {
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate", request.URL.Path)

				bytes, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				updatedDogu := &k8sv1.Dogu{}
				require.NoError(t, json.Unmarshal(bytes, updatedDogu))
				assert.Equal(t, "toUpdate", updatedDogu.Name)
				assert.Equal(t, "1.0.2", updatedDogu.Spec.Version)

				writer.Header().Add("content-type", "application/json")
				writer.WriteHeader(200)
				_, err = writer.Write(bytes)
				require.NoError(t, err)
				return
			}
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.UpdateSpecWithRetry(context.TODO(), dogu, func(spec k8sv1.DoguSpec) k8sv1.DoguSpec {
			spec.Version = "1.0.2"
			return spec
		}, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_doguClient_UpdateStatusWithRetry(t *testing.T) {
	t.Run("should retry on conflict error", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "toUpdate", Namespace: "test"}, Spec: k8sv1.DoguSpec{Version: "1.0.0"}}

		firstPut := true

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// First update return conflict error
			if request.Method == http.MethodPut && firstPut {
				firstPut = false
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate/status", request.URL.Path)

				bytes, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				updatedDogu := &k8sv1.Dogu{}
				require.NoError(t, json.Unmarshal(bytes, updatedDogu))
				assert.Equal(t, "toUpdate", updatedDogu.Name)
				assert.Equal(t, true, updatedDogu.Status.Stopped)

				writer.Header().Add("content-type", "application/json")
				conflict := errors.NewConflict(schema.GroupResource{}, "toUpdate", assert.AnError)

				marshal, err := json.Marshal(conflict)
				require.NoError(t, err)

				writer.WriteHeader(409)
				_, err = writer.Write(marshal)
				require.NoError(t, err)
				return
			}

			// Get
			if request.Method == http.MethodGet {
				assert.Equal(t, "GET", request.Method)
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate", request.URL.Path)
				assert.Equal(t, http.NoBody, request.Body)

				writer.Header().Add("content-type", "application/json")
				doguRestart := &k8sv1.Dogu{ObjectMeta: v1.ObjectMeta{Name: "toUpdate", Namespace: "test"}, Status: k8sv1.DoguStatus{Stopped: false}}
				doguBytes, err := json.Marshal(doguRestart)
				require.NoError(t, err)
				writer.WriteHeader(200)
				_, err = writer.Write(doguBytes)
				require.NoError(t, err)
				return
			}

			// Retry
			if request.Method == http.MethodPut && !firstPut {
				assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/dogus/toUpdate/status", request.URL.Path)

				bytes, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				updatedDogu := &k8sv1.Dogu{}
				require.NoError(t, json.Unmarshal(bytes, updatedDogu))
				assert.Equal(t, "toUpdate", updatedDogu.Name)
				assert.Equal(t, true, updatedDogu.Status.Stopped)

				writer.Header().Add("content-type", "application/json")
				writer.WriteHeader(200)
				_, err = writer.Write(bytes)
				require.NoError(t, err)
				return
			}
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Dogus("test")

		// when
		_, err = dClient.UpdateStatusWithRetry(context.TODO(), dogu, func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
			status.Stopped = true
			return status
		}, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}
