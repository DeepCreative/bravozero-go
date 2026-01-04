package bravozero

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// PersonaAuthenticator handles PERSONA attestation signing.
type PersonaAuthenticator struct {
	agentID    string
	privateKey ed25519.PrivateKey
}

// NewPersonaAuthenticator creates a new authenticator from a private key file.
func NewPersonaAuthenticator(agentID, privateKeyPath string) (*PersonaAuthenticator, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := parsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &PersonaAuthenticator{
		agentID:    agentID,
		privateKey: privateKey,
	}, nil
}

func parsePrivateKey(pemData []byte) (ed25519.PrivateKey, error) {
	pemStr := string(pemData)

	// Extract base64 content
	start := strings.Index(pemStr, "-----BEGIN PRIVATE KEY-----")
	end := strings.Index(pemStr, "-----END PRIVATE KEY-----")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid PEM format")
	}

	base64Content := pemStr[start+len("-----BEGIN PRIVATE KEY-----") : end]
	base64Content = strings.ReplaceAll(base64Content, "\n", "")
	base64Content = strings.ReplaceAll(base64Content, "\r", "")
	base64Content = strings.TrimSpace(base64Content)

	derBytes, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Ed25519 private key is the last 32 bytes of DER encoding
	if len(derBytes) < 32 {
		return nil, fmt.Errorf("private key data too short")
	}

	seed := derBytes[len(derBytes)-32:]
	return ed25519.NewKeyFromSeed(seed), nil
}

// CreateAttestation creates a signed PERSONA attestation.
func (a *PersonaAuthenticator) CreateAttestation(action string) (string, error) {
	timestamp := time.Now().Unix()
	nonce := fmt.Sprintf("%d-%d", timestamp, time.Now().UnixNano())

	payload := map[string]interface{}{
		"agent_id":  a.agentID,
		"timestamp": timestamp,
		"nonce":     nonce,
	}

	if action != "" {
		payload["action"] = action
	}

	// Sort keys for consistent serialization
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sortedPayload := make(map[string]interface{})
	for _, k := range keys {
		sortedPayload[k] = payload[k]
	}

	payloadBytes, err := json.Marshal(sortedPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Sign with Ed25519
	signature := ed25519.Sign(a.privateKey, payloadBytes)

	// Build attestation
	attestation := map[string]string{
		"payload":   base64.StdEncoding.EncodeToString(payloadBytes),
		"signature": base64.StdEncoding.EncodeToString(signature),
		"algorithm": "Ed25519",
	}

	attestationBytes, err := json.Marshal(attestation)
	if err != nil {
		return "", fmt.Errorf("failed to marshal attestation: %w", err)
	}

	return base64.StdEncoding.EncodeToString(attestationBytes), nil
}

// GetPublicKey returns the public key as base64.
func (a *PersonaAuthenticator) GetPublicKey() string {
	publicKey := a.privateKey.Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(publicKey)
}

