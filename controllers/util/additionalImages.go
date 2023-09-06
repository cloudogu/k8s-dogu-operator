package util

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dlclark/regexp2"
)

const (
	// OperatorAdditionalImagesConfigmapName contains the configmap name which consists of auxiliary yet necessary container images.
	OperatorAdditionalImagesConfigmapName = "k8s-dogu-operator-additional-images"
	// ChownInitImageConfigmapNameKey contains the key to retrieve the chown init container image from the OperatorAdditionalImagesConfigmapName configmap.
	ChownInitImageConfigmapNameKey = "chownInitImage"
)

// imageTagValidator defines a regexp string that validates a container reference. These include:
//   - standard DNS rules
//   - optional hostnames
//   - optional port numbers like :30099
//   - optional tags
var imageTagValidationString = "^(?:(?=[^:\\/]{1,253})(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(?:\\.(?!-)[a-zA-Z0-9-]{1,63}(?<!-))*(?::[0-9]{1,5})?/)?((?![._-])(?:[a-z0-9._-]*)(?<![._-])(?:/(?![._-])[a-z0-9._-]*(?<![._-]))*)(?::(?![.-])[a-zA-Z0-9_.-]{1,128})?$"
var imageTagValidationRegexp, _ = regexp2.Compile(imageTagValidationString, regexp2.None)

type additionalImageGetter struct {
	configmapClient client.Client
	namespace       string
}

func NewAdditionalImageGetter(client client.Client, namespace string) *additionalImageGetter {
	return &additionalImageGetter{configmapClient: client, namespace: namespace}
}

// ImageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
func (adig *additionalImageGetter) ImageForKey(ctx context.Context, key string) (string, error) {
	configMap := corev1.ConfigMap{}
	id := types.NamespacedName{Name: OperatorAdditionalImagesConfigmapName, Namespace: adig.namespace}
	err := adig.configmapClient.Get(ctx, id, &configMap)
	if err != nil {
		return "", fmt.Errorf("error while getting configmap '%s': %w", OperatorAdditionalImagesConfigmapName, err)
	}

	imageTag := configMap.Data[key]
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
