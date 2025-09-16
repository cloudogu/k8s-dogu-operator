package initfx

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"

	"github.com/dlclark/regexp2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func GetAdditionalImages(configMapClient corev1.ConfigMapInterface) (resource.AdditionalImages, error) {
	ctx := context.Background()

	configMap, err := configMapClient.Get(ctx, config.OperatorAdditionalImagesConfigmapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error while getting configmap '%s': %w", config.OperatorAdditionalImagesConfigmapName, err)
	}

	additionalImageChownInitContainer, err := imageForKey(config.ChownInitImageConfigmapNameKey, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalExportModeContainer, err := imageForKey(config.ExporterImageConfigmapNameKey, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalMountsContainer, err := imageForKey(config.AdditionalMountsInitContainerImageConfigmapNameKey, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalImages := map[string]string{config.ChownInitImageConfigmapNameKey: additionalImageChownInitContainer,
		config.ExporterImageConfigmapNameKey:                      additionalExportModeContainer,
		config.AdditionalMountsInitContainerImageConfigmapNameKey: additionalMountsContainer}
	return additionalImages, nil
}

// imageTagValidator defines a regexp string that validates a container reference. These include:
//   - standard DNS rules
//   - optional hostnames
//   - optional port numbers like :30099
//   - optional tags
var imageTagValidationString = "^(?:(?=[^:\\/]{1,253})(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(?:\\.(?!-)[a-zA-Z0-9-]{1,63}(?<!-))*(?::[0-9]{1,5})?/)?((?![._-])(?:[a-z0-9._-]*)(?<![._-])(?:/(?![._-])[a-z0-9._-]*(?<![._-]))*)(?::(?![.-])[a-zA-Z0-9_.-]{1,128})?$"
var imageTagValidationRegexp, _ = regexp2.Compile(imageTagValidationString, regexp2.None)

// imageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
func imageForKey(key string, configMap *v1.ConfigMap) (string, error) {
	imageTag := configMap.Data[key]
	if imageTag == "" {
		return "", fmt.Errorf("configmap '%s' must not contain empty image name for key %s", config.OperatorAdditionalImagesConfigmapName, key)
	}

	err := verifyImageTag(imageTag)
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
