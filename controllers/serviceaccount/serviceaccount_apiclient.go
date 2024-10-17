package serviceaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const apiKeyHeader = "X-CES-SA-API-KEY"

type serviceAccountApiClient interface {
	createServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string, params []string) (Credentials, error)
	deleteServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string) error
}

type apiClient struct{}

type createRequest struct {
	Consumer string   `json:"consumer"`
	Params   []string `json:"params"`
}

type Credentials map[string]string

func (ac *apiClient) createServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string, params []string) (Credentials, error) {
	jsonData, err := json.Marshal(createRequest{
		Consumer: consumer,
		Params:   params,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling json-body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, apiKey)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while sending request: %w", err)
	}
	defer closeBody(ctx, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request was not successful: %s", resp.Status)
	}

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

func (ac *apiClient) deleteServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string) error {
	fullUrl, err := url.JoinPath(baseUrl, consumer)
	if err != nil {
		return fmt.Errorf("error creating url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set(apiKeyHeader, apiKey)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while sending request: %w", err)
	}
	defer closeBody(ctx, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request was not successful: %s", resp.Status)
	}

	return nil
}

func closeBody(ctx context.Context, c io.Closer) {
	logger := log.FromContext(ctx)
	if err := c.Close(); err != nil {
		logger.Error(err, "closed http body with error")
	}
}
