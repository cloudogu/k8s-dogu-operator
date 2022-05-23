package registry

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
)

// httpDoguRegistry is a component which can communicate with a dogu registry.
// It is used for pulling the dogu descriptor via http
type httpDoguRegistry struct {
	username string
	password string
	url      string
}

// New creates a new instance of httpDoguRegistry
func New(username string, password string, url string) *httpDoguRegistry {
	return &httpDoguRegistry{
		username: username,
		password: password,
		url:      url,
	}
}

// GetDogu fetches a dogu.json with a given dogu custom resource. It uses basic auth for registry authentication
func (h httpDoguRegistry) GetDogu(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", h.url, doguResource.Spec.Name, doguResource.Spec.Version), nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}
	req.SetBasicAuth(h.username, h.password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error doing request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dogu registry returned status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var dogu core.Dogu
	err = json.Unmarshal(body, &dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dogu: %w", err)
	}

	return &dogu, nil
}
