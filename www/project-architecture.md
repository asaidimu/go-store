---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Project Architecture"
description: "Project Architecture documentation and guidance"
---
# Project Architecture

---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Project Architecture"
description: "Project Architecture documentation and guidance"
---
# Project Architecture

## Project Architecture

`go-store` is structured around several key components that work together to provide a robust in-memory document store.

*   **`Store`**: The central component of the database. It manages a collection of `DocumentHandle`s and a set of `fieldIndex`es. It coordinates all document operations (Insert, Update, Delete, Get) and index management (Create, Drop, Lookup). All access to the document and index maps are protected by a global RWMutex.
*   **`Document`**: A type alias for `map[string]any`, representing the flexible, schemaless data structure for each record in the store. Values are deep-copied on mutations to ensure immutability of snapshots.
*   **`DocumentHandle`**: A thread-safe wrapper around a document's current state. It uses `atomic.Pointer` and `sync.RWMutex` to manage the transition between different `DocumentSnapshot`s, ensuring atomic updates and safe concurrent reads. It's the primary reference to a document within the store's internal maps.
*   **`DocumentSnapshot`**: An immutable, versioned snapshot of a document's data. Each update to a document generates a new snapshot. It includes a `refCount` for memory management, ensuring that snapshots are only cleared when no longer referenced by any `DocumentHandle` or ongoing query.
*   **`fieldIndex`**: Represents a B-tree based index for one or more document fields. It uses `github.com/google/btree` for efficient storage and retrieval of `indexEntry` items. It maintains `docRefs` (map of document IDs to `DocumentHandle`s) for each key, allowing quick access to matching documents. Each `fieldIndex` has its own RWMutex for concurrent access.
*   **`DocumentStream`**: Provides an iterator-like interface for consuming documents from the store. It's backed by Go channels and includes context-based cancellation, enabling efficient streaming of large result sets without loading all documents into memory at once.

### Data Flow

1.  **Insertion (`Insert`)**: A new document `ID` is generated, a `DocumentSnapshot` is created (versioned 1), and a `DocumentHandle` is initialized. This handle is added to the `Store`'s `documents` map. All active `fieldIndex`es are then updated to include the new document.
2.  **Update (`Update`)**: A new `DocumentSnapshot` with an incremented global version is created from the updated data. The `DocumentHandle`'s `atomic.Pointer` is atomically swapped to point to this new snapshot. The old snapshot's reference count is decremented. All `fieldIndex`es are updated to reflect potential changes in indexed fields.
3.  **Deletion (`Delete`)**: The `DocumentHandle` is removed from the `Store`'s `documents` map, and its internal snapshot pointer is set to `nil`. The final `DocumentSnapshot` is then passed to all `fieldIndex`es for removal, and its reference count is decremented.
4.  **Retrieval (`Get`, `Lookup`, `Stream`)**: When a document is read, its `DocumentHandle` is accessed. A `DocumentSnapshot` is retrieved, and its reference count is temporarily incremented (`read()` method) to prevent it from being garbage collected while in use. Once processed, the snapshot's reference count is decremented (`release()` method).


