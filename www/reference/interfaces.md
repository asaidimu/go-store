---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Interfaces Reference"
description: "Complete reference of all interface definitions"
---
# Interfaces Reference

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

#### Parameter Object Structures


---

