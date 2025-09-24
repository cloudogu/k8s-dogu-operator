package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
)

func TestGetMapKeysAsString(t *testing.T) {
	type args struct {
		input map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty string for nil", args{input: nil}, ""},
		{"empty string for empty map", args{input: map[string]string{}}, ""},
		{"single key", args{input: map[string]string{"key": "value"}}, "key"},
		{"many keys", args{input: map[string]string{"key1": "value1", "key2": "value2"}}, "key1, key2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wandSplit := strings.Split(strings.ReplaceAll(tt.want, " ", ""), ",")
			got := GetMapKeysAsString(tt.args.input)
			resultSplit := strings.Split(strings.ReplaceAll(got, " ", ""), ",")

			if len(wandSplit) != len(resultSplit) {
				t.Errorf("GetMapKeysAsString() = %v, want %v", got, tt.want)
			}

			for _, want := range wandSplit {
				if !containsInSlice(resultSplit, want) {
					t.Errorf("GetMapKeysAsString() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func containsInSlice(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}

	return false
}

func TestSetPreviousDoguVersionInAnnotations(t *testing.T) {
	t.Run("Successfully set dogu version in annotation", func(t *testing.T) {
		deployment := &v1.Deployment{}
		SetPreviousDoguVersionInAnnotations("1.0.0", deployment)

		assert.Equal(t, "1.0.0", deployment.Annotations[PreviousDoguVersionAnnotationKey])
	})
}
