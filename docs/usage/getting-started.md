# Getting Started

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Getting Started"
description: "Getting Started documentation and guidance"
---

This section will guide you through setting up `go-store` and performing your first tasks.

### Prerequisites

To use `go-store`, you need a working Go environment. The library is tested with and recommends:

*   [Go](https://go.dev/dl/) (version 1.24.4 or higher, as specified in `go.mod` if present).

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

You can verify the installation and explore basic functionality by running the provided examples from the `go-store` repository:

1.  **Clone the repository**:

```bash
git clone https://github.com/asaidimu/go-store/v3.git
cd go-store
```

2.  **Run examples**:
```bash
go run examples/basic/main.go
go run examples/intermediate/main.go
go run examples/advanced/main.go
```

### Basic Document Operations

The most common operations involve creating, reading, updating, and deleting documents. Documents are represented as a `store.Document` (which is a `map[string]any`).

```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	// 1. Initialize the Store
	// Always defer s.Close() to ensure resources are properly released.
	s := store.NewStore()
	defer s.Close()

	// 2. Insert a New Document
	// The Insert method returns the unique ID of the new document.
	doc1 := store.Document{"title": "My First Doc", "author": "Alice"}
	id1, err := s.Insert(doc1)
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
	}
	fmt.Printf("Inserted document with ID: %s\n", id1)

	// 3. Retrieve a Document by ID
	// Get returns a DocumentResult containing the ID, data, and version.
	retrievedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Failed to get document %s: %v", id1, err)
	}
	fmt.Printf("Retrieved: ID=%s, Title='%s', Version=%d\n",
		retrievedDoc.ID, retrievedDoc.Data["title"], retrievedDoc.Version)

	// 4. Update an Existing Document
	// Provide the ID and the new document data. Existing fields are replaced,
	// new fields are added, and fields not present in the new doc are removed.
	updatedDoc1 := store.Document{"title": "My First Doc (Revised)", "author": "Alice Smith", "pages": 150}
	err = s.Update(id1, updatedDoc1)
	if err != nil {
		log.Fatalf("Failed to update document %s: %v", id1, err)
	}
	fmt.Printf("Updated document with ID: %s\n", id1)

	// Verify the update
	retrievedUpdatedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Failed to get updated document %s: %v", id1, err)
	}
	fmt.Printf("Updated retrieved: ID=%s, Title='%s', Pages=%.0f, Version=%d\n",
		retrievedUpdatedDoc.ID, retrievedUpdatedDoc.Data["title"], retrievedUpdatedDoc.Data["pages"], retrievedUpdatedDoc.Version)

	// 5. Delete a Document
	err = s.Delete(id1)
	if err != nil {
		log.Fatalf("Failed to delete document %s: %v", id1, err)
	}
	fmt.Printf("Deleted document with ID: %s\n", id1)

	// 6. Attempt to retrieve a deleted document (expected to fail)
	_, err = s.Get(id1)
	if err != nil {
		fmt.Printf("Attempted to get deleted document %s: %v (Expected error)\n", id1, err)
	}
}


---
*Generated using Gemini AI on 7/19/2025, 12:26:00 PM. Review and refine as needed.*