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
		bravozero.WithAgentID("your-agent-id"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Evaluate an action
	result, err := client.Constitution.Evaluate(ctx, bravozero.EvaluationRequest{
		Action: "read_file",
		Context: map[string]any{
			"path": "/project/src/main.go",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if result.Decision == bravozero.DecisionPermit {
		fmt.Println("Action allowed!")
	} else {
		fmt.Printf("Denied: %s\n", result.Reasoning)
	}

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

	// List files
	files, err := client.Bridge.ListFiles(ctx, "/project/src", false)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
	}
}
```

## Configuration

Using environment variables:

```bash
export BRAVOZERO_API_KEY="your-api-key"
export BRAVOZERO_AGENT_ID="your-agent-id"
```

```go
client, _ := bravozero.NewClient() // Uses env vars
```

## Error Handling

```go
result, err := client.Constitution.Evaluate(ctx, req)
if err != nil {
	switch e := err.(type) {
	case *bravozero.RateLimitError:
		fmt.Printf("Rate limited, retry after %ds\n", e.RetryAfter)
	case *bravozero.ConstitutionDeniedError:
		fmt.Printf("Denied: %s\n", e.Reasoning)
	case *bravozero.NotFoundError:
		fmt.Println("Resource not found")
	default:
		log.Fatal(err)
	}
}
```

## Documentation

- [Quickstart Guide](https://docs.bravozero.ai/getting-started)
- [API Reference](https://docs.bravozero.ai/api)

## License

MIT

