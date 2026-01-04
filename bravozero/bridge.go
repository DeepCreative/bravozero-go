package bravozero

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// FileInfo represents information about a file in the VFS.
type FileInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	IsDirectory bool      `json:"isDirectory"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	Permissions string    `json:"permissions"`
}

// DirectoryListing represents a listing of files in a directory.
type DirectoryListing struct {
	Path       string     `json:"path"`
	Files      []FileInfo `json:"files"`
	TotalCount int        `json:"totalCount"`
}

// SyncStatus represents VFS synchronization status.
type SyncStatus struct {
	Path           string    `json:"path"`
	Synced         bool      `json:"synced"`
	LastSyncAt     time.Time `json:"lastSyncAt,omitempty"`
	PendingChanges int       `json:"pendingChanges"`
}

// BridgeClient provides access to the Forge Bridge API.
type BridgeClient struct {
	baseURL       string
	apiKey        string
	agentID       string
	authenticator *PersonaAuthenticator
	httpClient    *http.Client
}

// NewBridgeClient creates a new Forge Bridge client.
func NewBridgeClient(
	baseURL, apiKey, agentID string,
	auth *PersonaAuthenticator,
	timeoutSeconds int,
) *BridgeClient {
	return &BridgeClient{
		baseURL:       baseURL + "/v1/bridge",
		apiKey:        apiKey,
		agentID:       agentID,
		authenticator: auth,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

func (c *BridgeClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Agent-ID", c.agentID)
	req.Header.Set("User-Agent", "bravozero-go/1.0.0")

	if c.authenticator != nil {
		attestation, err := c.authenticator.CreateAttestation("")
		if err != nil {
			return nil, fmt.Errorf("failed to create attestation: %w", err)
		}
		req.Header.Set("X-Persona-Attestation", attestation)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, &RateLimitError{RetryAfter: 60}
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// ListFiles lists files in a directory.
func (c *BridgeClient) ListFiles(ctx context.Context, path string, recursive bool, pattern string) (*DirectoryListing, error) {
	params := url.Values{}
	params.Set("path", path)
	if recursive {
		params.Set("recursive", "true")
	}
	if pattern != "" {
		params.Set("pattern", pattern)
	}

	resp, err := c.doRequest(ctx, "GET", "/files?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Path       string `json:"path"`
		Files      []struct {
			Path        string `json:"path"`
			Name        string `json:"name"`
			Size        int64  `json:"size"`
			IsDirectory bool   `json:"isDirectory"`
			ModifiedAt  string `json:"modifiedAt"`
			CreatedAt   string `json:"createdAt"`
			Permissions string `json:"permissions"`
		} `json:"files"`
		TotalCount int `json:"totalCount"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	files := make([]FileInfo, len(data.Files))
	for i, f := range data.Files {
		modifiedAt, _ := time.Parse(time.RFC3339, f.ModifiedAt)
		createdAt, _ := time.Parse(time.RFC3339, f.CreatedAt)
		files[i] = FileInfo{
			Path:        f.Path,
			Name:        f.Name,
			Size:        f.Size,
			IsDirectory: f.IsDirectory,
			ModifiedAt:  modifiedAt,
			CreatedAt:   createdAt,
			Permissions: f.Permissions,
		}
	}

	return &DirectoryListing{
		Path:       data.Path,
		Files:      files,
		TotalCount: data.TotalCount,
	}, nil
}

// ReadFile reads a file's contents.
func (c *BridgeClient) ReadFile(ctx context.Context, path string) (string, error) {
	params := url.Values{}
	params.Set("path", path)

	resp, err := c.doRequest(ctx, "GET", "/file?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return data.Content, nil
}

// ReadFileBytes reads a file as bytes.
func (c *BridgeClient) ReadFileBytes(ctx context.Context, path string) ([]byte, error) {
	params := url.Values{}
	params.Set("path", path)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/file/bytes?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Agent-ID", c.agentID)

	if c.authenticator != nil {
		attestation, err := c.authenticator.CreateAttestation("")
		if err != nil {
			return nil, fmt.Errorf("failed to create attestation: %w", err)
		}
		req.Header.Set("X-Persona-Attestation", attestation)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// WriteFile writes content to a file.
func (c *BridgeClient) WriteFile(ctx context.Context, path, content string, createDirs bool) (*FileInfo, error) {
	body := map[string]interface{}{
		"path":       path,
		"content":    content,
		"createDirs": createDirs,
	}

	resp, err := c.doRequest(ctx, "PUT", "/file", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Path        string `json:"path"`
		Name        string `json:"name"`
		Size        int64  `json:"size"`
		IsDirectory bool   `json:"isDirectory"`
		ModifiedAt  string `json:"modifiedAt"`
		CreatedAt   string `json:"createdAt"`
		Permissions string `json:"permissions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	modifiedAt, _ := time.Parse(time.RFC3339, data.ModifiedAt)
	createdAt, _ := time.Parse(time.RFC3339, data.CreatedAt)

	return &FileInfo{
		Path:        data.Path,
		Name:        data.Name,
		Size:        data.Size,
		IsDirectory: data.IsDirectory,
		ModifiedAt:  modifiedAt,
		CreatedAt:   createdAt,
		Permissions: data.Permissions,
	}, nil
}

// DeleteFile deletes a file.
func (c *BridgeClient) DeleteFile(ctx context.Context, path string) error {
	params := url.Values{}
	params.Set("path", path)

	resp, err := c.doRequest(ctx, "DELETE", "/file?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Sync triggers VFS synchronization.
func (c *BridgeClient) Sync(ctx context.Context, path string) (*SyncStatus, error) {
	if path == "" {
		path = "/"
	}

	body := map[string]string{"path": path}

	resp, err := c.doRequest(ctx, "POST", "/sync", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Path           string `json:"path"`
		Synced         bool   `json:"synced"`
		LastSyncAt     string `json:"lastSyncAt"`
		PendingChanges int    `json:"pendingChanges"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	lastSync, _ := time.Parse(time.RFC3339, data.LastSyncAt)

	return &SyncStatus{
		Path:           data.Path,
		Synced:         data.Synced,
		LastSyncAt:     lastSync,
		PendingChanges: data.PendingChanges,
	}, nil
}
