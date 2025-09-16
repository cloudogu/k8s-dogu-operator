package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newApp(t *testing.T) {
	// given
	// TODO overwrite operator config
	// TODO overwrite manager options
	// TODO overwrite kubernetes clients

	// when
	newApp := newApp()

	// then
	assert.NoError(t, newApp.Err())
}
