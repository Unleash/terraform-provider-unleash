package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	unleash "github.com/Unleash/unleash-server-api-go/client"
)

// The generated unleash-server-api-go client has no feature flag endpoints, so feature flags are
// managed with plain HTTP calls that reuse the client's base URL, headers and HTTP client
// (including its auth header, app name header and concurrency limit).

// featureSchema is the subset of the admin API's featureSchema that this provider reads.
type featureSchema struct {
	Name           string                    `json:"name"`
	Type           string                    `json:"type"`
	Description    *string                   `json:"description"`
	Project        string                    `json:"project"`
	ImpressionData bool                      `json:"impressionData"`
	Archived       bool                      `json:"archived"`
	Stale          bool                      `json:"stale"`
	Environments   []featureEnvironmentState `json:"environments"`
}

type featureEnvironmentState struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// createFeatureRequest mirrors the admin API's createFeatureSchema.
type createFeatureRequest struct {
	Name           string `json:"name"`
	Type           string `json:"type,omitempty"`
	Description    string `json:"description"`
	ImpressionData bool   `json:"impressionData"`
}

// updateFeatureRequest mirrors the admin API's updateFeatureSchema.
type updateFeatureRequest struct {
	Type           string `json:"type,omitempty"`
	Description    string `json:"description"`
	ImpressionData bool   `json:"impressionData"`
}

type featureAPI struct {
	client *unleash.APIClient
}

func featureURL(project, name string) string {
	return fmt.Sprintf("/api/admin/projects/%s/features/%s", url.PathEscape(project), url.PathEscape(name))
}

func (a featureAPI) do(ctx context.Context, method, path string, body any, out any) (*http.Response, error) {
	cfg := a.client.GetConfig()
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("unleash client has no server configured")
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.Servers[0].URL+path, reader)
	if err != nil {
		return nil, err
	}
	for name, value := range cfg.DefaultHeader {
		req.Header.Set(name, value)
	}
	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return res, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return res, err
	}
	if res.StatusCode >= 300 {
		return res, fmt.Errorf("%s %s returned %s: %s", method, path, res.Status, string(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return res, fmt.Errorf("could not decode response from %s %s: %w", method, path, err)
		}
	}
	return res, nil
}

func (a featureAPI) get(ctx context.Context, project, name string) (*featureSchema, *http.Response, error) {
	var feature featureSchema
	res, err := a.do(ctx, http.MethodGet, featureURL(project, name), nil, &feature)
	if err != nil {
		return nil, res, err
	}
	return &feature, res, nil
}

func (a featureAPI) create(ctx context.Context, project string, body createFeatureRequest) (*http.Response, error) {
	return a.do(ctx, http.MethodPost, fmt.Sprintf("/api/admin/projects/%s/features", url.PathEscape(project)), body, nil)
}

func (a featureAPI) update(ctx context.Context, project, name string, body updateFeatureRequest) (*http.Response, error) {
	return a.do(ctx, http.MethodPut, featureURL(project, name), body, nil)
}

// archive moves the flag to the archive, which is what deleting a flag in the Unleash UI does.
func (a featureAPI) archive(ctx context.Context, project, name string) (*http.Response, error) {
	return a.do(ctx, http.MethodDelete, featureURL(project, name), nil, nil)
}

// deleteArchived permanently deletes a flag that is already archived.
func (a featureAPI) deleteArchived(ctx context.Context, name string) (*http.Response, error) {
	return a.do(ctx, http.MethodDelete, fmt.Sprintf("/api/admin/archive/%s", url.PathEscape(name)), nil, nil)
}
