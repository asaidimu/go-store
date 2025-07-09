# Task-Based Guide

# Task-Based Guide

This section provides practical guides for common tasks, including indexing, querying, and streaming documents.

## 1. Indexing and Querying

`go-store` allows you to create B-tree based indexes on document fields, enabling fast lookups and range queries. Indexes are updated automatically when documents are inserted, updated, or deleted.

### Creating an Index

Use `CreateIndex` to define an index on one or more fields. Composite indexes (multiple fields) are useful for queries involving combinations of criteria.

```go
package main

import (
	"fmt"
	"log"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Insert some initial documents
	s.Insert(store.Document{"name": "Alice", "age": 30, "city": "New York"})
	s.Insert(store.Document{"name": "Bob", "age": 25, "city": "London"})

	// Wait briefly to ensure async index updates process before creation
	time.Sleep(50 * time.Millisecond)

	// Create a single-field index
	err := s.CreateIndex("by_city", []string{"city"})
	if err != nil {
		log.Fatalf("Failed to create 'by_city' index: %v", err)
	}
	fmt.Println("Created index 'by_city' on field 'city'.")

	// Create a composite index
	err = s.CreateIndex("by_name_and_age", []string{"name", "age"})
	if err != nil {
		log.Fatalf("Failed to create 'by_name_and_age' index: %v", err)
	}
	fmt.Println("Created index 'by_name_and_age' on fields 'name' and 'age'.")

	// Attempt to create an index that already exists
	err = s.CreateIndex("by_city", []string{"city"})
	if err == store.ErrIndexExists {
		fmt.Println("Handled: Index 'by_city' already exists.")
	} else if err != nil {
		log.Fatalf("Expected ErrIndexExists, got %v", err)
	}
}
```

### Performing Exact Lookups

Use the `Lookup` method with an index name and a slice of values corresponding to the indexed fields to find exact matches.

```go
package main

import (
	"fmt"
	"log"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	s.Insert(store.Document{"name": "Alice", "city": "New York"})
	s.Insert(store.Document{"name": "Bob", "city": "London"})
	s.Insert(store.Document{"name": "Charlie", "city": "New York"})
	// Wait for async index updates
	time.Sleep(50 * time.Millisecond)

	_ = s.CreateIndex("by_city", []string{"city"})

	fmt.Println("\nDocuments in 'New York':")
	nyDocs, err := s.Lookup("by_city", []any{"New York"})
	if err != nil {
		log.Fatalf("Lookup failed: %v", err)
	}
	for _, doc := range nyDocs {
		fmt.Printf("  ID: %s, Name: %s, City: %s\n", doc.ID, doc.Data["name"], doc.Data["city"])
	}

	fmt.Println("\nDocuments in 'Paris' (non-existent):")
	parisDocs, err := s.Lookup("by_city", []any{"Paris"})
	if err != nil {
		log.Fatalf("Lookup failed for non-existent city: %v", err)
	}
	if len(parisDocs) == 0 {
		fmt.Println("  No documents found for Paris, as expected.")
	}
}
```

### Performing Range Lookups

For numerical or comparable string fields, `LookupRange` allows you to retrieve documents where the indexed field's value falls within a specified (inclusive) range.

```go
package main

import (
	"fmt"
	"log"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	s.Insert(store.Document{"item": "A", "value": 10})
	s.Insert(store.Document{"item": "B", "value": 25})
	s.Insert(store.Document{"item": "C", "value": 50})
	s.Insert(store.Document{"item": "D", "value": 75})
	// Wait for async index updates
	time.Sleep(50 * time.Millisecond)

	_ = s.CreateIndex("by_value", []string{"value"})

	fmt.Println("\nDocuments with 'value' between 20 and 60 (inclusive):")
	rangeDocs, err := s.LookupRange("by_value", []any{20}, []any{60})
	if err != nil {
		log.Fatalf("LookupRange failed: %v", err)
	}
	for _, doc := range rangeDocs {
		fmt.Printf("  ID: %s, Item: %s, Value: %.0f\n", doc.ID, doc.Data["item"], doc.Data["value"])
	}
}
```

### Dropping an Index

If an index is no longer needed, you can remove it using `DropIndex`.

```go
package main

import (
	"fmt"
	"log"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	_ = s.Insert(store.Document{"foo": "bar"})
	// Wait for async index updates
	time.Sleep(50 * time.Millisecond)

	_ = s.CreateIndex("my_index", []string{"foo"})
	fmt.Println("Index 'my_index' created.")

	err := s.DropIndex("my_index")
	if err != nil {
		log.Fatalf("Failed to drop index 'my_index': %v", err)
	}
	fmt.Println("Index 'my_index' dropped successfully.")

	// Attempt to drop a non-existent index
	err = s.DropIndex("non_existent_index")
	if err == store.ErrIndexNotFound {
		fmt.Println("Handled: Index 'non_existent_index' not found, as expected.")
	} else if err != nil {
		log.Fatalf("Expected ErrIndexNotFound, got %v", err)
	}
}
```

## 2. Streaming Documents

For iterating over all documents in the store, `Stream` provides an efficient, channel-based approach that allows you to process documents as they become available without loading the entire dataset into memory at once.

```go
package main

import (
	"fmt"
	"log"
	"time"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Insert several documents
	for i := 0; i < 5; i++ {
		s.Insert(store.Document{"id": i, "data": fmt.Sprintf("Document %d", i)})
	}
	// Wait for async index updates
	time.Sleep(50 * time.Millisecond)

	fmt.Println("\nStreaming all documents:")
	// Create a stream with a buffer size of 2
	docStream := s.Stream(2)

	for {
		docResult, err := docStream.Next()
		if err != nil {
			if err == store.ErrStreamClosed || err.Error() == "context canceled" {
				fmt.Println("Document stream finished or cancelled.")
				break
			}
			log.Printf("Error reading from stream: %v", err)
			break
		}
		fmt.Printf("  Streamed: ID=%s, Data: %+v\n", docResult.ID, docResult.Data)
	}
	docStream.Close() // Always close the stream when done

	// Demonstrating early stream closure
	fmt.Println("\nStreaming documents with early closure:")
	shortStream := s.Stream(1)
	for i := 0; i < 2; i++ { // Read only two documents
		docResult, err := shortStream.Next()
		if err != nil {
			break
		}
		fmt.Printf("  Reading partial stream: %s\n", docResult.ID)
	}
	shortStream.Close()
	fmt.Println("Stream closed early.")
}
```

**Decision Pattern: Choosing a Stream Buffer Size**

*   **IF** you need to process documents immediately and have low memory constraints per document,
    *   **THEN** use a small buffer size (e.g., `s.Stream(1)` or `s.Stream(0)` for unbuffered).
*   **ELSE IF** you expect to process a large number of documents and can tolerate some latency for batching,
    *   **THEN** use a larger buffer size (e.g., `s.Stream(100)` or more) to potentially improve throughput by reducing channel operations.

---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*