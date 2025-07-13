---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Patterns Reference"
description: "Common patterns, examples, and validation approaches"
---
# Examples

## `Basic Document CRUD`

The fundamental pattern for creating, reading, updating, and deleting documents within the store.



```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Insert
	doc := store.Document{"name": "Example", "value": 10}
	id, err := s.Insert(doc)
	if err != nil { log.Fatalf("Insert failed: %v", err) }
	fmt.Printf("Inserted ID: %s\n", id)

	// Get
	retrieved, err := s.Get(id)
	if err != nil { log.Fatalf("Get failed: %v", err) }
	fmt.Printf("Retrieved: %+v\n", retrieved.Data)

	// Update
	updatedDoc := store.Document{"name": "Updated Example", "value": 20, "status": "done"}
	err = s.Update(id, updatedDoc)
	if err != nil { log.Fatalf("Update failed: %v", err) }
	fmt.Println("Document updated.")

	// Delete
	err = s.Delete(id)
	if err != nil { log.Fatalf("Delete failed: %v", err) }
	fmt.Println("Document deleted.")
}
```


Successful execution logs indicating document creation, retrieval, update, and deletion without errors. Attempting to retrieve after delete results in 'document not found' error.

**Related Methods**: `Insert`, `Get`, `Update`, `Delete`

**Related Errors**: `ErrDocumentNotFound`, `ErrInvalidDocument`

---

## `Indexed Lookup`

How to create and utilize indexes for efficient exact and range-based document retrieval.



```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Insert sample data
	s.Insert(store.Document{"city": "New York", "age": 30})
	s.Insert(store.Document{"city": "London", "age": 25})
	s.Insert(store.Document{"city": "New York", "age": 35})

	// Create an index
	err := s.CreateIndex("by_city_age", []string{"city", "age"})
	if err != nil { log.Fatalf("CreateIndex failed: %v", err) }
	fmt.Println("Index 'by_city_age' created.")

	// Exact Lookup
	ny30Docs, err := s.Lookup("by_city_age", []any{"New York", 30})
	if err != nil { log.Fatalf("Lookup failed: %v", err) }
	fmt.Printf("\nDocs in NY, age 30: %d\n", len(ny30Docs))
	for _, doc := range ny30Docs { fmt.Printf("  %+v\n", doc.Data) }

	// Range Lookup
	ageRangeDocs, err := s.LookupRange("by_city_age", []any{"New York", 20}, []any{"New York", 40})
	if err != nil { log.Fatalf("LookupRange failed: %v", err) }
	fmt.Printf("\nDocs in NY, age 20-40: %d\n", len(ageRangeDocs))
	for _, doc := range ageRangeDocs { fmt.Printf("  %+v\n", doc.Data) }
}
```


The output should correctly list documents matching the exact lookup (`New York`, `30`) and the range lookup (`New York`, `20-40`). The number of results should match the expected count based on inserted data.

**Related Methods**: `CreateIndex`, `Lookup`, `LookupRange`

**Related Errors**: `ErrIndexNotFound`, `ErrIndexExists`

---

## `Streaming Documents`

Iterating over all documents in the store efficiently using `DocumentStream`.



```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Insert some documents
	for i := 0; i < 5; i++ {
		s.Insert(store.Document{"item": fmt.Sprintf("Item%d", i)})
	}

	// Create and consume a stream
	stream := s.Stream(2) // Buffered stream
	defer stream.Close() // Important to close the stream

	count := 0
	for {
		doc, err := stream.Next()
		if err != nil {
			if err == store.ErrStreamClosed { break }
			log.Fatalf("Stream error: %v", err)
		}
		fmt.Printf("Streamed: %s\n", doc.Data["item"])
		count++
	}
	fmt.Printf("Total streamed documents: %d\n", count)
}
```


The program should print 'Streamed: ItemX' for each item inserted, and the final count should match the number of inserted documents. `ErrStreamClosed` should be returned at the end, not other errors.

**Related Methods**: `Stream`, `DocumentStream.Next`, `DocumentStream.Close`

**Related Errors**: `ErrStreamClosed`

---

## `Concurrent Update Pattern`

Demonstrates how `go-store` handles multiple goroutines updating the same document safely using its optimistic concurrency control.



```go
package main

import (
	"fmt"
	"log"
	"sync"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	id, _ := s.Insert(store.Document{"counter": 0})

	var wg sync.WaitGroup
	const numUpdates = 100

	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			// Each goroutine tries to update the counter to its iteration value
			err := s.Update(id, store.Document{"counter": val})
			if err != nil {
				// ErrDocumentNotFound is expected if another goroutine deletes it
				fmt.Printf("Update %d failed: %v\n", val, err)
			}
		}(i + 1)
	}
	wg.Wait()

	finalDoc, err := s.Get(id)
	if err != nil { log.Fatalf("Failed to get final doc: %v", err) }
	fmt.Printf("Final Counter: %v, Final Version: %d\n", finalDoc.Data["counter"], finalDoc.Version)

	// The final counter might be any of the update values, but the version
	// will be numUpdates + 1 (initial insert + all updates).
	if finalDoc.Version != uint64(numUpdates+1) {
		log.Printf("WARNING: Expected version %d, got %d. This indicates a potential test scenario issue, not a bug.", numUpdates+1, finalDoc.Version)
	}
}
```


The `Final Version` printed should be `101` (1 initial insert + 100 updates), demonstrating that all update attempts were processed, even if their `counter` values were overwritten. The `Final Counter` will be the value from the last successful update.

**Related Methods**: `Update`, `Get`

**Related Errors**: `ErrDocumentNotFound`, `ErrDocumentDeleted`

---

