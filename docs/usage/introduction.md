# Introduction

**Software Type**: API/Library (Confidence: 95%)

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Introduction"
description: "Introduction to go-store: an in-memory, concurrent-safe document store"
---
# Welcome to go-store!

`go-store` is an efficient, in-memory, concurrent-safe document store with built-in indexing capabilities. It's designed for Go applications requiring fast, concurrent access to schemaless data, making it ideal for caching layers, temporary data storage, or embedded application-level databases.

## Core Concepts

At its heart, `go-store` operates on documents, which are flexible, schemaless key-value maps (`map[string]any`). Each document is assigned a unique identifier (UUID) upon insertion and is versioned to support optimistic concurrency control.

### In-Memory Design

All data is held in RAM, providing lightning-fast read and write operations. However, this also means data is volatile and will be lost when the application shuts down or crashes. For persistence, `go-store` would need to be integrated with an external serialization/deserialization mechanism.

### Concurrency Model

`go-store` is built with concurrency as a first-class citizen. It uses a combination of `sync.RWMutex` for coarse-grained protection of overall store structures (like adding/removing indexes or documents from internal maps) and `atomic.Pointer` for fine-grained, non-blocking updates and reads of individual document states. This allows multiple goroutines to safely interact with the store simultaneously without corrupting data.

### Indexing

To enable efficient data retrieval, `go-store` supports the creation of B-tree based indexes on document fields. These indexes can be single-field or composite (multiple fields), allowing for fast exact-match lookups and range queries.

## Architecture Overview

The system is composed of several interconnected internal components:

*   **`Store`**: The central orchestrator, managing documents and indexes.
*   **`Document`**: The basic unit of data, represented as a flexible `map[string]any`.
*   **`DocumentHandle`**: An internal, thread-safe reference to a document, managing its versioned snapshots.
*   **`DocumentSnapshot`**: An immutable, versioned copy of a document's data, used for consistent reads and managed with reference counting for memory safety.
*   **`fieldIndex`**: Represents an individual B-tree index, facilitating fast queries.
*   **`DocumentStream`**: Provides an iterator-like interface for efficient, buffered iteration over documents.

This robust architecture ensures data consistency, high performance, and safe concurrent access in Go applications.


---
*Generated using Gemini AI on 7/19/2025, 12:26:00 PM. Review and refine as needed.*