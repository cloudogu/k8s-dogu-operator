package controllers

import (
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"testing"
)

func TestHTTPDoguRegistry_GetDogu(t *testing.T) {
	doguRegistry := NewHTTPDoguRegistry("user", "pw", "url")
	err := errors.New("err")

	t.Run("Error on Do", func(t *testing.T) {
		httpMock := &mocks.HttpClient{}
		doguRegistry.HttpClient = httpMock
		httpMock.Mock.On("Do", mock.Anything).Return(nil, err)

		result, err := doguRegistry.GetDogu(doguCr)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("Error on status code >= 300", func(t *testing.T) {
		response := &http.Response{StatusCode: 300}
		httpMock := &mocks.HttpClient{}
		doguRegistry.HttpClient = httpMock
		httpMock.Mock.On("Do", mock.Anything).Return(response, nil)

		result, err := doguRegistry.GetDogu(doguCr)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("Error reading body", func(t *testing.T) {
		response := &http.Response{StatusCode: 200}
		httpMock := &mocks.HttpClient{}
		doguRegistry.HttpClient = httpMock
		doguRegistry.IoReader = ReadFailure
		httpMock.Mock.On("Do", mock.Anything).Return(response, nil)

		result, err := doguRegistry.GetDogu(doguCr)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("Error unmarshal dogu", func(t *testing.T) {
		response := &http.Response{StatusCode: 200}
		doguRegistry.IoReader = ReadSuccessNoDogu
		httpMock := &mocks.HttpClient{}
		doguRegistry.HttpClient = httpMock
		httpMock.Mock.On("Do", mock.Anything).Return(response, nil)

		result, err := doguRegistry.GetDogu(doguCr)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("Successful get dogu", func(t *testing.T) {
		response := &http.Response{StatusCode: 200}
		httpMock := &mocks.HttpClient{}
		doguRegistry.HttpClient = httpMock
		httpMock.Mock.On("Do", mock.Anything).Return(response, nil)
		doguRegistry.HttpClient = httpMock
		doguRegistry.IoReader = ReadSuccessDogu

		result, err := doguRegistry.GetDogu(doguCr)

		assert.Equal(t, ldapDogu, result)
		assert.NoError(t, err)
	})
}

func ReadSuccessDogu(_ io.Reader) ([]byte, error) {
	return ldapBytes, nil
}

func ReadSuccessNoDogu(_ io.Reader) ([]byte, error) {
	return make([]byte, 8), nil
}

func ReadFailure(_ io.Reader) ([]byte, error) {
	return ldapBytes, errors.New("error")
}
