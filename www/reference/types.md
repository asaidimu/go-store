---
outline: "deep"
lastUpdated: true
editLink: true
sidebar: true
title: "Types Reference"
description: "Complete reference of all type definitions"
---
# Types Reference

## `DocumentResult`

### Definition



```go
type DocumentResult struct {
    ID      string   // Document identifier
    Data    Document // Document data (deep copy)
    Version uint64   // Document version
}
```


**Purpose**: Represents a snapshot of a document returned from a query or retrieval operation. It includes the document's unique identifier, a deep copy of its data, and its current version number.

**Related Methods**: `Get`, `Stream`, `Lookup`, `LookupRange`

### Interface Contract

#### Parameter Object Structures


---

