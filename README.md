# go-store

[![Go Reference](https://pkg.go.dev/badge/github.com/asaidimu/go-store/v3.svg)](https://pkg.go.dev/github.com/asaidimu/go-store/v3)
[![Build Status](https://github.com/asaidimu/go-store/v3/workflows/Test%20Workflow/badge.svg)](https://github.com/asaidimu/go-store/v3/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An efficient, in-memory, concurrent-safe document store with built-in indexing capabilities.

**Note:** This project is currently in beta. While designed for concurrency and robustness, use conservatively and consider its in-memory nature for production environments.

## Table of Contents

- [Overview & Features](#overview--features)
- [Installation & Setup](#installation--setup)
- [Usage Documentation](#usage-documentation)
  - [Basic Operations](#basic-operations)
  - [Indexing and Querying](#indexing-and-querying)
  - [Streaming Documents](#streaming-documents)
  - [Concurrent Operations](#concurrent-operations)
- [Project Architecture](#project-architecture)
- [Development & Contributing](#development--contributing)
- [Additional Information](#additional-information)

---

## Overview & Features

`go-store` is a lightweight, high-performance in-memory document database designed for applications requiring fast, concurrent access to schemaless data. It provides core CRUD (Create, Read, Update, Delete) operations and advanced features like field-based indexing, range queries, and document streaming.

The store leverages Go's concurrency primitives (`sync.RWMutex`, `atomic.Pointer`) and `github.com/google/btree` for efficient, thread-safe operations. Documents are stored as flexible `map[string]any` types, and changes are managed via an optimistic concurrency control mechanism using versioned `DocumentSnapshot`s and reference counting, ensuring data consistency even under heavy concurrent loads.

Key considerations for `go-store` are its in-memory nature (data is not persisted to disk and will be lost upon application shutdown) and its focus on efficient concurrent access patterns, making it suitable for caching layers, temporary data storage, or embedded application-level databases where persistence is handled externally or not required.

### Key Features

*   💾 **In-Memory Storage**: All data resides in RAM for maximum speed.
*   ⚙️ **Concurrency Safe**: Designed from the ground up for safe concurrent access by multiple goroutines using fine-grained locking and atomic operations.
*   🚀 **Optimistic Concurrency Control**: Documents are versioned, allowing for robust updates and reads without explicit transaction management for single document operations.
*   🔍 **Flexible Document Schema**: Documents are simple `map[string]any`, allowing for dynamic, schemaless data structures.
*   ⚡ **Field-Based Indexing**: Create B-tree based indexes on single or composite document fields for efficient lookups.
*   🎯 **Exact Lookups**: Quickly retrieve documents matching exact values on indexed fields.
*   ↔️ **Range Queries**: Query documents within a specified range on indexed fields.
*   🌊 **Document Streaming**: Iterate efficiently over all documents in the store, providing a consistent snapshot.
*   🗑️ **Graceful Shutdown**: Proper resource release upon store closure.
*   🚫 **Custom Error Handling**: Specific error types for common scenarios like `ErrDocumentNotFound` or `ErrIndexExists`.

---

## Installation & Setup

### Prerequisites

*   [Go](https://go.dev/dl/) (version 1.24.4 or higher recommended, as specified in `go.mod`)

### Installation Steps

To integrate `go-store` into your Go project, use `go get`:

```bash
go get github.com/asaidimu/go-store/v3
```

Then, import it into your Go source files:

```go
import store "github.com/asaidimu/go-store/v3"
```

### Verification

You can verify the installation by running the provided examples:

```bash
git clone https://github.com/asaidimu/go-store/v3.git
cd go-store
go run examples/basic/main.go
go run examples/intermediate/main.go
go run examples/advanced/main.go
```

---

## Usage Documentation

This section provides practical examples demonstrating the various functionalities of `go-store`.

### Basic Operations

The `store` package provides fundamental CRUD operations for documents.

```go
// examples/basic/main.go
package main

import (
	"fmt"
	"log"

	store "github.com/asaidimu/go-store/v3"
)

func main() {
	fmt.Println("--- Basic Store Usage Example ---")
	s := store.NewStore()
	defer s.Close() // Ensure the store is closed and resources are released

	// 1. Insert Document
	doc1 := store.Document{"title": "My First Document", "author": "Alice"}
	id1, err := s.Insert(doc1)
	if err != nil {
		log.Fatalf("Basic: Failed to insert document: %v", err)
	}
	fmt.Printf("Basic: Inserted document with ID: %s\n", id1)

	// 2. Get Document
	retrievedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to get document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Retrieved document: ID=%s, Title='%s', Version=%d\n",
		retrievedDoc.ID, retrievedDoc.Data["title"], retrievedDoc.Version)

	// 3. Update Document
	// Note: Existing fields are replaced, new fields are added.
	updatedDoc1 := store.Document{"title": "My First Document (Revised)", "author": "Alice Smith", "pages": 150}
	err = s.Update(id1, updatedDoc1)
	if err != nil {
		log.Fatalf("Basic: Failed to update document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Updated document with ID: %s\n", id1)

	retrievedUpdatedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to get updated document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Retrieved updated document: ID=%s, Title='%s', Pages=%.0f, Version=%d\n",
		retrievedUpdatedDoc.ID, retrievedUpdatedDoc.Data["title"], retrievedUpdatedDoc.Data["pages"], retrievedUpdatedDoc.Version)

	// 4. Delete Document
	err = s.Delete(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to delete document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Deleted document with ID: %s\n", id1)

	// 5. Try to Get Deleted Document (should fail)
	_, err = s.Get(id1)
	if err != nil {
		fmt.Printf("Basic: Attempted to get deleted document %s: %v (Expected error)\n", id1, err)
	}
}
```

### Indexing and Querying

`go-store` allows you to create B-tree based indexes on document fields for efficient data retrieval.

```go
// examples/intermediate/main.go
package main

import (
	"fmt"
	"log"
	"time"

	store "github.com/asaidimu/go-store/v3"
)

func main() {
	fmt.Println("--- Intermediate Store Usage Example ---")
	s := store.NewStore()
	defer s.Close()

	// Insert several documents
	s.Insert(store.Document{"name": "Alice", "age": 30, "city": "New York"})
	s.Insert(store.Document{"name": "Bob", "age": 25, "city": "London"})
	s.Insert(store.Document{"name": "Charlie", "age": 35, "city": "New York"})
	s.Insert(store.Document{"name": "David", "age": 28, "city": "Paris"})
	s.Insert(store.Document{"name": "Eve", "age": 30, "city": "New York"})

	time.Sleep(100 * time.Millisecond) // Give async index updates time to process

	// 1. Create an Index
	// Indexes can be created on single or multiple fields (composite indexes).
	err := s.CreateIndex("by_city", []string{"city"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to create 'by_city' index: %v", err)
	}
	fmt.Println("Intermediate: Created index 'by_city' on field 'city'.")

	err = s.CreateIndex("by_age", []string{"age"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to create 'by_age' index: %v", err)
	}
	fmt.Println("Intermediate: Created index 'by_age' on field 'age'.")

	// 2. Lookup Documents by Exact Match (using "by_city" index)
	fmt.Println("\nIntermediate: Looking up documents in 'New York':")
	nyDocs, err := s.Lookup("by_city", []any{"New York"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to lookup by city: %v", err)
	}
	for _, doc := range nyDocs {
		fmt.Printf("  ID: %s, Name: %s, Age: %.0f\n", doc.ID, doc.Data["name"], doc.Data["age"])
	}

	// 3. Lookup Documents by Range (using "by_age" index)
	// Range queries are inclusive for both min and max values.
	fmt.Println("\nIntermediate: Looking up documents with age between 27 and 32:")
	ageRangeDocs, err := s.LookupRange("by_age", []any{27}, []any{32})
	if err != nil {
		log.Fatalf("Intermediate: Failed to lookup age range: %v", err)
	}
	for _, doc := range ageRangeDocs {
		fmt.Printf("  ID: %s, Name: %s, Age: %.0f\n", doc.ID, doc.Data["name"], doc.Data["age"])
	}

	// 4. Stream All Documents (shown in next section for clarity)
}
```

### Streaming Documents

The `Stream` method allows iterating over all documents in the store efficiently, providing a consistent snapshot.

```go
// Part of examples/intermediate/main.go (continuing from previous section)
// ...
	// 4. Stream All Documents
	fmt.Println("\nIntermediate: Streaming all documents:")
	docStream := s.Stream(2) // Small buffer for demonstration, use a larger buffer for performance
	for {
		docResult, err := docStream.Next()
		if err != nil {
			if err.Error() == "stream closed" { // ErrStreamClosed
				fmt.Println("Intermediate: Document stream finished or cancelled.")
				break
			}
			log.Printf("Intermediate: Error reading from stream: %v", err)
			break
		}
		fmt.Printf("  Streamed: ID=%s, Name=%s, City=%s\n",
			docResult.ID, docResult.Data["name"], docResult.Data["city"])
	}
	docStream.Close() // Important to close the stream when done
// ...
```

### Concurrent Operations

`go-store` is built for concurrency. This example demonstrates concurrent updates and simulated functional indexing.

```go
// examples/advanced/main.go
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	store "github.com/asaidimu/go-store/v3"
)

func main() {
	fmt.Println("--- Advanced Store Usage Example ---")
	s := store.NewStore()
	defer s.Close()

	// Insert initial document
	id1, _ := s.Insert(store.Document{"name": "Original", "value": 100})
	fmt.Printf("Advanced: Initial document ID: %s\n", id1)

	var wg sync.WaitGroup
	const numConcurrentUpdates = 5

	// 1. Concurrent Updates to a single document
	fmt.Printf("\nAdvanced: Performing %d concurrent updates on document %s...\n", numConcurrentUpdates, id1)
	for i := 0; i < numConcurrentUpdates; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			newVal := 100 + iteration + 1
			err := s.Update(id1, store.Document{"name": fmt.Sprintf("Updated %d", iteration+1), "value": newVal})
			if err != nil {
				fmt.Printf("Advanced: Concurrent update %d failed: %v\n", iteration+1, err)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("Advanced: Concurrent updates finished.")

	// Verify the final state and version
	finalDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Advanced: Failed to get final document after concurrent updates: %v", err)
	}
	fmt.Printf("Advanced: Final document state: ID=%s, Name='%s', Value=%.0f, Version=%d\n",
		finalDoc.ID, finalDoc.Data["name"], finalDoc.Data["value"], finalDoc.Version)
	if finalDoc.Version != uint64(numConcurrentUpdates+1) {
		fmt.Printf("Advanced: Warning: Expected version %d, got %d. (Async updates can cause non-sequential versions from a single goroutine's perspective, but total versions should be correct).\n", numConcurrentUpdates+1, finalDoc.Version)
	}

	// 2. Using a theorized Functional Index (simulated)
	// The `CreateFunctionalIndex` and `Filter` methods are not yet implemented in the core store.
	// This example simulates the behavior by manually iterating through documents.
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

	// 3. Dropping an Index
	// This example attempts to drop an index that might have been created
	// in the intermediate example.
	fmt.Println("\nAdvanced: Attempting to drop 'by_city' index (created in intermediate example).")
	err = s.DropIndex("by_city")
	if err != nil {
		fmt.Printf("Advanced: Failed to drop index 'by_city' (might not exist, run intermediate_example.go first): %v\n", err)
	} else {
		fmt.Println("Advanced: Successfully dropped 'by_city' index.")
	}
}
```

---

## Project Architecture

`go-store` is structured around several key components that work together to provide a robust in-memory document store.

*   **`Store`**: The central component of the database. It manages a collection of `DocumentHandle`s and a set of `fieldIndex`es. It coordinates all document operations (Insert, Update, Delete, Get) and index management (Create, Drop, Lookup). All access to the document and index maps are protected by a global RWMutex.
*   **`Document`**: A type alias for `map[string]any`, representing the flexible, schemaless data structure for each record in the store. Values are deep-copied on mutations to ensure immutability of snapshots.
*   **`DocumentHandle`**: A thread-safe wrapper around a document's current state. It uses `atomic.Pointer` and `sync.RWMutex` to manage the transition between different `DocumentSnapshot`s, ensuring atomic updates and safe concurrent reads. It's the primary reference to a document within the store's internal maps.
*   **`DocumentSnapshot`**: An immutable, versioned snapshot of a document's data. Each update to a document generates a new snapshot. It includes a `refCount` for memory management, ensuring that snapshots are only cleared when no longer referenced by any `DocumentHandle` or ongoing query.
*   **`fieldIndex`**: Represents a B-tree based index for one or more document fields. It uses `github.com/google/btree` for efficient storage and retrieval of `indexEntry` items. It maintains `docRefs` (map of document IDs to `DocumentHandle`s) for each key, allowing quick access to matching documents. Each `fieldIndex` has its own RWMutex for concurrent access.
*   **`DocumentStream`**: Provides an iterator-like interface for consuming documents from the store. It's backed by Go channels and includes context-based cancellation, enabling efficient streaming of large result sets without loading all documents into memory at once.

### Data Flow

1.  **Insertion (`Insert`)**: A new document `ID` is generated, a `DocumentSnapshot` is created (versioned 1), and a `DocumentHandle` is initialized. This handle is added to the `Store`'s `documents` map. All active `fieldIndex`es are then updated to include the new document.
2.  **Update (`Update`)**: A new `DocumentSnapshot` with an incremented global version is created from the updated data. The `DocumentHandle`'s `atomic.Pointer` is atomically swapped to point to this new snapshot. The old snapshot's reference count is decremented. All `fieldIndex`es are updated to reflect potential changes in indexed fields.
3.  **Deletion (`Delete`)**: The `DocumentHandle` is removed from the `Store`'s `documents` map, and its internal snapshot pointer is set to `nil`. The final `DocumentSnapshot` is then passed to all `fieldIndex`es for removal, and its reference count is decremented.
4.  **Retrieval (`Get`, `Lookup`, `Stream`)**: When a document is read, its `DocumentHandle` is accessed. A `DocumentSnapshot` is retrieved, and its reference count is temporarily incremented (`read()` method) to prevent it from being garbage collected while in use. Once processed, the snapshot's reference count is decremented (`release()` method).

---

## Development & Contributing

We welcome contributions to `go-store`! Here's how to get started.

### Development Setup

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/asaidimu/go-store/v3.git
    cd go-store
    ```
2.  **Build the project**:
    ```bash
    go build -v ./...
    ```
    This command compiles all packages in the current module.

### Scripts

The project includes a simple `Makefile` for common development tasks:

*   `make build`: Compiles the entire project. Equivalent to `go build -v ./...`.
*   `make test`: Runs all unit tests with verbose output. Equivalent to `go test -v ./...`.
*   `make clean`: Removes generated executable files.

### Testing

To run the test suite and ensure everything is working correctly, execute:

```bash
go test -v ./...
```

To run performance benchmarks, which are crucial for an in-memory store:

```bash
go test -bench=. -benchmem
```

The tests cover basic CRUD operations, indexing, concurrency, and error handling.

### Contributing Guidelines

Contributions are what make the open-source community an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1.  **Fork the repository**: Click the "Fork" button at the top right of this page.
2.  **Create your feature branch**:
    ```bash
    git checkout -b feature/amazing-feature
    ```
3.  **Commit your changes**: We follow Conventional Commits (see below).
    ```bash
    git commit -m 'feat: Add some amazing feature'
    ```
4.  **Push to the branch**:
    ```bash
    git push origin feature/amazing-feature
    ```
5.  **Open a Pull Request**: Describe your changes clearly and link to any relevant issues.

#### Commit Message Format

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for automated semantic versioning. Please adhere to this format for your commit messages:

*   `feat`: A new feature (corresponds to `MINOR` in SemVer)
*   `fix`: A bug fix (corresponds to `PATCH` in SemVer)
*   `build`: Changes that affect the build system or external dependencies
*   `ci`: Changes to CI configuration files and scripts
*   `docs`: Documentation only changes
*   `perf`: A code change that improves performance
*   `refactor`: A code change that neither fixes a bug nor adds a feature
*   `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semicolons, etc.)
*   `test`: Adding missing tests or correcting existing tests

For breaking changes, append `!` after the type/scope, or include `BREAKING CHANGE:` in the footer:

*   `feat!: introduce breaking API change`
*   `fix(auth)!: correct authentication flow`

### Issue Reporting

If you find a bug or have a feature request, please open an issue on the [GitHub Issues page](https://github.com/asaidimu/go-store/v3/issues). When reporting a bug, please include:

*   A clear and concise description of the problem.
*   Steps to reproduce the behavior.
*   Expected behavior.
*   Actual behavior.
*   Your Go version and OS.

---

## Additional Information

### Troubleshooting

*   **`ErrDocumentNotFound`**: This error indicates that the document with the specified ID does not exist in the store, or it has been deleted.
*   **`ErrIndexNotFound`**: Occurs when attempting to `Lookup` or `DropIndex` an index that hasn't been created yet.
*   **`ErrIndexExists`**: You cannot create an index with a name that already exists. Drop the existing index first if you need to redefine it.
*   **`ErrStoreClosed`**: Operations on the `Store` are not permitted after `s.Close()` has been called.
*   **Data Loss**: Remember that `go-store` is an in-memory database. All data is lost when the application exits. For persistence, you would need to implement a serialization/deserialization layer or use an external persistent store.

### FAQ

**Q: Is `go-store` a persistent database?**
A: No, `go-store` is an in-memory database. All data is volatile and will be lost when the application shuts down or crashes. It's ideal for caching, session management, or in-process data storage.

**Q: How does `go-store` handle concurrency?**
A: `go-store` is designed to be concurrency-safe. It uses `sync.RWMutex` for broad store-level operations (like adding/removing documents or indexes) and `atomic.Pointer` along with reference counting (`DocumentSnapshot`) for fine-grained, non-blocking reads and atomic document state transitions. Indexes also have their own mutexes.

**Q: Does it support ACID properties?**
A: `go-store` provides ACID-like properties for single-document operations due to its atomic updates and versioning (optimistic concurrency). However, it does not support multi-document transactions in the traditional sense, as it's a simple key-value document store.

**Q: Can I use `go-store` for large datasets?**
A: Performance for `go-store` is excellent for in-memory operations. However, scalability is limited by available RAM. For very large datasets, consider traditional disk-based databases or distributed systems.

### Roadmap

*   **Functional Indexing**: Implement `CreateFunctionalIndex` and `Filter` methods to allow custom filter functions to create and query specialized indexes.
*   **Persistence Layer**: Introduce optional mechanisms for snapshotting/recovering the store to disk.
*   **Change Feeds/Watchers**: Provide a way to subscribe to changes (insertions, updates, deletions) in the store.
*   **More Advanced Querying**: Support for more complex query operators beyond exact and range lookups.

### License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

### Acknowledgments

*   This project utilizes the `github.com/google/btree` library for efficient B-tree implementations in its indexing.
*   UUID generation is handled by `github.com/google/uuid`.
