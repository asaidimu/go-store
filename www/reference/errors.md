---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Error Reference"
description: "Complete error reference with scenarios, diagnosis, and resolution"
---
# Error Reference

## `ErrDocumentNotFound`

- **Type**: 

error
- **Symptoms**: 

A `*DocumentResult` pointer is `nil` and the returned `error` is `ErrDocumentNotFound` from `Get`, `Update`, `Delete`, `Lookup`, or `LookupRange` operations.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Attempting to retrieve a document with an ID that has never been inserted or has been successfully deleted.

**Example**:



```go
package main

import (
	"fmt"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()
	_, err := s.Get("non-existent-id")
	if err == store.ErrDocumentNotFound {
		fmt.Println("Successfully caught ErrDocumentNotFound for non-existent ID.")
	}
}
```


**Reason**:

 The provided `docID` does not correspond to any active document in the store's internal map.

---

###### Attempting to update or delete a document that has already been deleted (possibly by another concurrent operation).

**Example**:



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
	id, _ := s.Insert(store.Document{"foo": "bar"})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// This goroutine deletes the document
		err := s.Delete(id)
		if err != nil && err != store.ErrDocumentNotFound { log.Println(err) }
		fmt.Println("Deleter finished.")
	}()

	go func() {
		defer wg.Done()
		_ = s.Update(id, store.Document{"foo": "baz"})
		if err := s.Update(id, store.Document{"foo": "baz"}); err == store.ErrDocumentNotFound {
			fmt.Println("Updater caught ErrDocumentNotFound (document was deleted by other goroutine).")
		}
	}()
	wg.Wait()
}
```


**Reason**:

 The document's handle or associated snapshot was removed from the store before the operation could complete.

---

- **Diagnosis**: 

Verify the `docID` or index `values` used. If expected to exist, check if a preceding `Delete` operation or a concurrent process might have removed it. For lookups, ensure the index name is correct and values match the index definition.
- **Resolution**: 

For `Get`, `Update`, `Delete`: Ensure the document ID is correct and valid. If it's a transient condition due to concurrency, consider retry logic or design your application to gracefully handle non-existent documents. For `Lookup`/`LookupRange`: Ensure `indexName` is correct and `values` (or `minValues`/`maxValues`) are consistent with the index definition.
- **Prevention**: 

Implement ID validation. For concurrent scenarios, use patterns like idempotency or ensure operations are coordinated if a document's presence is strictly required. For indexes, ensure correct index names and field values are consistently used.
- **Handling Patterns**: 



```
Typically handled by checking `if err == store.ErrDocumentNotFound { ... }` and providing user feedback (e.g., "Item not found") or skipping the operation. No recovery is possible if the document is truly absent.
```

- **Propagation Behavior**: 

Bubbles up to the caller of the `Store` method. It is not caught internally by the store.

---

## `ErrDocumentDeleted`

- **Type**: 

error
- **Symptoms**: 

Returned by `Get` or `Update` if the document was previously logically deleted (its `DocumentHandle`'s snapshot pointer was set to `nil`) but its `DocumentHandle` still exists in the map (e.g., during a race condition between `Get`/`Update` and `Delete`).
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### A `Get` or `Update` operation races with a `Delete` operation. The document's `DocumentHandle` might still be in the `Store.documents` map, but its internal `current` snapshot pointer has been set to `nil` by the `Delete` operation.

**Example**:



```go
package main

import (
	"fmt"
	"sync"
	"time"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	id, _ := s.Insert(store.Document{"key": "value"})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond) // Give updater a chance to start
		_ = s.Delete(id)
		fmt.Println("Deleter finished.")
	}()

	go func() {
		defer wg.Done()
		_, err := s.Get(id)
		if err == store.ErrDocumentDeleted {
			fmt.Println("Getter caught ErrDocumentDeleted: document was concurrently deleted.")
		} else if err != nil {
			fmt.Printf("Getter caught unexpected error: %v\n", err)
		}
	}()
	wg.Wait()
}
```


**Reason**:

 The underlying `DocumentSnapshot` has been marked for deletion (its pointer set to nil), even if the `DocumentHandle` itself hasn't been completely removed from the store's map yet. This indicates the document is no longer active.

---

- **Diagnosis**: 

This error is transient and indicates a race condition where a document was accessed after it began its deletion process. It's often followed shortly by `ErrDocumentNotFound` if the `Delete` fully completes. This usually signifies that the document is indeed gone.
- **Resolution**: 

Treat `ErrDocumentDeleted` similarly to `ErrDocumentNotFound` in most application logic. It implies the document is no longer available. In robust concurrent systems, you might retry the operation if it was an update, but for `Get` it means the data is not there.
- **Prevention**: 

While hard to prevent race conditions entirely in high concurrency, ensuring that client-side operations account for documents potentially being deleted by other threads is key.
- **Handling Patterns**: 



```
Check `if err == store.ErrDocumentDeleted || err == store.ErrDocumentNotFound { ... }` to handle both states as an absent document. Log the error for debugging if unexpected, but don't typically retry immediately unless your application logic specifically requires it for idempotent operations.
```

- **Propagation Behavior**: 

Bubbles up to the caller of the `Store` method. It is not caught internally by the store.

---

## `ErrIndexExists`

- **Type**: 

error
- **Symptoms**: 

Calling `CreateIndex` with an `indexName` that is already in use by an existing index.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Attempting to create an index with a name that is already assigned to another index in the store.

**Example**:



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

	_ = s.CreateIndex("my_index", []string{"field1"})

	err := s.CreateIndex("my_index", []string{"field2"})
	if err == store.ErrIndexExists {
		fmt.Println("Successfully caught ErrIndexExists.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 The `Store` maintains a map of indexes by their unique names. An attempt was made to add a new entry with a key that already exists.

---

- **Diagnosis**: 

Verify index names before creation. Check if your application logic attempts to create the same index multiple times without prior checks or dropping.
- **Resolution**: 

Choose a unique name for your new index. If you intend to redefine an existing index, `DropIndex` it first, then `CreateIndex`.
- **Prevention**: 

Use a consistent naming convention for indexes. Check for index existence with a `Lookup` (which would return `ErrIndexNotFound` if not present) before attempting `CreateIndex` if dynamic index management is needed, or ensure `CreateIndex` is called only once during initialization.
- **Handling Patterns**: 



```
Catch `ErrIndexExists` and log it, or skip index creation if the goal is to ensure an index exists but not necessarily create a new one every time. If redefining is truly intended, follow `DropIndex` with `CreateIndex`.
```

- **Propagation Behavior**: 

Bubbles up to the caller of `CreateIndex`.

---

## `ErrEmptyIndex`

- **Type**: 

error
- **Symptoms**: 

Calling `CreateIndex` with an empty slice (`[]string{}`) for the `fields` parameter.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Providing an empty slice of field names when creating a new index.

**Example**:



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

	err := s.CreateIndex("invalid_index", []string{})
	if err == store.ErrEmptyIndex {
		fmt.Println("Successfully caught ErrEmptyIndex.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 An index requires at least one field to be defined for its keys. An empty `fields` slice is logically invalid for a B-tree index.

---

- **Diagnosis**: 

Review the `fields` parameter passed to `CreateIndex`. Ensure it contains at least one valid string representing a document field.
- **Resolution**: 

Provide a non-empty slice of field names. For example, `[]string{"name"}` or `[]string{"city", "age"}`.
- **Prevention**: 

Add input validation on the `fields` slice before calling `CreateIndex` if the field names are user-provided or dynamically generated.
- **Handling Patterns**: 



```
Catch `ErrEmptyIndex` and return a user-friendly error message, or correct the `fields` input.
```

- **Propagation Behavior**: 

Bubbles up to the caller of `CreateIndex`.

---

## `ErrIndexNotFound`

- **Type**: 

error
- **Symptoms**: 

Returned by `DropIndex`, `Lookup`, or `LookupRange` when the specified `indexName` does not correspond to any active index in the store.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Attempting to query or drop an index that has not been created or has already been dropped.

**Example**:



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

	// Try to lookup using a non-existent index
	_, err := s.Lookup("non_existent_index", []any{"value"})
	if err == store.ErrIndexNotFound {
		fmt.Println("Successfully caught ErrIndexNotFound for lookup.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// Try to drop a non-existent index
	err = s.DropIndex("another_non_existent_index")
	if err == store.ErrIndexNotFound {
		fmt.Println("Successfully caught ErrIndexNotFound for drop.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 The `indexName` provided does not match any index currently managed by the `Store`.

---

- **Diagnosis**: 

Verify the `indexName` spelling and confirm that the index was indeed created before attempting to use or drop it.
- **Resolution**: 

Ensure the index is created successfully with `CreateIndex` before performing `Lookup`, `LookupRange`, or `DropIndex` operations. Correct the `indexName` if misspelled.
- **Prevention**: 

Centralize index creation during application startup. Implement checks for index existence if dynamic index management is required.
- **Handling Patterns**: 



```
Catch `ErrIndexNotFound` and provide user feedback (e.g., "Index not found"). For idempotent operations like dropping an index, you might ignore this error if the goal is simply to ensure the index is not present.
```

- **Propagation Behavior**: 

Bubbles up to the caller of the respective `Store` method.

---

## `ErrStreamClosed`

- **Type**: 

error
- **Symptoms**: 

Returned by `DocumentStream.Next()` when all documents have been consumed from the stream, or the `DocumentStream.Close()` method has been called, or the `Store` itself has been closed.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Attempting to call `DocumentStream.Next()` after the stream has delivered all documents.

**Example**:



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
	_, _ = s.Insert(store.Document{"test": 1})

	stream := s.Stream(0)
	_, _ = stream.Next() // Consume the only document

	_, err := stream.Next() // Call again when exhausted
	if err == store.ErrStreamClosed {
		fmt.Println("Caught ErrStreamClosed after stream exhausted.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 The internal channel feeding the stream has been closed, indicating no more `DocumentResult` values will be sent.

---

###### Attempting to call `DocumentStream.Next()` after `DocumentStream.Close()` has been explicitly called.

**Example**:



```go
package main

import (
	"fmt"
	"time"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	defer s.Close()

	stream := s.Stream(0) // Unbuffered stream to easily block Next()

	go func() {
		// In a real scenario, this goroutine might be waiting for documents
		_, err := stream.Next()
		if err == store.ErrStreamClosed {
			fmt.Println("Goroutine caught ErrStreamClosed due to explicit stream close.")
		} else {
			fmt.Printf("Goroutine caught unexpected error: %v\n", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // Allow goroutine to reach Next()
	stream.Close() // Explicitly close the stream

	time.Sleep(10 * time.Millisecond) // Allow goroutine to finish
}
```


**Reason**:

 The stream's `Close()` method explicitly signals termination.

---

- **Diagnosis**: 

This error typically signifies the end of a stream. If it occurs unexpectedly, check if `DocumentStream.Close()` is being called prematurely or if the producing goroutine is exiting early.
- **Resolution**: 

Use `ErrStreamClosed` as the loop termination condition for consuming streams. Ensure `DocumentStream.Close()` is called only when the stream is truly no longer needed.
- **Prevention**: 

Always `defer stream.Close()` immediately after creating a stream. Structure stream consumption loops to gracefully break on `ErrStreamClosed`.
- **Handling Patterns**: 



```
The standard pattern is `for { doc, err := stream.Next(); if err != nil { if err == store.ErrStreamClosed { break }; // handle other errors; break }; // process doc }`.
```

- **Propagation Behavior**: 

Bubbles up to the caller of `DocumentStream.Next()`.

---

## `ErrStoreClosed`

- **Type**: 

error
- **Symptoms**: 

Any `Store` method (`Insert`, `Get`, `Update`, `Delete`, `Stream`, `CreateIndex`, `DropIndex`, `Lookup`, `LookupRange`) returning `ErrStoreClosed`.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Attempting to perform any operation on a `Store` instance after its `Close()` method has been called.

**Example**:



```go
package main

import (
	"fmt"
	"log"
	store "github.com/asaidimu/go-store/v3"
)

func main() {
	s := store.NewStore()
	s.Close() // Close the store immediately

	_, err := s.Insert(store.Document{"foo": "bar"})
	if err == store.ErrStoreClosed {
		fmt.Println("Successfully caught ErrStoreClosed.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 The store's internal `closed` atomic flag has been set to `true`, indicating it's no longer operational.

---

- **Diagnosis**: 

This indicates an attempt to interact with a store instance that has already been shut down. Review the lifecycle of your `Store` instance.
- **Resolution**: 

Ensure `Store` operations are only performed on an active, open store. If a store needs to be reused after closing, a new `Store` instance must be created with `NewStore()`.
- **Prevention**: 

Place `defer s.Close()` immediately after `s := store.NewStore()`. Ensure references to the `Store` instance are correctly scoped or nullified after closure to prevent accidental reuse.
- **Handling Patterns**: 



```
Typically indicates a programming error or an attempt to use a shared resource after it's been disposed. Handle by logging or returning a fatal error, as operations on a closed store are generally unrecoverable.
```

- **Propagation Behavior**: 

Bubbles up to the caller of the `Store` method.

---

## `ErrInvalidDocument`

- **Type**: 

error
- **Symptoms**: 

Calling `Insert` or `Update` with a `nil` `store.Document`.
- **Properties**: 

Standard Go `error` interface. No additional properties.

### Scenarios

###### Passing `nil` as the `doc` parameter to `Insert` or `Update`.

**Example**:



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

	_, err := s.Insert(nil) // Invalid
	if err == store.ErrInvalidDocument {
		fmt.Println("Caught ErrInvalidDocument for Insert with nil doc.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	id, _ := s.Insert(store.Document{"a":1})
	err = s.Update(id, nil) // Invalid
	if err == store.ErrInvalidDocument {
		fmt.Println("Caught ErrInvalidDocument for Update with nil doc.")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
```


**Reason**:

 The store expects valid document data (a `map[string]any`) for insertion and update operations to ensure data integrity.

---

- **Diagnosis**: 

Check the value of the `doc` parameter before calling `Insert` or `Update`. Ensure it is a properly initialized `store.Document` (or `map[string]any`) and not `nil`.
- **Resolution**: 

Always pass a non-`nil` `store.Document` to `Insert` and `Update`. Even if the document has no fields, pass an empty map: `store.Document{}`.
- **Prevention**: 

Implement input validation on user-provided document data if it can potentially be `nil`.
- **Handling Patterns**: 



```
Catch `ErrInvalidDocument` and provide specific feedback that the document content cannot be empty or invalid. Prevent the call from happening with a client-side check.
```

- **Propagation Behavior**: 

Bubbles up to the caller of `Insert` or `Update`.

---

