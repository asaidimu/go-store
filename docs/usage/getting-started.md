# Getting Started

# Getting Started

This section guides you through setting up `go-store` and understanding its core concepts.

## Overview & Core Concepts

`go-store` operates as an in-memory document store, meaning all data resides in your application's RAM. It's built for speed and concurrent access, offering a schemaless approach where documents are flexible key-value maps. The primary interaction is through a `Store` instance, which manages documents and their associated indexes.

Each document is internally versioned, facilitating optimistic concurrency control. This means multiple goroutines can safely read and write to documents, and the system ensures consistency by managing document snapshots.

## Quick Setup Guide

### Prerequisites

Before you begin, ensure you have Go installed on your system. `go-store` is developed and tested with Go version 1.24.4 or higher.

### Installation

To add `go-store` to your Go project, use the `go get` command:

```bash
go get github.com/asaidimu/go-store
```

After installation, you can import the library into your Go source files:

```go
import store "github.com/asaidimu/go-store"
```

## First Tasks with Decision Patterns

### Initializing the Store

To begin using `go-store`, you first need to create a new `Store` instance. It's crucial to defer the `Close()` method to ensure proper resource cleanup when your application exits.

```go
package main

import (
	"fmt"
	store "github.com/asaidimu/go-store"
)

func main() {
	// Create a new in-memory store
	s := store.NewStore()
	// Ensure the store is closed when the main function exits
	defer s.Close()

	fmt.Println("Store initialized and ready.")

	// Your application logic here
}
```

**Decision Pattern: When to Initialize and Close the Store?**

*   **IF** you need a global, application-wide document store that lives for the duration of your application's runtime,
    *   **THEN** initialize `store.NewStore()` once at application startup (e.g., in `main` or `init` function) and ensure `defer s.Close()` is called to release resources cleanly on exit.
*   **ELSE IF** you need a temporary, isolated store for a specific task or scope,
    *   **THEN** create and close the store within that function or scope, remembering to call `defer s.Close()` immediately after `NewStore()`.

---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*