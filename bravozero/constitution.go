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

// Decision represents a constitution evaluation decision.
type Decision string

const (
	DecisionPermit   Decision = "permit"
	DecisionDeny     Decision = "deny"
	DecisionEscalate Decision = "escalate"
)

// AppliedRule represents a rule applied during evaluation.
type AppliedRule struct {
	RuleID       string  `json:"ruleId"`
	Name         string  `json:"name"`
	Matched      bool    `json:"matched"`
	Contribution float64 `json:"contribution"`
}

// EvaluationResult represents the result of a constitution evaluation.
type EvaluationResult struct {
	RequestID      string        `json:"requestId"`
	Decision       Decision      `json:"decision"`
	Confidence     float64       `json:"confidence"`
	AlignmentScore float64       `json:"alignmentScore"`
	AppliedRules   []AppliedRule `json:"appliedRules"`
	Reasoning      string        `json:"reasoning"`
	EvaluatedAt    time.Time     `json:"evaluatedAt"`
}

// OmegaScore represents the global alignment score.
type OmegaScore struct {
	Omega      float64            `json:"omega"`
	Components map[string]float64 `json:"components"`
	Trend      string             `json:"trend"`
	Timestamp  time.Time          `json:"timestamp"`
}

// EvaluateRequest represents a request to evaluate an action.
type EvaluateRequest struct {
	Action   string                 `json:"action"`
	Context  map[string]interface{} `json:"context"`
	Priority string                 `json:"priority"`
}

// Rule represents a constitution rule.
type Rule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Condition   string `json:"condition"`
	Action      string `json:"action"`
	Active      bool   `json:"active"`
}

// ConstitutionClient provides access to the Constitution Agent API.
type ConstitutionClient struct {
	baseURL       string
	apiKey        string
	agentID       string
	authenticator *PersonaAuthenticator
	httpClient    *http.Client
}

// NewConstitutionClient creates a new Constitution Agent client.
func NewConstitutionClient(
	baseURL, apiKey, agentID string,
	auth *PersonaAuthenticator,
	timeoutSeconds int,
) *ConstitutionClient {
	return &ConstitutionClient{
		baseURL:       baseURL + "/v1/constitution",
		apiKey:        apiKey,
		agentID:       agentID,
		authenticator: auth,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

func (c *ConstitutionClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
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

// Evaluate evaluates an action against the constitution.
func (c *ConstitutionClient) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluationResult, error) {
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}

	body := map[string]interface{}{
		"agentId":  c.agentID,
		"action":   req.Action,
		"context":  req.Context,
		"priority": req.Priority,
	}

	resp, err := c.doRequest(ctx, "POST", "/evaluate", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		RequestID      string        `json:"requestId"`
		Decision       string        `json:"decision"`
		Confidence     float64       `json:"confidence"`
		AlignmentScore float64       `json:"alignmentScore"`
		AppliedRules   []AppliedRule `json:"appliedRules"`
		Reasoning      string        `json:"reasoning"`
		EvaluatedAt    string        `json:"evaluatedAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	evaluatedAt, _ := time.Parse(time.RFC3339, data.EvaluatedAt)

	result := &EvaluationResult{
		RequestID:      data.RequestID,
		Decision:       Decision(data.Decision),
		Confidence:     data.Confidence,
		AlignmentScore: data.AlignmentScore,
		AppliedRules:   data.AppliedRules,
		Reasoning:      data.Reasoning,
		EvaluatedAt:    evaluatedAt,
	}

	if result.Decision == DecisionDeny {
		return result, &ConstitutionDeniedError{
			Reasoning: result.Reasoning,
			Result:    result,
		}
	}

	return result, nil
}

// GetOmega retrieves the current global Omega alignment score.
func (c *ConstitutionClient) GetOmega(ctx context.Context) (*OmegaScore, error) {
	resp, err := c.doRequest(ctx, "GET", "/omega", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Omega      float64            `json:"omega"`
		Components map[string]float64 `json:"components"`
		Trend      string             `json:"trend"`
		Timestamp  string             `json:"timestamp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	timestamp, _ := time.Parse(time.RFC3339, data.Timestamp)

	return &OmegaScore{
		Omega:      data.Omega,
		Components: data.Components,
		Trend:      data.Trend,
		Timestamp:  timestamp,
	}, nil
}

// ListRules retrieves all constitution rules.
func (c *ConstitutionClient) ListRules(ctx context.Context, category, priority string) ([]Rule, error) {
	path := "/rules"
	if category != "" || priority != "" {
		path += "?"
		if category != "" {
			path += "category=" + category
		}
		if priority != "" {
			if category != "" {
				path += "&"
			}
			path += "priority=" + priority
		}
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rules []Rule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rules, nil
}

// GetRule retrieves a specific rule by ID.
func (c *ConstitutionClient) GetRule(ctx context.Context, ruleID string) (*Rule, error) {
	resp, err := c.doRequest(ctx, "GET", "/rules/"+ruleID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rule Rule
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &rule, nil
}
