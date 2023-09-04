package util

import (
	"context"
	"fmt"

	"github.com/dlclark/regexp2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// OperatorAdditionalImagesConfigmapName contains the configmap name which consists of auxiliary yet necessary container images.
	OperatorAdditionalImagesConfigmapName = "k8s-ces-additional-images"
	ChownInitImageConfigmapNameKey        = "chownInitImage"
)

// imageTagValidator defines a regexp string that validates a container reference. These include:
//   - standard DNS rules
//   - optional hostnames
//   - optional port numbers like :30099
//   - optional tags
var imageTagValidationString = "^(?:(?=[^:\\/]{1,253})(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(?:\\.(?!-)[a-zA-Z0-9-]{1,63}(?<!-))*(?::[0-9]{1,5})?/)?((?![._-])(?:[a-z0-9._-]*)(?<![._-])(?:/(?![._-])[a-z0-9._-]*(?<![._-]))*)(?::(?![.-])[a-zA-Z0-9_.-]{1,128})?$"
var imageTagValidationRegexp, _ = regexp2.Compile(imageTagValidationString, regexp2.None)

type additionalImageGetter struct {
	configmapClient v1.ConfigMapInterface
}

func NewAdditionalImageGetter(configmapClient v1.ConfigMapInterface) *additionalImageGetter {
	return &additionalImageGetter{configmapClient: configmapClient}
}

// ImageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
func (adig *additionalImageGetter) ImageForKey(ctx context.Context, key string) (string, error) {
	configMap, err := adig.configmapClient.Get(ctx, OperatorAdditionalImagesConfigmapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error while getting configmap '%s': %w", OperatorAdditionalImagesConfigmapName, err)
	}

	imageTag := configMap.Data[ChownInitImageConfigmapNameKey]
	if imageTag == "" {
		return "", fmt.Errorf("configmap '%s' must not contain empty chown init image name", OperatorAdditionalImagesConfigmapName)
	}

	err = verifyImageTag(imageTag)
	if err != nil {
		return "", fmt.Errorf("configmap '%s' contains an invalid image tag: %w", OperatorAdditionalImagesConfigmapName, err)
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