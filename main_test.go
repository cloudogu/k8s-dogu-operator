package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockExiter struct {
	Error error `json:"error"`
}

func (e *mockExiter) Exit(err error) {
	e.Error = err
}

func Test_getK8sManagerOptions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		exiter := &mockExiter{}
		t.Setenv("WATCH_NAMESPACE", "default")

		getK8sManagerOptions(exiter)

		assert.Nil(t, exiter.Error)
	})
}
