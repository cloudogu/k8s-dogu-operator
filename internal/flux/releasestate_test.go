package flux

import (
	"testing"

	helmflux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDesiredRelease(t *testing.T) {
	release := &helmflux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Generation: 3}}

	desired := NewDesiredRelease("3.86.2-8", release)

	assert.Equal(t, DesiredRelease{ChartVersion: "3.86.2-8", Generation: 3}, desired)
}
