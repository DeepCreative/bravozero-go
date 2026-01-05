# Bravo Zero Go SDK

Official Go SDK for the Bravo Zero Breaking the Limits platform.

## Installation

```bash
go get github.com/bravozero/bravozero-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bravozero/bravozero-go/bravozero"
)

func main() {
	ctx := context.Background()

	// Create client
	client, err := bravozero.NewClient(
		bravozero.WithAPIKey("your-api-key"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check system health
	health, err := client.Governance.GetHealth(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("System: %s, Omega: %.2f\n", health.State, health.OmegaScore)
}
```

## Governance Examples

### Evaluate Actions

```go
// Evaluate an action against the constitution
result, err := client.Governance.Evaluate(ctx, &bravozero.EvaluateRequest{
	AgentID: "agent-123",
	Action:  "Generate a summary of the user document",
	Context: map[string]any{"userId": "user-456"},
})
if err != nil {
	log.Fatal(err)
}

switch result.Decision {
case "allow":
	fmt.Println("Action allowed!")
	performAction()
case "deny":
	fmt.Printf("Denied: %s\n", result.Reasoning)
case "escalate":
	fmt.Println("Requires human review")
	requestApproval(result)
}
```

### Monitor Omega Score

```go
// Get current system alignment
omega, err := client.Governance.GetOmega(ctx)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Omega Score: %.2f\n", omega.Omega)
fmt.Printf("Trend: %s\n", omega.Trend)

for name, score := range omega.Components {
	fmt.Printf("  %s: %.2f\n", name, score)
}
```

### Submit Governance Proposals

```go
// Submit a proposal for new rule
proposal, err := client.Governance.SubmitProposal(ctx, &bravozero.ProposalRequest{
	Title:       "Add data retention rule",
	Description: "Require agents to respect data retention preferences",
	Category:    "rule",
})
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Proposal %s submitted\n", proposal.ProposalID)
fmt.Printf("Voting ends: %s\n", proposal.VotingEndsAt)
```

### Check Active Alerts

```go
// Get system alerts
alerts, err := client.Governance.GetAlerts(ctx)
if err != nil {
	log.Fatal(err)
}

for _, alert := range alerts.Alerts {
	fmt.Printf("[%s] %s\n", alert.Severity, alert.Title)
}
```

## Memory Examples

```go
// Store a memory
memory, err := client.Memory.Record(ctx, bravozero.RecordRequest{
	Content:    "User prefers Go for systems programming",
	MemoryType: bravozero.MemoryTypeSemantic,
	Importance: 0.8,
	Tags:       []string{"preference", "language"},
})
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Stored memory: %s\n", memory.ID)

// Query memories
results, err := client.Memory.Query(ctx, bravozero.QueryRequest{
	Query: "programming preferences",
	Limit: 5,
})
if err != nil {
	log.Fatal(err)
}

for _, match := range results.Matches {
	fmt.Printf("[%.2f] %s\n", match.Relevance, match.Memory.Content)
}
```

## VFS Examples

```go
// List files
files, err := client.Bridge.ListFiles(ctx, "/project/src", false)
if err != nil {
	log.Fatal(err)
}

for _, f := range files {
	fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
}
```

## Configuration

Using environment variables:

```bash
export BRAVOZERO_API_KEY="your-api-key"
```

```go
client, _ := bravozero.NewClient() // Uses env vars
```

## Error Handling

```go
result, err := client.Governance.Evaluate(ctx, req)
if err != nil {
	switch e := err.(type) {
	case *bravozero.RateLimitError:
		fmt.Printf("Rate limited, retry after %ds\n", e.RetryAfter)
	case *bravozero.ConstitutionDeniedError:
		fmt.Printf("Denied: %s\n", e.Reasoning)
	case *bravozero.NotFoundError:
		fmt.Println("Resource not found")
	case *bravozero.ServiceUnavailableError:
		fmt.Println("Governance unavailable, using fallback")
	default:
		log.Fatal(err)
	}
}
```

## Documentation

- [Quickstart Guide](https://docs.bravozero.ai/getting-started)
- [Governance Integration](https://docs.bravozero.ai/guides/governance-integration)
- [API Reference](https://docs.bravozero.ai/api/governance-api)

## License

MIT

