# Problem Solving

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Problem Solving"
description: "Problem Solving documentation and guidance"
---
# Problem Solving

## Problem Solving

This section addresses common issues you might encounter while using `go-store` and provides solutions or workarounds.

### Troubleshooting Common Errors

`go-store` defines specific error types to help you diagnose and handle issues programmatically. Here's a quick reference for the most common ones:

*   **`store.ErrDocumentNotFound`**: This error indicates that the document with the specified ID does not exist in the store, or it has been deleted. Check the document ID, ensure it was correctly inserted, and hasn't been removed.
*   **`store.ErrIndexNotFound`**: Occurs when attempting to `Lookup` or `DropIndex` an index that hasn't been created yet. Verify the index name and ensure `s.CreateIndex` was successfully called for that name.
*   **`store.ErrIndexExists`**: You cannot create an index with a name that already exists. If you need to redefine an index, `DropIndex` the existing one first, then `CreateIndex`.
*   **`store.ErrStoreClosed`**: Operations on the `Store` are not permitted after `s.Close()` has been called. Ensure you are not attempting to use a store instance after it has been explicitly closed.
*   **`store.ErrInvalidDocument`**: This error is returned if you attempt to `Insert` or `Update` a document with a `nil` `store.Document` value. Always provide a valid `map[string]any`.
*   **`store.ErrStreamClosed`**: This is a normal signal that a `DocumentStream` has completed iterating through all available documents or was explicitly closed via `DocumentStream.Close()`. It also appears if the store itself was closed while a stream was active.

### FAQ (Frequently Asked Questions)

**Q: Is `go-store` a persistent database?**
A: No, `go-store` is an in-memory database. All data is volatile and will be lost when the application shuts down or crashes. It's ideal for caching, session management, or in-process data storage where persistence is handled externally or not required.

**Q: How does `go-store` handle concurrency?**
A: `go-store` is designed to be concurrency-safe. It uses `sync.RWMutex` for broad store-level operations (like adding/removing documents or indexes) and `atomic.Pointer` along with reference counting (`DocumentSnapshot`) for fine-grained, non-blocking reads and atomic document state transitions. Indexes also have their own mutexes, ensuring consistent data access under high concurrency.

**Q: Does it support ACID properties?**
A: `go-store` provides ACID-like properties for single-document operations due to its atomic updates and versioning (optimistic concurrency). This means individual `Insert`, `Update`, `Delete` operations are atomic and isolated. However, it does not support multi-document transactions in the traditional relational database sense.

**Q: Can I use `go-store` for very large datasets?**
A: Performance for `go-store` is excellent for in-memory operations. However, scalability is fundamentally limited by available RAM on the machine where your application is running. For datasets that exceed your application's memory capacity, consider traditional disk-based databases or distributed systems.

**Q: Are functional indexes supported?**
A: As of the current version, `go-store` provides field-based indexes for exact and range lookups. The `advanced/main.go` example includes a *simulated* functional index, indicating this could be a potential future feature, but it's not natively implemented yet.


---
*Generated using Gemini AI on 7/13/2025, 8:40:41 PM. Review and refine as needed.*