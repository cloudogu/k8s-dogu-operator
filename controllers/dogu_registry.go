package controllers

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
)

type HttpDoguRegistry struct {
	username string
	password string
	url      string
}

func NewHttpDoguRegistry(username string, password string, url string) *HttpDoguRegistry {
	return &HttpDoguRegistry{
		username: username,
		password: password,
		url:      url,
	}
}

func (h HttpDoguRegistry) GetDogu(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", h.url, doguResource.Spec.Name, doguResource.Spec.Version), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(h.username, h.password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dogu registry returned status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dogu core.Dogu
	err = json.Unmarshal(body, &dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dogu: %w", err)
	}

	return &dogu, nil
}
