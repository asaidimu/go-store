---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Methods Reference"
description: "Complete reference of all available methods and functions"
---
# Methods Reference

## `NewStore`

- **Use Case**: To initialize a new, empty in-memory document store. This must be the first call before performing any other store operations.
- **Signature**:

```go
func NewStore() *Store
```


- **Prerequisites**: None.
- **Side Effects**: Allocates and initializes internal data structures for documents and indexes.
- **Return Value**: A pointer to a new `Store` instance (`*Store`).
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Basic Document CRUD`

---

## `Insert`

- **Use Case**: To add a new document to the store. A unique ID is automatically generated for the document.
- **Signature**:

```go
func (s *Store) Insert(doc Document) (string, error)
```

- **Prerequisites**: The store must not be closed. The `doc` parameter must not be `nil`.
- **Side Effects**: Generates a new UUID for the document. Creates a new `DocumentHandle` and `DocumentSnapshot`. Adds the document to the store's internal map. Increments the global store version. Updates all active indexes to include the new document.
- **Return Value**: The generated unique identifier (`string`) for the new document and an `error` if the operation fails.
- **Exceptions**: `ErrStoreClosed`, `ErrInvalidDocument`
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Basic Document CRUD`
- **Related Errors**: `ErrStoreClosed`, `ErrInvalidDocument`

---

## `Update`

- **Use Case**: To modify an existing document identified by its ID. The provided document data completely replaces the existing data.
- **Signature**:

```go
func (s *Store) Update(docID string, doc Document) error
```


- **Prerequisites**: The store must not be closed. The `doc` parameter must not be `nil`. A document with the given `docID` must exist in the store and not have been concurrently deleted.
- **Side Effects**: Creates a new `DocumentSnapshot` with the updated data and an incremented version. Atomically replaces the old snapshot in the `DocumentHandle`. Updates all active indexes if the fields relevant to those indexes have changed. Releases the old `DocumentSnapshot` when no longer referenced.
- **Return Value**: `nil` on successful update, or an `error` if the document is not found, the store is closed, or the input is invalid.
- **Exceptions**: `ErrStoreClosed`, `ErrInvalidDocument`, `ErrDocumentNotFound`, `ErrDocumentDeleted`
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Basic Document CRUD`, `Concurrent Update Pattern`
- **Related Errors**: `ErrStoreClosed`, `ErrInvalidDocument`, `ErrDocumentNotFound`, `ErrDocumentDeleted`

---

## `Delete`

- **Use Case**: To permanently remove a document from the store.
- **Signature**:

```go
func (s *Store) Delete(docID string) error
```

- **Prerequisites**: The store must not be closed. A document with the given `docID` must exist.
- **Side Effects**: Removes the `DocumentHandle` from the store's internal map. Removes the document's entry from all active indexes. Releases the `DocumentSnapshot` associated with the deleted document when its reference count drops to zero.
- **Return Value**: `nil` on successful deletion, or an `error` if the document is not found or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrDocumentNotFound`
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Basic Document CRUD`
- **Related Errors**: `ErrStoreClosed`, `ErrDocumentNotFound`

---

## `Get`

- **Use Case**: To retrieve a single document by its unique identifier.
- **Signature**:

```go
func (s *Store) Get(docID string) (*DocumentResult, error)
```

- **Prerequisites**: The store must not be closed. The document with `docID` must exist and not have been deleted.
- **Side Effects**: None (read-only operation).
- **Return Value**: A pointer to a `DocumentResult` containing the document's ID, a deep copy of its data, and its version. Returns `nil` and an `error` if the document is not found, has been deleted, or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrDocumentNotFound`, `ErrDocumentDeleted`
- **Availability**: sync
- **Status**: active
- **Related Types**: `DocumentResult`
- **Related Patterns**: `Basic Document CRUD`
- **Related Errors**: `ErrStoreClosed`, `ErrDocumentNotFound`, `ErrDocumentDeleted`

---

## `Stream`

- **Use Case**: To obtain an iterator-like stream for efficiently processing all documents currently in the store without loading them all into memory at once. It provides a consistent snapshot of documents at the time of its creation.
- **Signature**:

```go
func (s *Store) Stream(bufferSize int) *DocumentStream
```

- **Prerequisites**: The store must not be closed.
- **Side Effects**: Initializes a new `DocumentStream` and starts a goroutine to populate it with documents from the store's current snapshot. This is a read-only operation and does not modify the store's state.
- **Return Value**: A pointer to a new `DocumentStream` instance. You must call `DocumentStream.Close()` on the returned stream when you are finished consuming documents to release resources.
- **Exceptions**: `ErrStoreClosed`
- **Availability**: async
- **Status**: active
- **Related Patterns**: `Streaming Documents`
- **Related Errors**: `ErrStoreClosed`

---

## `CreateIndex`

- **Use Case**: To build a new B-tree based index on one or more specified document fields. This enables fast exact and range lookups.
- **Signature**:

```go
func (s *Store) CreateIndex(indexName string, fields []string) error
```

- **Prerequisites**: The store must not be closed. The `indexName` must be unique (an index with the same name must not already exist). The `fields` slice must not be empty.
- **Side Effects**: Allocates memory for a new `fieldIndex` (B-tree). Populates the new index with existing documents from the store that contain all specified fields. Adds the new index to the store's internal index map.
- **Return Value**: `nil` on successful index creation, or an `error` if the index name already exists, the `fields` slice is empty, or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrEmptyIndex`, `ErrIndexExists`
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Indexed Lookup`
- **Related Errors**: `ErrStoreClosed`, `ErrEmptyIndex`, `ErrIndexExists`

---

## `DropIndex`

- **Use Case**: To remove an existing index from the store, freeing up its associated memory.
- **Signature**:

```go
func (s *Store) DropIndex(indexName string) error
```


- **Prerequisites**: The store must not be closed. An index with the given `indexName` must exist.
- **Side Effects**: Removes the specified index from the store's internal index map. Releases the memory allocated by the index's B-tree and its entries.
- **Return Value**: `nil` on successful index removal, or an `error` if the index does not exist or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrIndexNotFound`
- **Availability**: sync
- **Status**: active
- **Related Errors**: `ErrStoreClosed`, `ErrIndexNotFound`

---

## `Lookup`

- **Use Case**: To find documents that exactly match a given set of values on an existing index.
- **Signature**:

```go
func (s *Store) Lookup(indexName string, values []any) ([]*DocumentResult, error)
```

- **Prerequisites**: The store must not be closed. An index with the given `indexName` must exist.
- **Side Effects**: None (read-only operation).
- **Return Value**: A slice of pointers to `DocumentResult` instances that match the query. Returns an empty slice if no documents are found, or `nil` and an `error` if the index does not exist or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrIndexNotFound`
- **Availability**: sync
- **Status**: active
- **Related Types**: `DocumentResult`
- **Related Patterns**: `Indexed Lookup`
- **Related Errors**: `ErrStoreClosed`, `ErrIndexNotFound`

---

## `LookupRange`

- **Use Case**: To find documents within a specified range of values on an existing index. This is particularly useful for numerical or lexicographically sortable fields.
- **Signature**:

```go
func (s *Store) LookupRange(indexName string, minValues, maxValues []any) ([]*DocumentResult, error)
```

- **Prerequisites**: The store must not be closed. An index with the given `indexName` must exist. The `minValues` should logically precede or be equal to `maxValues` according to the index's sorting logic, otherwise an empty result set will be returned.
- **Side Effects**: None (read-only operation).
- **Return Value**: A slice of pointers to `DocumentResult` instances that fall within the specified range. Returns an empty slice if no documents are found within the range, or `nil` and an `error` if the index does not exist or the store is closed.
- **Exceptions**: `ErrStoreClosed`, `ErrIndexNotFound`
- **Availability**: sync
- **Status**: active
- **Related Types**: `DocumentResult`
- **Related Patterns**: `Indexed Lookup`
- **Related Errors**: `ErrStoreClosed`, `ErrIndexNotFound`

---

## `DocumentStream.Next`

- **Use Case**: To retrieve the next available document from an active stream. This method blocks until a document is available, the stream is closed, or an error occurs.
- **Signature**:

```go
func (ds *DocumentStream) Next() (DocumentResult, error)
```

- **Prerequisites**: The `DocumentStream` must be active (not closed yet).
- **Side Effects**: Consumes one document from the internal stream channel.
- **Return Value**: A `DocumentResult` struct if a document is available. Returns an empty `DocumentResult{}` and an `error` if the stream is closed, exhausted, or encounters an issue (e.g., context cancellation).
- **Exceptions**: `ErrStreamClosed`, `context.Canceled`, `context.DeadlineExceeded`
- **Availability**: sync
- **Status**: active
- **Related Types**: `DocumentResult`
- **Related Patterns**: `Streaming Documents`
- **Related Errors**: `ErrStreamClosed`

---

## `DocumentStream.Close`

- **Use Case**: To explicitly close a `DocumentStream`, releasing its resources and signaling that no more documents should be processed. This is crucial for resource management, especially when you stop consuming documents early.
- **Signature**:

```go
func (ds *DocumentStream) Close()
```

- **Prerequisites**: None.
- **Side Effects**: Cancels the stream's internal `context.Context` and closes its `results` and `errors` channels. Any pending `Next()` calls will be unblocked and return `ErrStreamClosed` or a context-related error.
- **Return Value**: None
- **Availability**: sync
- **Status**: active
- **Related Patterns**: `Streaming Documents`

---

## `Store.Close`

- **Use Case**: To gracefully shut down the store and release all associated resources. This should be called when the application no longer needs the store.
- **Signature**:

```go
func (s *Store) Close()
```

- **Prerequisites**: None.
- **Side Effects**: Sets the store's internal `closed` flag to true, preventing any further operations. Clears the `documents` and `indexes` maps to aid Go's garbage collection. Releases all `DocumentSnapshot` resources once their reference counts drop to zero.
- **Return Value**: None
- **Availability**: sync
- **Status**: active
- **Related Errors**: `ErrStoreClosed`

---

