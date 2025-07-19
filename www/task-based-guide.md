---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Task-Based Guide"
description: "Task-Based Guide documentation and guidance"
---
# Task-Based Guide

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Task-Based Guide"
description: "Task-Based Guide documentation and guidance"
---

This section focuses on common tasks you'll perform with `go-store` beyond basic CRUD operations.

### Indexing and Querying Data

Indexes are essential for efficient data retrieval, especially for large datasets. `go-store` supports B-tree based indexes on single or composite fields.

#### Creating an Index

Use `s.CreateIndex` to define a new index. You provide a unique name for the index and a slice of field names (`[]string`) on which the index should be built.

```go
// Create a single-field index on 'city'
err := s.CreateIndex("by_city", []string{"city"})
if err != nil {
    log.Fatalf("Failed to create by_city index: %v", err)
}
fmt.Println("Index 'by_city' created.")

// Create a composite index on 'product' and 'category'
err = s.CreateIndex("by_product_category", []string{"product", "category"})
if err != nil {
    log.Fatalf("Failed to create by_product_category index: %v", err)
}
fmt.Println("Index 'by_product_category' created.")

// Documents inserted after index creation will automatically be added to the index.
// Existing documents are also populated into the index upon creation.
_, _ = s.Insert(store.Document{"name": "Alice", "city": "New York", "product": "Book", "category": "Fiction"})
_, _ = s.Insert(store.Document{"name": "Bob", "city": "London", "product": "Laptop", "category": "Electronics"})
_, _ = s.Insert(store.Document{"name": "Charlie", "city": "New York", "product": "Book", "category": "Education"})
```

*   **`indexName`**: Must be unique within the store. Attempting to create an index with an existing name will return `ErrIndexExists`.
*   **`fields`**: A slice of strings specifying the document fields to index. The order matters for composite indexes as it defines the sorting order in the B-tree. An empty `fields` slice will result in `ErrEmptyIndex`.

#### Performing Exact Lookups

Use `s.Lookup` to find documents that exactly match specific values on an indexed field (or composite fields).

```go
fmt.Println("\nDocuments in New York:")
nyDocs, err := s.Lookup("by_city", []any{"New York"})
if err != nil {
    log.Fatalf("Lookup by_city failed: %v", err)
}
for _, doc := range nyDocs {
    fmt.Printf("  ID: %s, Name: %s\n", doc.ID, doc.Data["name"])
}

fmt.Println("\nBooks in Fiction category:")
bookFictionDocs, err := s.Lookup("by_product_category", []any{"Book", "Fiction"})
if err != nil {
    log.Fatalf("Lookup by_product_category failed: %v", err)
}
for _, doc := range bookFictionDocs {
    fmt.Printf("  ID: %s, Name: %s\n", doc.ID, doc.Data["name"])
}
```

*   **`values`**: A slice of `any` matching the number and order of `fields` specified during index creation. For a single-field index like `by_city`, you'd pass `[]any{"New York"}`.

#### Performing Range Queries

For numerical or comparable string fields, `s.LookupRange` allows you to retrieve documents within a specified range.

```go
err = s.CreateIndex("by_age", []string{"age"})
if err != nil {
    log.Fatalf("Failed to create by_age index: %v", err)
}

_, _ = s.Insert(store.Document{"name": "David", "age": 28})
_, _ = s.Insert(store.Document{"name": "Eve", "age": 30})
_, _ = s.Insert(store.Document{"name": "Frank", "age": 32})

fmt.Println("\nDocuments with age between 27 and 32 (inclusive):")
ageRangeDocs, err := s.LookupRange("by_age", []any{27}, []any{32})
if err != nil {
    log.Fatalf("Lookup age range failed: %v", err)
}
for _, doc := range ageRangeDocs {
    fmt.Printf("  ID: %s, Name: %s, Age: %.0f\n", doc.ID, doc.Data["name"], doc.Data["age"])
}
```

*   **`minValues`, `maxValues`**: Slices of `any` defining the lower and upper bounds of the range. Both bounds are inclusive. Like `Lookup`, the order and count of values must match the index's `fields`.

#### Dropping an Index

If an index is no longer needed, you can remove it using `s.DropIndex`.

```go
err = s.DropIndex("by_city")
if err != nil {
    log.Fatalf("Failed to drop by_city index: %v", err)
}
fmt.Println("Index 'by_city' dropped successfully.")
```

*   **Side Effects**: The index and its associated memory will be released. Attempting to drop a non-existent index will return `ErrIndexNotFound`.

### Streaming Documents

For iterating over all documents in the store, `s.Stream` provides an efficient, channel-based mechanism. It returns a `*store.DocumentStream` which can be consumed using its `Next()` method.

```go
fmt.Println("\nStreaming all documents:")
// Create a stream with a buffer size (e.g., 10 for demonstration). 
// A larger buffer can improve performance for high-throughput scenarios.
docStream := s.Stream(10)
defer docStream.Close() // Ensure the stream is closed to release resources

for {
    docResult, err := docStream.Next()
    if err != nil {
        // ErrStreamClosed indicates the stream has finished or was cancelled.
        if err == store.ErrStreamClosed { break }
        log.Fatalf("Stream error: %v", err)
    }
    fmt.Printf("  Streamed: %s\n", docResult.Data["item"])
    count++
}
fmt.Printf("Total streamed documents: %d\n", count)
```

*   **Snapshot Consistency**: When `Stream` is called, it captures a snapshot of the documents currently in the store. Subsequent modifications (inserts, updates, deletes) to the store will *not* affect the documents being streamed.
*   **Buffer Size**: The `bufferSize` parameter affects the capacity of the internal Go channel used for streaming. A non-zero buffer can improve performance by allowing documents to be pushed ahead of consumption, especially when the consumer is slower than the producer. A `bufferSize` of `0` creates an unbuffered channel.
*   **Cancellation**: The `DocumentStream` includes an internal `context.Context` which can be cancelled by calling `docStream.Close()`. This is crucial for stopping iteration early and releasing resources if you don't need to consume all documents.
*   **Error Handling**: `Next()` will return `store.ErrStreamClosed` when all documents have been streamed or if the stream was explicitly closed. Other errors might indicate underlying issues (e.g., context cancellation).


