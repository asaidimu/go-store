# Advanced Usage

# Advanced Usage

This section delves into more complex scenarios, focusing on concurrent operations and how `go-store` handles them.

## Concurrent Operations

`go-store` is designed to be highly concurrent, allowing multiple goroutines to safely interact with the store simultaneously. It uses a combination of fine-grained locking and optimistic concurrency control to ensure data integrity and consistency.

### Concurrent Updates to a Single Document

When multiple goroutines attempt to update the same document, `go-store`'s versioning mechanism ensures that updates are applied sequentially. Each successful update increments the document's version.

```go
package main

import (
	"fmt"
	"log"
	"sync"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	fmt.Println("--- Demonstrating Concurrent Updates ---")
	s := store.NewStore()
	defer s.Close()

	// Insert an initial document
	docID, _ := s.Insert(store.Document{"counter": 0})
	fmt.Printf("Initial document ID: %s, counter: %d\n", docID, (func() float64 { d, _ := s.Get(docID); return d.Data["counter"].(float64) })())

	var wg sync.WaitGroup
	const numUpdates = 100

	fmt.Printf("Performing %d concurrent updates on document %s...\n", numUpdates, docID)
	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			// Get the current document, increment counter, then update
			// This simulates a read-modify-write pattern
			doc, err := s.Get(docID)
			if err != nil {
				// Document might be deleted by another goroutine if this were a mixed workload
				// For pure updates, this indicates an issue.
				log.Printf("Error getting document %s in iteration %d: %v\n", docID, iteration, err)
				return
			}

			currentCounter := int(doc.Data["counter"].(float64))
			newCounter := currentCounter + 1
			// Attempt to update. If another goroutine updated between Get and Update,
			// this specific Update might fail if a more robust OCC (e.g., version check in Update itself)
			// were implemented. go-store's current Update replaces the whole snapshot, so last one wins.
			err = s.Update(docID, store.Document{"counter": newCounter})
			if err != nil {
				log.Printf("Concurrent update %d failed for doc %s: %v\n", iteration, docID, err)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("Concurrent updates finished.")

	// Verify the final state and version
	finalDoc, err := s.Get(docID)
	if err != nil {
		log.Fatalf("Failed to get final document after concurrent updates: %v", err)
	}
	fmt.Printf("Final document state: ID=%s, Counter=%.0f, Version=%d\n",
		finalDoc.ID, finalDoc.Data["counter"], finalDoc.Version)

	// Due to the simplistic 'last write wins' nature of `Update` on a single field
	// without an explicit CAS (Compare-And-Swap) mechanism, the final counter value
	// might not strictly be `numUpdates`. The version will reflect total updates.
	// The `store_test.go` shows more robust concurrent testing with multiple documents.
	fmt.Printf("Note: The 'counter' value reflects the outcome of concurrent updates where the last successful write determines the final state. The 'Version' field is a more reliable indicator of total updates processed.\n")

	// For strict sequential consistency of a single counter, a higher-level CAS would be needed,
	// or ensuring only one goroutine increments the counter while others read.

	// Expected final version should be numUpdates + 1 (initial insert + numUpdates)
	if finalDoc.Version != uint64(numUpdates+1) {
		fmt.Printf("Warning: Expected final version %d, got %d.\n", numUpdates+1, finalDoc.Version)
	}
}
```

## Customization & Optimization

`go-store`'s in-memory nature means performance is largely governed by CPU speed and available RAM. Here are some points:

*   **Document Structure**: Keep documents relatively flat. Deeply nested structures will incur higher costs during `copyDocument` operations.
*   **Index Selection**: Create indexes strategically. Too many indexes can slow down writes (Insert, Update, Delete) as each index needs to be maintained. Index only fields you frequently query.
*   **Stream Buffer Size**: When using `Stream`, adjust the `bufferSize` parameter in `NewDocumentStream`. A larger buffer can lead to higher throughput by batching, while a smaller or zero buffer reduces memory footprint and provides documents more immediately.

### Future Enhancements (Roadmap)

The `advanced/main.go` example briefly touches upon a *simulated* functional index. This highlights a potential future feature where users could define custom functions to derive index keys, enabling more complex query capabilities. Currently, indexes are strictly field-based.

**Example of a 'simulated' functional query (manual filtering):**

```go
// Part of examples/advanced/main.go
// ...
	fmt.Println("\nAdvanced: Demonstrating (simulated) Functional Index usage...")
	s.Insert(store.Document{"product": "Laptop", "price": 1200, "in_stock": true})
	s.Insert(store.Document{"product": "Mouse", "price": 25, "in_stock": true})
	s.Insert(store.Document{"product": "Monitor", "price": 300, "in_stock": false})
	s.Insert(store.Document{"product": "Keyboard", "price": 75, "in_stock": true})

	time.Sleep(100 * time.Millisecond) // Allow updates to process

	fmt.Println("Advanced: (Simulated) Expensive In-Stock Items (price > 100 and in_stock == true):")
	streamAll := s.Stream(10)
	filteredCount := 0
	for {
		docRes, err := streamAll.Next()
		if err != nil {
			break // Stream closed or error
		}
		// Apply the filter logic manually
		price, priceOK := docRes.Data["price"].(int)
		inStock, stockOK := docRes.Data["in_stock"].(bool)
		if priceOK && stockOK && price > 100 && inStock {
			fmt.Printf("  Product: %s, Price: %d\n", docRes.Data["product"], price)
			filteredCount++
		}
	}
	streamAll.Close()
	if filteredCount == 0 {
		fmt.Println("  No expensive in-stock items found (simulated, or if functional index not implemented).")
	}
// ...
```

This manual filtering demonstrates the *intent* of a functional index, which would automate this filtering process at the index level for performance gains.

---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*