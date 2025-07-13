# Core Operations

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Core Operations"
description: "Core Operations documentation and guidance"
---

This section elaborates on the fundamental document management operations provided by `go-store`.

### Initializing the Store

Before you can perform any operations, you must create a new store instance.

```go
s := store.NewStore()
defer s.Close() // Crucial for releasing resources
```

The `NewStore()` function returns a pointer to a `Store` instance. It's vital to call `s.Close()` when your application no longer needs the store, typically using `defer`, to ensure all internal resources and memory are properly released, especially when the application exits or the store goes out of scope.

### Inserting Documents

The `Insert` method adds a new document to the store. A unique ID is automatically generated for each document.

```go
doc := store.Document{"item": "Laptop", "price": 1200.0, "in_stock": true}
id, err := s.Insert(doc)
if err != nil {
    // Handle error, e.g., ErrInvalidDocument or ErrStoreClosed
    log.Printf("Error inserting document: %v", err)
}
fmt.Printf("Inserted document with ID: %s\n", id)
```

*   **Document Structure**: `store.Document` is a `map[string]any`, allowing for flexible, schemaless data.
*   **Return Value**: Returns the generated `string` ID of the new document and an `error`.
*   **Side Effects**: Increments the internal global version counter and triggers asynchronous updates to all existing indexes that might include the fields present in the new document.

### Retrieving Documents

The `Get` method allows you to retrieve a document by its unique ID.

```go
retrievedDoc, err := s.Get(id)
if err != nil {
    // Handle error, e.g., ErrDocumentNotFound or ErrDocumentDeleted
    log.Printf("Error retrieving document %s: %v", id, err)
} else {
    fmt.Printf("Retrieved document version: %d, data: %v\n", retrievedDoc.Version, retrievedDoc.Data)
}
```

*   **Return Value**: Returns a `*store.DocumentResult` containing the `ID`, a deep copy of the `Data`, and the `Version` of the document, or `nil` and an `error`.
*   **Deep Copy**: The `Data` field in `DocumentResult` is a deep copy of the internal document state. This means you can modify `retrievedDoc.Data` without affecting the store's internal state.

### Updating Documents

To modify an existing document, use the `Update` method with its ID and the new document data.

```go
updatedData := store.Document{"item": "Laptop Pro", "price": 1500.0, "status": "available"}
err = s.Update(id, updatedData)
if err != nil {
    // Handle error, e.g., ErrDocumentNotFound or ErrInvalidDocument
    log.Printf("Error updating document %s: %v", id, err)
}
```

*   **Behavior**: The `doc` provided completely replaces the existing document's data. If new fields are present, they are added. If existing fields are not in `updatedData`, they are removed. If fields present in `updatedData` are already in the document, their values are overwritten.
*   **Optimistic Concurrency**: The store automatically increments the document's internal version. If multiple concurrent updates occur to the same document, `go-store`'s internal optimistic concurrency ensures that only one "wins" based on the internal locking mechanism, and the version reflects the total number of modifications.
*   **Side Effects**: Updates the document's internal snapshot and increments its version. Triggers updates to all indexes if indexed fields have changed.

### Deleting Documents

The `Delete` method permanently removes a document from the store.

```go
err = s.Delete(id)
if err != nil {
    // Handle error, e.g., ErrDocumentNotFound
    log.Printf("Error deleting document %s: %v", id, err)
}
```

*   **Idempotency**: Deleting a non-existent document will result in `ErrDocumentNotFound`.
*   **Side Effects**: Removes the document from the main document map and from all associated indexes. Memory is released through garbage collection when the `DocumentSnapshot`'s reference count drops to zero.


---
*Generated using Gemini AI on 7/13/2025, 8:40:41 PM. Review and refine as needed.*