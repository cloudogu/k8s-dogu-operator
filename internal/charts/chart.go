package charts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	helmChart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
)

const (
	patchTemplateFileName  = "chart-patch-tpl.yaml"
	doguValuesMetaFileName = "dogu-values-metadata.yaml"
)

type ReleaseRef struct {
	Name      string
	Namespace string
}

type chart struct {
	raw  *helmChart.Chart
	ref  ReleaseRef
	caps *chartutil.Capabilities
}

// processDependencies applies Helm's dependency resolution to the chart in place: it evaluates
// subchart conditions/tags, prunes disabled dependencies, and merges import-values into values.
// This mirrors what Helm does during installation, so validation and rendering only consider what
// is actually applied. It mutates the chart (and the values map), so callers must operate on a chart
// that is not shared — the Service parses a fresh chart per use.
func (c *chart) processDependencies(values map[string]any) error {
	if err := chartutil.ProcessDependenciesWithMerge(c.raw, values); err != nil {
		return fmt.Errorf("failed to process chart dependencies: %w", err)
	}

	return nil
}

func (c *chart) render(values map[string]any, restConfig *rest.Config, op flux.ReleaseOperation) ([]*unstructured.Unstructured, error) {
	if err := c.processDependencies(values); err != nil {
		return nil, fmt.Errorf("failed to process dependencies: %w", err)
	}

	releaseOptions := chartutil.ReleaseOptions{
		Name:      c.ref.Name,
		Namespace: c.ref.Namespace,
		IsInstall: op.IsInstall,
		IsUpgrade: op.IsUpgrade,
		Revision:  op.Revision,
	}

	rv, err := chartutil.ToRenderValuesWithSchemaValidation(c.raw, values, releaseOptions, c.caps, true)
	if err != nil {
		return nil, fmt.Errorf("failed to render values for chart: %w", err)
	}

	manifests, err := engine.RenderWithClient(c.raw, rv, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart against cluster: %w", err)
	}

	resList, err := parseRenderedFilesToObjects(manifests)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest strings to objects: %w", err)
	}

	return resList, nil
}

func (c *chart) validateValues(values map[string]any) error {
	if err := c.processDependencies(values); err != nil {
		return fmt.Errorf("failed to process dependencies: %w", err)
	}

	coalescedValues, err := chartutil.CoalesceValues(c.raw, values)
	if err != nil {
		return fmt.Errorf("failed coalescing values: %w", err)
	}

	if vErr := chartutil.ValidateAgainstSchema(c.raw, coalescedValues); vErr != nil {
		return fmt.Errorf("schema validation error occurred: %w", vErr)
	}

	return nil
}

func (c *chart) getDoguValuesMeta() ([]byte, bool) {
	return c.getTemplateFile(doguValuesMetaFileName)
}

func (c *chart) getChartPatchTpl() ([]byte, bool) {
	return c.getTemplateFile(patchTemplateFileName)
}

func (c *chart) getTemplateFile(name string) ([]byte, bool) {
	for _, f := range c.raw.Files {
		if filepath.Base(f.Name) == name {
			return f.Data, true
		}
	}

	return nil, false
}

// parseRenderedFilesToObjects parses the map[string]string output from engine.RenderWithClient
// into a flat slice of client.Object.
func parseRenderedFilesToObjects(renderedFiles map[string]string) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	for fileName, content := range renderedFiles {
		// Skip empty files, NOTES.txt, and template partials (_helpers.tpl)
		if !isRenderableYAML(fileName, content) {
			continue
		}

		// bufferSize is set to defaultBufferSize of the bufio package from go
		// NewYAMLOrJSONDecoder natively supports multi-docs ("---")
		decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(content), 4096)

		for {
			u := &unstructured.Unstructured{}

			if err := decoder.Decode(u); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				return nil, fmt.Errorf("failed to decode manifest in %s: %w", fileName, err)
			}

			if len(u.Object) == 0 {
				continue
			}

			objects = append(objects, u)
		}
	}

	return objects, nil
}

// isRenderableYAML checks if a rendered file is a valid Kubernetes manifest template
func isRenderableYAML(fileName string, content string) bool {
	// Skip empty string content
	if strings.TrimSpace(content) == "" {
		return false
	}

	// Only process .yaml and .yml files
	ext := strings.ToLower(filepath.Ext(fileName))

	return ext == ".yaml" || ext == ".yml"
}
