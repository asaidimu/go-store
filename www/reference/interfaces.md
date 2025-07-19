---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Interfaces Reference"
description: "Complete reference of all interface definitions"
---
# Interfaces Reference

## `DocumentLike`

### Definition



```go
type DocumentLike interface {
	~map[string]any
}
```


**Purpose**: A type constraint that allows any type alias of `map[string]any` to be used as a document within generic contexts like `Cursor`.

### Interface Contract

#### Parameter Object Structures


---

## `Cursor`

### Definition



```go
type Cursor[T DocumentLike] interface {
	Next() (*T, bool /* has next */, error)
	Previous() (*T, bool /* has previous */, error)
	Advance(count int /* can be negative */) (*T, bool /* has next/previous depending on intended direction */, error)
	Reset() error     // For algorithms that need multiple passes
	Clone() Cursor[T] // For nested operations
	Count() int       // Maximum number of documents we can iterate over
	Close() error     // Clean up resources
}
```


**Purpose**: Provides a generic interface for bidirectional iteration over a collection of `DocumentLike` types, allowing flexible navigation and snapshot handling.

**Related Methods**: `StoreCursor.Next`, `StoreCursor.Previous`, `StoreCursor.Advance`, `StoreCursor.Reset`, `StoreCursor.Clone`, `StoreCursor.Count`, `StoreCursor.Close`

### Interface Contract

#### Required Methods

##### `Next`

**Signature**: 

```go
Next() (*T, bool, error)
```


- **Parameters**: 

```go

```

- **Return Value**: A pointer to the next document of type T, a boolean indicating if there are more documents, and an error if the stream is closed or the document is deleted.
- **Side Effects**: Advances the cursor's internal position by one. May return `ErrStreamClosed` if called on a closed cursor or `ErrDocumentDeleted` if the document at the next position was removed.

##### `Previous`

**Signature**: 

```go
Previous() (*T, bool, error)
```


- **Parameters**: 

```go

```

- **Return Value**: A pointer to the previous document of type T, a boolean indicating if there are more documents, and an error.
- **Side Effects**: Moves the cursor's internal position back by one. Equivalent to calling `Advance(-1)`.

##### `Advance`

**Signature**: 

```go
Advance(count int) (*T, bool, error)
```


- **Parameters**: 

```go

```

- **Return Value**: A pointer to the document at the new position, a boolean indicating if there are more documents in the direction of advance, and an error.
- **Side Effects**: Adjusts the cursor's internal position by `count`. Clamps to the start or end of the document list if `count` would move it out of bounds.

##### `Reset`

**Signature**: 

```go
Reset() error
```


- **Parameters**: 

```go

```

- **Return Value**: `nil` on success, `ErrStreamClosed` if the cursor is already closed.
- **Side Effects**: Resets the cursor's internal position to the beginning (index 0) of its document snapshot.

##### `Clone`

**Signature**: 

```go
Clone() Cursor[T]
```


- **Parameters**: 

```go

```

- **Return Value**: A new `Cursor` instance that is an independent copy of the current cursor, starting at the same position and referencing the same document snapshot.
- **Side Effects**: None directly on the original cursor; creates a new object.

##### `Count`

**Signature**: 

```go
Count() int
```


- **Parameters**: 

```go

```

- **Return Value**: The total number of documents in the cursor's snapshot.
- **Side Effects**: None (read-only operation).

##### `Close`

**Signature**: 

```go
Close() error
```


- **Parameters**: 

```go

```

- **Return Value**: `nil` on success.
- **Side Effects**: Marks the cursor as closed and releases its internal reference to the document handles, aiding garbage collection.

#### Parameter Object Structures


---

## `Document`

### Definition



```go
type Document map[string]any
```


**Purpose**: Represents a flexible, schemaless document in the store. It's a map where keys are string field names and values can be of any Go type (including nested maps, slices, and primitive types).

**Related Methods**: `Insert`, `Update`, `Get`, `Stream`

**Related Patterns**: `Basic Document CRUD`

### Interface Contract

#### Parameter Object Structures


---

## `DocumentStream`

### Definition



```go
type DocumentStream struct {
    results chan DocumentResult
    errors  chan error
    ctx     context.Context
    cancel  context.CancelFunc
}
```


**Purpose**: Provides an iterator-like interface for streaming documents from the store. It allows consuming documents asynchronously and efficiently without loading all results into memory at once.

**Related Methods**: `Stream`, `DocumentStream.Next`, `DocumentStream.Close`

**Related Patterns**: `Streaming Documents`

### Interface Contract

#### Required Methods

##### `Next`

**Signature**: 

```go
func (ds *DocumentStream) Next() (DocumentResult, error)
```


- **Parameters**: 

```go

```

- **Return Value**: A `DocumentResult` struct containing the document ID, a deep copy of its data, and its version. If the stream is exhausted or closed, `DocumentResult{}` is returned with an error.
- **Side Effects**: Consumes the next available document from the internal channel. If the channel is empty, it blocks until a document is available, an error occurs, or the stream is closed/cancelled.

##### `Close`

**Signature**: 

```go
func (ds *DocumentStream) Close()
```


- **Parameters**: 

```go

```

- **Return Value**: None
- **Side Effects**: Cancels the stream's internal `context.Context` and closes its `results` and `errors` channels. Any pending `Next()` calls will be unblocked and return `ErrStreamClosed` or a context-related error.

#### Parameter Object Structures


---

