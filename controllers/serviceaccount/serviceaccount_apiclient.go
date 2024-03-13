package serviceaccount

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

type serviceAccountApiClient interface {
	createServiceAccount(baseUrl string, apiKey string, consumer string, params []string) (Credentials, error)
	deleteServiceAccount(baseUrl string, apiKey string, consumer string) error
}

type apiClient struct{}

type createRequest struct {
	Consumer string   `json:"consumer"`
	Params   []string `json:"params"`
}

type Credentials map[string]string

func (ac *apiClient) createServiceAccount(baseUrl string, apiKey string, consumer string, params []string) (Credentials, error) {
	jsonData, err := json.Marshal(createRequest{
		Consumer: consumer,
		Params:   params,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling json-body: %w", err)
	}

	req, err := http.NewRequest("POST", baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CES-SA-API-KEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while sending request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request was not successful: %s", resp.Status)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var credentials Credentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return nil, fmt.Errorf("error parsing credentials from response: %w", err)
	}

	return credentials, nil
}

func (ac *apiClient) deleteServiceAccount(baseUrl string, apiKey string, consumer string) error {
	req, err := http.NewRequest("DELETE", path.Join(baseUrl, consumer), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-CES-SA-API-KEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while sending request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request was not successful: %s", resp.Status)
	}

	return nil
}
