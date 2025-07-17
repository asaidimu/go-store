// cursor.go
package gostore

import (
	"fmt"
	"slices"
	"sort"
)

// DocumentLike constrains types that can be used as documents
type DocumentLike interface {
	~map[string]any
}

// Cursor interface for bidirectional document iteration
type Cursor[T DocumentLike] interface {
	Next() (*T, bool /* has next */, error)
	Previous() (*T, bool /* has previous */, error)
	Advance(count int /* can be negative */) (*T, bool /* has next/previous depending on intended direction */, error)
	Reset() error     // For algorithms that need multiple passes
	Clone() Cursor[T] // For nested operations
	Count() int       // Maximum number of documents we can iterate over
	Close() error     // Clean up resources
}

// StoreCursor provides bidirectional iteration over documents in the store
type StoreCursor[T DocumentLike] struct {
	store      *Store
	collection *Collection
	handles    []*DocumentHandle // Snapshot of document handles
	position   int               // Current position in the handles slice
	closed     bool              // Whether the cursor has been closed
}

// Next returns the next document and advances the cursor by one position
func (sc *StoreCursor[T]) Next() (*T, bool, error) {
	if sc.closed {
		return nil, false, ErrStreamClosed
	}

	if sc.position >= len(sc.handles) {
		return nil, false, nil
	}

	doc, err := sc.getDocumentAt(sc.position)
	if err != nil {
		sc.position++
		return nil, false, err
	}

	sc.position++
	hasNext := sc.position < len(sc.handles)
	typedDoc := T(doc)
	return &typedDoc, hasNext, nil
}

// Next returns the next document and advances the cursor by one position
func (sc *StoreCursor[T]) Previous() (*T, bool, error) {
	return sc.Advance(-1)
}

// Advance moves the cursor by 'count' positions and returns the document at the new position.
// 'count' can be negative to move backward.
// If the new position is out of bounds, the cursor is clamped to the nearest valid position (first or last document).
func (sc *StoreCursor[T]) Advance(count int) (*T, bool, error) {
	if sc.closed {
		return nil, false, ErrStreamClosed
	}

	newPosition := sc.position + count

	if newPosition < 0 {
		newPosition = 0
	} else if newPosition >= len(sc.handles) {
		newPosition = len(sc.handles) - 1
	}

	if len(sc.handles) == 0 {
		return nil, false, nil
	}

	sc.position = newPosition
	doc, err := sc.getDocumentAt(sc.position)
	if err != nil {
		return nil, false, err
	}

	hasMore := (count > 0 && sc.position < len(sc.handles)-1) || (count < 0 && sc.position > 0)
	typedDoc := T(doc)
	return &typedDoc, hasMore, nil
}

// Reset moves the cursor back to the beginning of the stream.
func (sc *StoreCursor[T]) Reset() error {
	if sc.closed {
		return ErrStreamClosed
	}
	sc.position = 0
	return nil
}

// Clone creates a new cursor at the same position and with the same underlying snapshot
func (sc *StoreCursor[T]) Clone() Cursor[T] {
	if sc.closed {
		// A cloned cursor from a closed cursor should also be closed
		return &StoreCursor[T]{
			closed: true,
		}
	}
	return &StoreCursor[T]{
		store:      sc.store,
		collection: sc.collection,
		handles:    sc.handles,
		position:   sc.position,
		closed:     false,
	}
}

// Count returns the total number of documents in the cursor's snapshot.
func (sc *StoreCursor[T]) Count() int {
	if sc.closed || sc.handles == nil { // Handles case where handles slice is nilled out on close
		return 0
	}
	return len(sc.handles)
}

// Close releases any resources held by the cursor.
func (sc *StoreCursor[T]) Close() error {
	if sc.closed {
		return nil // Already closed
	}
	sc.closed = true
	// Release the snapshot to allow garbage collection
	sc.handles = nil
	sc.store = nil
	sc.collection = nil
	return nil
}

// getDocumentAt retrieves the document at a specific index from the collection.
// It handles cases where the document might have been deleted or doesn't exist.
func (sc *StoreCursor[T]) getDocumentAt(index int) (map[string]any, error) {
	if index < 0 || index >= len(sc.handles) {
		return nil, fmt.Errorf("index out of bounds: %d", index)
	}

	handle := sc.handles[index]
	doc, ok := sc.collection.Get(handle.index)
	if !ok {
		return nil, ErrDocumentDeleted
	}

	return doc.data, nil
}

// Read creates a cursor that iterates over all documents in the store
func (s *Store) Read() (*StoreCursor[map[string]any], error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Capture snapshot of all document handles
	handles := make([]*DocumentHandle, 0, len(s.handles))

	for _, entry := range s.handles {
		handles = append(handles, entry.handle)
	}

	sort.Slice(handles, func(i, j int) bool {
		return handles[i].index < handles[j].index
	})

	return &StoreCursor[map[string]any]{
		store:      s,
		collection: s.collection,
		handles:    handles,
		position:   0,
		closed:     false,
	}, nil
}

// ReadIndex creates a cursor that iterates over documents in index order
func (s *Store) ReadIndex(indexName string) (*StoreCursor[map[string]any], error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	defer s.mu.RUnlock() // Corrected: Changed Unlock to RUnlock

	_, exists := s.indexes[indexName]

	if !exists {
		return nil, ErrIndexNotFound
	}

	// Collect all document handles from the index in sorted order
	var handles []*DocumentHandle

	for _, entry := range s.handles { // Assuming s.handles is already sorted by internal index
		if slices.Contains(entry.indexes, indexName) {
			handles = append(handles, entry.handle) // Found in this index, move to next document entry
		}
	}

	return &StoreCursor[map[string]any]{
		store:      s,
		collection: s.collection,
		handles:    handles,
		position:   0,
		closed:     false,
	}, nil
}
