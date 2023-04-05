package annotation_test

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
)

func getTestFileMap(t *testing.T) map[string]string {
	testFiles := map[string]string{}

	testdataDir := "./testdata"
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(info.Name(), "test_") && strings.Contains(info.Name(), ".json") {
			expectedServiceFile := strings.Replace(info.Name(), ".json", "_expected.yaml", 1)
			testFilePath := fmt.Sprintf("%s/%s", testdataDir, info.Name())
			expectedServiceFilePath := fmt.Sprintf("%s/%s", testdataDir, expectedServiceFile)

			_, err := os.Stat(fmt.Sprintf("%s/%s", testdataDir, expectedServiceFile))
			if err == nil {
				testFiles[testFilePath] = expectedServiceFilePath
			} else if errors.Is(err, os.ErrNotExist) {
				testFiles[testFilePath] = ""
			} else {
				return err
			}
		}

		return nil
	})
	require.NoError(t, err)

	return testFiles
}

func getImageConfigFromTestFile(t *testing.T, fileName string) *imagev1.Config {
	data, err := os.ReadFile(fileName)
	require.NoError(t, err)

	var config imagev1.Config
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	return &config
}

func getExpectedService(t *testing.T, fileName string) *corev1.Service {
	if fileName == "" {
		return &corev1.Service{}
	}

	data, err := os.ReadFile(fileName)
	require.NoError(t, err)

	var service corev1.Service
	err = yaml.Unmarshal(data, &service)
	require.NoError(t, err)

	return &service
}

func TestCesServiceAnnotator_AnnotateService(t *testing.T) {
	testFiles := getTestFileMap(t)

	// this test iterates over all [test_*.json] files in the testdata directory. Every JSON contains an image
	// configuration which is used to generate our ces services. Each test has a related [test_*_expected.yaml] which
	// contains the resulting service as yaml. In the end the generated and expected service annotation are compared.
	for testFile, expectedFile := range testFiles {
		t.Run(fmt.Sprintf("successfully annotate service using input config [%s] and compare against [%s]", testFile, expectedFile), func(t *testing.T) {
			// given
			config := getImageConfigFromTestFile(t, testFile)
			service := getExpectedService(t, "./testdata/input_service.yaml")
			expectedService := getExpectedService(t, expectedFile)

			annotator := annotation.CesServiceAnnotator{}

			// when
			err := annotator.AnnotateService(service, config)
			require.NoError(t, err)

			// then
			assert.Equal(t, expectedService.Annotations, service.Annotations)
		})
	}

	t.Run("Annotating fails with invalid port definition", func(t *testing.T) {
		// given
		config := &imagev1.Config{
			ExposedPorts: map[string]struct{}{
				"tcp": {},
			},
		}
		service := &corev1.Service{}
		annotator := annotation.CesServiceAnnotator{}

		// when
		err := annotator.AnnotateService(service, config)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"tcp\": invalid syntax")
	})

	t.Run("Annotating fails with invalid environment variable", func(t *testing.T) {
		// given
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS-invalidEnvironmentVariable",
			},
		}
		service := &corev1.Service{}
		annotator := annotation.CesServiceAnnotator{}

		// when
		err := annotator.AnnotateService(service, config)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "environment variable [SERVICE_TAGS-invalidEnvironmentVariable] needs to be in form NAME=VALUE")
	})

	t.Run("Annotating fails with invalid json in additional services", func(t *testing.T) {
		// given
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_ADDITIONAL_SERVICES='\"name\": \"docker-registry\", \"location\": \"v2\", \"pass\": \"nexus/repository/docker-registry/v2/\"}]'",
			},
		}
		service := &corev1.Service{}
		annotator := annotation.CesServiceAnnotator{}

		// when
		err := annotator.AnnotateService(service, config)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal additional services: invalid character '\\'' looking for beginning of value")
	})
}
