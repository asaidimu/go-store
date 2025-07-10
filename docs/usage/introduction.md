# Introduction

**Software Type**: API/Library (Confidence: 100%)

# go-store

[![Go Reference](https://pkg.go.dev/badge/github.com/asaidimu/go-store.svg)](https://pkg.go.dev/github.com/asaidimu/go-store)
[![Build Status](https://github.com/asaidimu/go-store/workflows/Test%20Workflow/badge.svg)](https://github.com/asaidimu/go-store/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An efficient, in-memory, concurrent-safe document store with built-in indexing capabilities.

**Note:** This project is currently in beta. While designed for concurrency and robustness, use conservatively and consider its in-memory nature for production environments.

`go-store` is a lightweight, high-performance in-memory document database designed for applications requiring fast, concurrent access to schemaless data. It provides core CRUD (Create, Read, Update, Delete) operations and advanced features like field-based indexing, range queries, and document streaming.

The store leverages Go's concurrency primitives (`sync.RWMutex`, `atomic.Pointer`) and `github.com/google/btree` for efficient, thread-safe operations. Documents are stored as flexible `map[string]any` types, and changes are managed via an optimistic concurrency control mechanism using versioned `DocumentSnapshot`s and reference counting, ensuring data consistency even under heavy concurrent loads.

Key considerations for `go-store` are its in-memory nature (data is not persisted to disk and will be lost upon application shutdown) and its focus on efficient concurrent access patterns, making it suitable for caching layers, temporary data storage, or embedded application-level databases where persistence is handled externally or not required.

### Key Features

*   💾 **In-Memory Storage**: All data resides in RAM for maximum speed.
*   ⚙️ **Concurrency Safe**: Designed from the ground up for safe concurrent access by multiple goroutines using fine-grained locking and atomic operations.
*   🚀 **Optimistic Concurrency Control**: Documents are versioned, allowing for robust updates and reads without explicit transaction management for single document operations.
*   🔍 **Flexible Document Schema**: Documents are simple `map[string]any`, allowing for dynamic, schemaless data structures.
*   ⚡ **Field-Based Indexing**: Create B-tree based indexes on single or composite document fields for efficient lookups.
*   🎯 **Exact Lookups**: Quickly retrieve documents matching exact values on indexed fields.
*   ↔️ **Range Queries**: Query documents within a specified range on indexed fields.
*   🌊 **Document Streaming**: Iterate efficiently over all documents in the store, providing a consistent snapshot.
*   🗑️ **Graceful Shutdown**: Proper resource release upon store closure.
*   🚫 **Custom Error Handling**: Specific error types for common scenarios like `ErrDocumentNotFound` or `ErrIndexExists`.

---
*Generated using Gemini AI on 7/10/2025, 1:23:49 PM. Review and refine as needed.*