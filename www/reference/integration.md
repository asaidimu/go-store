---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Integration Reference"
description: "Documentation on integration aspects, requirements, and patterns"
---
# Integration

## Environment Requirements

Go Runtime Environment (Go 1.24.4+ recommended). No special compiler settings or platform constraints beyond standard Go compilation.

## Initialization Patterns

### The most common way to initialize the store and ensure proper resource cleanup upon application shutdown.



```go
package main

import (
	"fmt"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	// Initialize a new store instance
	s := store.NewStore()
	// Defer the Close method call to ensure resources are released
	// when the main function exits.
	defer s.Close()

	fmt.Println("Store initialized and ready for use.")

	// ... perform store operations here ...
}
```


## Common Pitfalls

### Forgetting to call `store.Close()`

**Solution**: Always use `defer s.Close()` immediately after `s := store.NewStore()` to ensure resources are released and to aid garbage collection, especially in long-running applications.

### Modifying `DocumentResult.Data` directly affects the stored document.

**Solution**: `DocumentResult.Data` is a deep copy. Modifying it will NOT affect the internal state of the document in the store. To update a document, you must call `s.Update(id, newDocumentData)`.

### Expecting data persistence across application restarts.

**Solution**: `go-store` is an in-memory database. All data is lost when the application exits. For persistence, you need to implement external serialization/deserialization or use a persistent storage layer.

## Lifecycle Dependencies

The `Store` instance should be initialized once at the application's startup phase (e.g., in `main` or a dedicated initialization function). Its `Close()` method should be called during the application's graceful shutdown procedure to release all allocated memory and resources. Operations performed on a `closed` store will result in `ErrStoreClosed`.

