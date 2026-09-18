package initfx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewChartArtifactHTTPClient(t *testing.T) {
	client := NewChartArtifactHTTPClient()

	assert.Equal(t, 30*time.Second, client.Timeout)
	assert.ErrorIs(t, client.CheckRedirect(&http.Request{}, nil), http.ErrUseLastResponse)
}
