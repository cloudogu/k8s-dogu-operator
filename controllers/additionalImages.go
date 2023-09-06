package controllers

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dlclark/regexp2"

	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
)

// imageTagValidator defines a regexp string that validates a container reference. These include:
//   - standard DNS rules
//   - optional hostnames
//   - optional port numbers like :30099
//   - optional tags
var imageTagValidationString = "^(?:(?=[^:\\/]{1,253})(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(?:\\.(?!-)[a-zA-Z0-9-]{1,63}(?<!-))*(?::[0-9]{1,5})?/)?((?![._-])(?:[a-z0-9._-]*)(?<![._-])(?:/(?![._-])[a-z0-9._-]*(?<![._-]))*)(?::(?![.-])[a-zA-Z0-9_.-]{1,128})?$"
var imageTagValidationRegexp, _ = regexp2.Compile(imageTagValidationString, regexp2.None)

type additionalImageGetter struct {
	configmapClient kubernetes.Interface
	namespace       string
}

func newAdditionalImageGetter(client kubernetes.Interface, namespace string) *additionalImageGetter {
	return &additionalImageGetter{configmapClient: client, namespace: namespace}
}

// imageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
func (adig *additionalImageGetter) imageForKey(ctx context.Context, key string) (string, error) {
	configMap, err := adig.configmapClient.CoreV1().
		ConfigMaps(adig.namespace).
		Get(ctx, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error while getting configmap '%s': %w", config.OperatorAdditionalImagesConfigmapName, err)
	}

	imageTag := configMap.Data[key]
	if imageTag == "" {
		return "", fmt.Errorf("configmap '%s' must not contain empty chown init image name", config.OperatorAdditionalImagesConfigmapName)
	}

	err = verifyImageTag(imageTag)
	if err != nil {
		return "", fmt.Errorf("configmap '%s' contains an invalid image tag: %w", config.OperatorAdditionalImagesConfigmapName, err)
	}

	return imageTag, nil
}

func verifyImageTag(imageTag string) error {
	matched, err := imageTagValidationRegexp.MatchString(imageTag)
	if err != nil {
		return fmt.Errorf("image tag validation of %s failed: %w", imageTag, err)
	}
	if !matched {
		return fmt.Errorf("image tag '%s' seems invalid (please compare it with the image tag specs)", imageTag)
	}
	return nil
}
