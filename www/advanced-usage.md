---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Advanced Usage"
description: "Advanced Usage documentation and guidance"
---
# Advanced Usage

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Advanced Usage"
description: "Advanced Usage documentation and guidance"
---
# Advanced Usage

## Advanced Usage

This section delves into more complex scenarios, including concurrent operations and internal design considerations that impact how you use `go-store`.

### Concurrent Operations

`go-store` is designed to be highly concurrent and thread-safe. Multiple goroutines can safely interact with the store simultaneously. This is achieved through a combination of `sync.RWMutex` for overall store management, `atomic.Pointer` for atomic document state transitions, and reference counting for memory safety.

#### Concurrent Updates to a Single Document

When multiple goroutines attempt to update the same document, `go-store` ensures atomicity and consistency through its optimistic concurrency control. Each successful update increments the document's version.

```go
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	id, _ := s.Insert(store.Document{"name": "Item", "count": 0})
	fmt.Printf("Initial document ID: %s\n", id)

	var wg sync.WaitGroup
	const numConcurrentUpdates = 100

	fmt.Printf("\nPerforming %d concurrent increments on document %s...\n", numConcurrentUpdates, id)
	for i := 0; i < numConcurrentUpdates; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			// In a real application, you might read-modify-write, but here we just update
			// with a new value for simplicity of demonstrating concurrency safety.
			// The internal versioning ensures consistency.
			err := s.Update(id, store.Document{"name": "Item", "count": iteration + 1})
			if err != nil {
				// ErrDocumentNotFound can occur if another goroutine deletes it first.
				fmt.Printf("Concurrent update %d failed for %s: %v\n", iteration+1, id, err)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("Concurrent updates finished.")

	// Verify the final state
	finalDoc, err := s.Get(id)
	if err != nil {
		log.Fatalf("Failed to get final document after concurrent updates: %v", err)
	}
	fmt.Printf("Final document state: ID=%s, Name='%s', Count=%.0f, Version=%d\n",
		finalDoc.ID, finalDoc.Data["name"], finalDoc.Data["count"], finalDoc.Version)

	// Note: The 'count' might not be 'numConcurrentUpdates' if the goroutines 
	// overwrite each other's values, but the 'Version' will accurately reflect 
	// the total number of successful updates (+1 for initial insert).
	if finalDoc.Version != uint64(numConcurrentUpdates+1) {
		fmt.Printf("Warning: Expected version %d, got %d. (Async updates can cause non-sequential versions from a single goroutine's perspective, but total versions should be correct).\n", numConcurrentUpdates+1, finalDoc.Version)
	}
}
```

#### Concurrent Inserts and Deletes

Multiple goroutines can insert and delete documents without interference. The store's internal locking ensures that operations on the document map and indexes are safe.

### Understanding Concurrency Control and Memory Management

`go-store`'s robustness stems from its internal mechanisms:

*   **`DocumentSnapshot`**: When a document is updated, a new `DocumentSnapshot` is created. This snapshot contains the new data and an incremented version. The old snapshot is not immediately garbage collected.
*   **Reference Counting (`refCount`)**: Each `DocumentSnapshot` has a reference count. When a `DocumentHandle` (the main pointer to a document) or a query (`Get`, `Stream`, `Lookup`) reads a snapshot, its `refCount` is incremented. When the `DocumentHandle` points to a new snapshot, or a query finishes using it, the `refCount` is decremented. A snapshot's data is cleared only when its `refCount` drops to zero, allowing safe memory management in highly concurrent environments.
*   **`atomic.Pointer`**: Used within `DocumentHandle` to atomically swap pointers to `DocumentSnapshot`s, ensuring that reads always get a consistent, complete snapshot of the document without needing to lock for every read operation.
*   **`sync.RWMutex`**: Used at higher levels (e.g., `Store`'s `documents` and `indexes` maps, `fieldIndex`'s B-tree) to protect structural changes (like adding/removing documents or indexes) and ensure read-write consistency.

### Importance of `Close()` Method

It is critical to call `store.Close()` on the `Store` instance when it's no longer needed, typically when your application is shutting down. This method performs several cleanup tasks:

*   Marks the store as closed, preventing any further operations and returning `ErrStoreClosed`.
*   Clears internal maps (`documents` and `indexes`). This assists the Go garbage collector in reclaiming memory more quickly, especially for large datasets.
*   Ensures all internal `DocumentSnapshot`s eventually have their reference counts drop to zero, allowing their associated memory to be reclaimed.

**Anti-Pattern**: Forgetting to call `Close()` can lead to memory leaks in long-running applications, as internal maps and document references might persist longer than necessary, delaying garbage collection.

### Data Copying Behavior

`go-store` employs deep copying for documents when they are inserted into or retrieved from the store.

*   **`Insert` and `Update`**: When you pass a `store.Document` to `Insert` or `Update`, a deep copy of that `map[string]any` (including nested maps and slices) is made. This ensures that modifications to your original `Document` variable outside the store do not inadvertently affect the store's internal data, and vice-versa.
*   **`Get` and `Stream`**: The `DocumentResult.Data` field returned by `Get` and `DocumentStream.Next()` is also a deep copy. This means you are free to modify the returned `DocumentResult.Data` map and its nested contents without impacting the actual data stored in the `go-store` instance. This guarantees snapshot isolation for read operations.

```go
package main

import (
	"fmt"
	"reflect"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	nestedDoc := store.Document{"value": 1, "list": []string{"a", "b"}}
	originalDoc := store.Document{
		"id":      "my_doc",
		"data":    nestedDoc,
		"tags":    []any{"tag1", "tag2"},
	}

	id, _ := s.Insert(originalDoc);

	// Retrieve the document
	retrievedDoc, _ := s.Get(id)

	// Modify the retrieved deep copy
	retrievedDoc.Data["data"].(store.Document)["value"] = 99
	retrievedDoc.Data["tags"].([]any)[0] = "new_tag1"
	fmt.Printf("Modified retrieved copy: %v\n", retrievedDoc.Data)

	// Get the document again to verify original state in store
	originalStateDoc, _ := s.Get(id)
	fmt.Printf("Original state in store: %v\n", originalStateDoc.Data)

	if originalStateDoc.Data["data"].(store.Document)["value"] == 1 && 
	   originalStateDoc.Data["tags"].([]any)[0] == "a" {
		fmt.Println("\nVerification successful: Original data in store was not affected.")
	} else {
		fmt.Println("\nVerification FAILED: Original data in store was unexpectedly affected.")
	}

	// Demonstrate with initial doc
	originalDoc["data"].(store.Document)["value"] = 500 // Modify original input
	fmt.Printf("Original input after modification: %v\n", originalDoc)

	originalStateDoc2, _ := s.Get(id)
	if originalStateDoc2.Data["data"].(store.Document)["value"] == 1 {
		fmt.Println("Original input modification did NOT affect stored data.")
	} else {
		fmt.Println("Original input modification DID affect stored data.")
	}
}


