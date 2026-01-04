package bravozero

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MemoryType represents the type of a memory.
type MemoryType string

const (
	MemoryTypeEpisodic   MemoryType = "episodic"
	MemoryTypeSemantic   MemoryType = "semantic"
	MemoryTypeProcedural MemoryType = "procedural"
	MemoryTypeWorking    MemoryType = "working"
)

// ConsolidationState represents the consolidation state of a memory.
type ConsolidationState string

const (
	ConsolidationActive       ConsolidationState = "active"
	ConsolidationConsolidating ConsolidationState = "consolidating"
	ConsolidationConsolidated ConsolidationState = "consolidated"
	ConsolidationDecaying     ConsolidationState = "decaying"
	ConsolidationDormant      ConsolidationState = "dormant"
)

// Memory represents a memory from the Trace Manifold.
type Memory struct {
	ID                 string                 `json:"id"`
	Content            string                 `json:"content"`
	MemoryType         MemoryType             `json:"memoryType"`
	Importance         float64                `json:"importance"`
	Strength           float64                `json:"strength"`
	ConsolidationState ConsolidationState     `json:"consolidationState"`
	Namespace          string                 `json:"namespace"`
	Tags               []string               `json:"tags"`
	CreatedAt          time.Time              `json:"createdAt"`
	LastAccessedAt     time.Time              `json:"lastAccessedAt"`
	AccessCount        int                    `json:"accessCount"`
	Embedding          []float64              `json:"embedding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// MemoryQueryResult represents a memory with its relevance score.
type MemoryQueryResult struct {
	Memory    Memory  `json:"memory"`
	Relevance float64 `json:"relevance"`
}

// Edge represents an edge between two memories.
type Edge struct {
	SourceID           string    `json:"sourceId"`
	TargetID           string    `json:"targetId"`
	Relationship       string    `json:"relationship"`
	Strength           float64   `json:"strength"`
	CreatedAt          time.Time `json:"createdAt"`
	LastStrengthenedAt time.Time `json:"lastStrengthenedAt"`
}

// RecordRequest represents a request to record a memory.
type RecordRequest struct {
	Content    string                 `json:"content"`
	MemoryType MemoryType             `json:"memoryType"`
	Importance float64                `json:"importance"`
	Namespace  string                 `json:"namespace"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// QueryRequest represents a request to query memories.
type QueryRequest struct {
	Query        string       `json:"query"`
	Limit        int          `json:"limit"`
	MinRelevance float64      `json:"minRelevance"`
	MemoryTypes  []MemoryType `json:"memoryTypes,omitempty"`
	Namespace    string       `json:"namespace,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
}

// MemoryClient provides access to the Memory Service API.
type MemoryClient struct {
	baseURL       string
	apiKey        string
	agentID       string
	authenticator *PersonaAuthenticator
	httpClient    *http.Client
}

// NewMemoryClient creates a new Memory Service client.
func NewMemoryClient(
	baseURL, apiKey, agentID string,
	auth *PersonaAuthenticator,
	timeoutSeconds int,
) *MemoryClient {
	return &MemoryClient{
		baseURL:       baseURL + "/v1/memory",
		apiKey:        apiKey,
		agentID:       agentID,
		authenticator: auth,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

func (c *MemoryClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
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

// Record records a new memory to the Trace Manifold.
func (c *MemoryClient) Record(ctx context.Context, req RecordRequest) (*Memory, error) {
	if req.MemoryType == "" {
		req.MemoryType = MemoryTypeSemantic
	}
	if req.Importance == 0 {
		req.Importance = 0.5
	}
	if req.Namespace == "" {
		req.Namespace = c.agentID
	}

	resp, err := c.doRequest(ctx, "POST", "/record", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseMemory(resp.Body)
}

// Query queries memories by semantic similarity.
func (c *MemoryClient) Query(ctx context.Context, req QueryRequest) ([]MemoryQueryResult, error) {
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.MinRelevance == 0 {
		req.MinRelevance = 0.5
	}

	resp, err := c.doRequest(ctx, "POST", "/query", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			Memory    json.RawMessage `json:"memory"`
			Relevance float64         `json:"relevance"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]MemoryQueryResult, len(data.Results))
	for i, r := range data.Results {
		memory, err := c.parseMemoryBytes(r.Memory)
		if err != nil {
			return nil, err
		}
		results[i] = MemoryQueryResult{
			Memory:    *memory,
			Relevance: r.Relevance,
		}
	}

	return results, nil
}

// Get retrieves a specific memory by ID.
func (c *MemoryClient) Get(ctx context.Context, memoryID string) (*Memory, error) {
	resp, err := c.doRequest(ctx, "GET", "/"+memoryID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseMemory(resp.Body)
}

// Delete deletes a memory.
func (c *MemoryClient) Delete(ctx context.Context, memoryID string) error {
	resp, err := c.doRequest(ctx, "DELETE", "/"+memoryID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// CreateEdge creates an edge between two memories.
func (c *MemoryClient) CreateEdge(ctx context.Context, sourceID, targetID, relationship string, strength float64) (*Edge, error) {
	if strength == 0 {
		strength = 0.5
	}

	body := map[string]interface{}{
		"sourceId":     sourceID,
		"targetId":     targetID,
		"relationship": relationship,
		"strength":     strength,
	}

	resp, err := c.doRequest(ctx, "POST", "/edges", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		SourceID           string `json:"sourceId"`
		TargetID           string `json:"targetId"`
		Relationship       string `json:"relationship"`
		Strength           float64 `json:"strength"`
		CreatedAt          string `json:"createdAt"`
		LastStrengthenedAt string `json:"lastStrengthenedAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, data.CreatedAt)
	lastStrengthened, _ := time.Parse(time.RFC3339, data.LastStrengthenedAt)

	return &Edge{
		SourceID:           data.SourceID,
		TargetID:           data.TargetID,
		Relationship:       data.Relationship,
		Strength:           data.Strength,
		CreatedAt:          createdAt,
		LastStrengthenedAt: lastStrengthened,
	}, nil
}

func (c *MemoryClient) parseMemory(r io.Reader) (*Memory, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return c.parseMemoryBytes(body)
}

func (c *MemoryClient) parseMemoryBytes(data []byte) (*Memory, error) {
	var raw struct {
		ID                 string                 `json:"id"`
		Content            string                 `json:"content"`
		MemoryType         string                 `json:"memoryType"`
		Importance         float64                `json:"importance"`
		Strength           float64                `json:"strength"`
		ConsolidationState string                 `json:"consolidationState"`
		Namespace          string                 `json:"namespace"`
		Tags               []string               `json:"tags"`
		CreatedAt          string                 `json:"createdAt"`
		LastAccessedAt     string                 `json:"lastAccessedAt"`
		AccessCount        int                    `json:"accessCount"`
		Embedding          []float64              `json:"embedding"`
		Metadata           map[string]interface{} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse memory: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
	lastAccessed, _ := time.Parse(time.RFC3339, raw.LastAccessedAt)

	return &Memory{
		ID:                 raw.ID,
		Content:            raw.Content,
		MemoryType:         MemoryType(raw.MemoryType),
		Importance:         raw.Importance,
		Strength:           raw.Strength,
		ConsolidationState: ConsolidationState(raw.ConsolidationState),
		Namespace:          raw.Namespace,
		Tags:               raw.Tags,
		CreatedAt:          createdAt,
		LastAccessedAt:     lastAccessed,
		AccessCount:        raw.AccessCount,
		Embedding:          raw.Embedding,
		Metadata:           raw.Metadata,
	}, nil
}
