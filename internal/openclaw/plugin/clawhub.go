package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ClawHubBaseURL = "https://hub.openclaw.ai/api/v1"

type ClawHubClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type PluginInfo struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Version     string      `json:"version"`
	Author      string      `json:"author"`
	Repository  string      `json:"repository"`
	Downloads   int         `json:"downloads"`
	Rating      float64     `json:"rating"`
	Tags        []string    `json:"tags"`
	ConfigKeys  []ConfigKey `json:"configKeys"`
	Installed   bool        `json:"installed"`
}

type SearchResult struct {
	Plugins []PluginInfo `json:"plugins"`
	Total   int          `json:"total"`
	Page    int          `json:"page"`
}

type InstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version"`
}

func NewClawHubClient(apiKey string) *ClawHubClient {
	return &ClawHubClient{
		baseURL: ClawHubBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ClawHubClient) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *ClawHubClient) doRequest(method, path string, body io.Reader, result interface{}) error {
	url := c.baseURL + path

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *ClawHubClient) ListPlugins(category string, page int) (*SearchResult, error) {
	path := fmt.Sprintf("/plugins?category=%s&page=%d", category, page)
	var result SearchResult
	if err := c.doRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClawHubClient) SearchPlugins(query string, page int) (*SearchResult, error) {
	path := fmt.Sprintf("/plugins/search?q=%s&page=%d", query, page)
	var result SearchResult
	if err := c.doRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClawHubClient) GetPlugin(id string) (*PluginInfo, error) {
	var result PluginInfo
	path := fmt.Sprintf("/plugins/%s", id)
	if err := c.doRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClawHubClient) GetPluginVersions(id string) ([]string, error) {
	var result struct {
		Versions []string `json:"versions"`
	}
	path := fmt.Sprintf("/plugins/%s/versions", id)
	if err := c.doRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result.Versions, nil
}

func (c *ClawHubClient) InstallPlugin(id string, version string) (*InstallResult, error) {
	var result InstallResult
	path := fmt.Sprintf("/plugins/%s/install", id)
	if version != "" {
		path += "?version=" + version
	}
	if err := c.doRequest("POST", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClawHubClient) UninstallPlugin(id string) error {
	path := fmt.Sprintf("/plugins/%s/uninstall", id)
	return c.doRequest("POST", path, nil, nil)
}

func (c *ClawHubClient) GetFeatured() ([]PluginInfo, error) {
	var result []PluginInfo
	if err := c.doRequest("GET", "/plugins/featured", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClawHubClient) GetPopular(category string, limit int) ([]PluginInfo, error) {
	path := fmt.Sprintf("/plugins/popular?category=%s&limit=%d", category, limit)
	var result []PluginInfo
	if err := c.doRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClawHubClient) CheckUpdates(installed []string) (map[string]string, error) {
	type checkRequest struct {
		Installed []string `json:"installed"`
	}
	type checkResponse struct {
		Updates map[string]string `json:"updates"`
	}

	path := "/plugins/check-updates"
	bodyBytes, _ := json.Marshal(checkRequest{Installed: installed})

	var result checkResponse
	if err := c.doRequest("POST", path, bytes.NewReader(bodyBytes), &result); err != nil {
		return nil, err
	}
	return result.Updates, nil
}

func (c *ClawHubClient) SubmitReview(pluginID string, rating float64, comment string) error {
	type reviewRequest struct {
		Rating  float64 `json:"rating"`
		Comment string  `json:"comment"`
	}

	path := fmt.Sprintf("/plugins/%s/reviews", pluginID)
	bodyBytes, _ := json.Marshal(reviewRequest{Rating: rating, Comment: comment})
	return c.doRequest("POST", path, bytes.NewReader(bodyBytes), nil)
}
