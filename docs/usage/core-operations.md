# Core Operations

# Core Operations

This section covers the fundamental CRUD (Create, Read, Update, Delete) operations provided by `go-store` for managing documents.

## Essential Functions

### Inserting Documents

To add a new document to the store, use the `Insert` method. It generates a unique ID for the document and returns it upon successful insertion.

```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	// Create a document as a map[string]any
	doc := store.Document{
		"title": "The Go Programming Language",
		"author": "Brian Kernighan",
		"pages": 400,
		"published": 2015
	}

	id, err := s.Insert(doc)
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
	}
	fmt.Printf("Inserted document with ID: %s\n", id)
}
```

### Retrieving Documents

Documents can be retrieved by their unique ID using the `Get` method. If the document is not found or has been deleted, an error is returned.

```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	docID, _ := s.Insert(store.Document{"name": "Example Document"})

	retrievedDoc, err := s.Get(docID)
	if err != nil {
		log.Fatalf("Failed to get document %s: %v", docID, err)
	}
	fmt.Printf("Retrieved document (ID: %s, Version: %d): %+v\n",
		retrievedDoc.ID, retrievedDoc.Version, retrievedDoc.Data)

	// Attempt to get a non-existent document
	_, err = s.Get("non-existent-id")
	if err == store.ErrDocumentNotFound {
		fmt.Println("Successfully handled non-existent document lookup.")
	} else if err != nil {
		log.Fatalf("Expected ErrDocumentNotFound, but got: %v", err)
	}
}
```

### Updating Documents

The `Update` method modifies an existing document. Provide the document ID and the new `Document` data. `go-store` performs a merge-like update: fields present in the new document replace existing ones, and new fields are added. Fields not present in the new document are retained from the previous version. Each update increments the document's internal version.

```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	docID, _ := s.Insert(store.Document{"product": "Laptop", "price": 1200, "in_stock": true})
	fmt.Printf("Initial product (ID: %s): %+v\n", docID, (func() store.Document { d, _ := s.Get(docID); return d.Data })())

	// Update price and add a new field "category"
	updateData := store.Document{
		"price": 1150,
		"category": "Electronics"
	}

	err := s.Update(docID, updateData)
	if err != nil {
		log.Fatalf("Failed to update document %s: %v", docID, err)
	}
	fmt.Printf("Updated product (ID: %s): %+v\n", docID, (func() store.Document { d, _ := s.Get(docID); return d.Data })())

	retrievedUpdatedDoc, _ := s.Get(docID)
	fmt.Printf("Updated document version: %d\n", retrievedUpdatedDoc.Version)
}
```

### Deleting Documents

To permanently remove a document from the store, use the `Delete` method. After deletion, attempts to `Get` or `Update` the document by its ID will result in an `ErrDocumentNotFound` error.

```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	docID, _ := s.Insert(store.Document{"status": "temporary"})
	fmt.Printf("Document to be deleted (ID: %s): %+v\n", docID, (func() store.Document { d, _ := s.Get(docID); return d.Data })())

	err := s.Delete(docID)
	if err != nil {
		log.Fatalf("Failed to delete document %s: %v", docID, err)
	}
	fmt.Printf("Document with ID %s deleted successfully.\n", docID)

	// Verify deletion
	_, err = s.Get(docID)
	if err == store.ErrDocumentNotFound {
		fmt.Println("Verification: Document is indeed not found after deletion.")
	} else if err != nil {
		log.Fatalf("Expected ErrDocumentNotFound after delete, but got: %v", err)
	}
}
```

---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*