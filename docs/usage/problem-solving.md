# Problem Solving

# Problem Solving

This section provides guidance on common issues you might encounter while using `go-store`, including troubleshooting steps and error explanations.

## Troubleshooting

### Store Is Unresponsive / Operations Fail

**Symptom**: `Insert`, `Get`, `Update`, `Delete`, `CreateIndex`, `DropIndex`, `Lookup`, `LookupRange` methods return `ErrStoreClosed`.

**Check**: Verify that `s.Close()` has not been called prematurely. Once a `Store` instance is closed, it cannot be reused.

**Fix**: Ensure `s.Close()` is called only when the store is no longer needed, typically at the end of your application's lifecycle, often using `defer s.Close()` right after `store.NewStore()`.

### Unexpected Data Loss

**Symptom**: Data previously inserted into the store is missing after an application restart or crash.

**Check**: Remember that `go-store` is an **in-memory database**. By design, it does not persist data to disk. All data is lost when the application exits or crashes.

**Fix**: For applications requiring persistence, `go-store` should be used as a caching layer or a transient data store. You will need to implement an external persistence mechanism (e.g., saving data to a file, database, or remote service) to load data into the store at startup and save it at shutdown.

### Concurrent Write Conflicts

**Symptom**: When multiple goroutines update the same document, the final state isn't what you expect from a simple sequential view (e.g., a counter doesn't increment perfectly).

**Check**: `go-store` uses optimistic concurrency control. While it ensures no data corruption, concurrent `Update` calls on the same document operate on a "last write wins" basis for the document's overall data. The document's `Version` will always increment with each successful write, indicating the total number of updates.

**Fix**: If you need strict, atomically incrementing counters or complex transactional guarantees across specific fields, implement higher-level synchronization or a compare-and-swap (CAS) logic around the `Get` and `Update` calls. For most document-oriented updates where eventual consistency is acceptable for concurrent modifications of different fields, `go-store`'s model is sufficient.

## Error Reference

`go-store` defines several custom error types to provide clear indications of what went wrong.

*   **`ErrDocumentNotFound`**
    *   **Trigger**: Attempting to `Get`, `Update`, or `Delete` a document using an ID that does not exist or refers to a document that has already been deleted.
    *   **Handling**: Check for this error after `Get`, `Update`, or `Delete` operations and respond appropriately, e.g., inform the user, log the event, or skip the operation.

*   **`ErrDocumentDeleted`**
    *   **Trigger**: An internal error indicating that a document handle no longer points to a valid snapshot, likely because the document was deleted concurrently. This is usually caught internally and surfaces as `ErrDocumentNotFound` from public APIs.
    *   **Handling**: Less likely to be handled directly by users from public API calls, as `ErrDocumentNotFound` is the typical outward manifestation. It is an internal state error.

*   **`ErrIndexExists`**
    *   **Trigger**: Attempting to `CreateIndex` with a name that is already in use by an existing index.
    *   **Handling**: Before creating an index, you might check if it exists or simply handle this error. If you need to redefine an index, `DropIndex` it first.

*   **`ErrEmptyIndex`**
    *   **Trigger**: Attempting to `CreateIndex` with an empty slice of `fields`.
    *   **Handling**: Always provide at least one field name when creating an index.

*   **`ErrIndexNotFound`**
    *   **Trigger**: Attempting to `Lookup`, `LookupRange`, or `DropIndex` an index using a name that does not correspond to any active index in the store.
    *   **Handling**: Ensure the index name used matches an index previously created with `CreateIndex`.

*   **`ErrStreamClosed`**
    *   **Trigger**: Attempting to call `Next()` on a `DocumentStream` that has already been exhausted or explicitly `Close()`d.
    *   **Handling**: This is the normal way a stream signals completion. Break from your `for` loop when `Next()` returns this error.

*   **`ErrStoreClosed`**
    *   **Trigger**: Any operation performed on a `Store` instance after its `Close()` method has been called.
    *   **Handling**: Prevent operations on a closed store. Design your application's lifecycle to ensure the store remains open for as long as it's needed.

*   **`ErrInvalidDocument`**
    *   **Trigger**: Attempting to `Insert` or `Update` a document with a `nil` `Document` object.
    *   **Handling**: Always provide a non-`nil` `store.Document` (even an empty one, `store.Document{}`) when inserting or updating.

---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*