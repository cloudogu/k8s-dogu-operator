package util

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/apps/v1"
)

const PreviousDoguVersionAnnotationKey = "k8s.cloudogu.com/dogu-previous-version"

// GetMapKeysAsString returns the key of a map as a string in form: "key1, key2, key3".
func GetMapKeysAsString(input map[string]string) string {
	output := ""

	for key := range input {
		output = fmt.Sprintf("%s, %s", output, key)
	}

	return strings.TrimLeft(output, ", ")
}

func SetPreviousDoguVersionInAnnotations(previousDoguVersion string, deployment *v1.Deployment) {
	if deployment.Annotations == nil {
		deployment.Annotations = map[string]string{}
	}
	deployment.Annotations[PreviousDoguVersionAnnotationKey] = previousDoguVersion
}
