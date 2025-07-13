package gostore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/google/btree"
	"github.com/google/uuid"
)

// Custom error types for better error handling
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrDocumentDeleted  = errors.New("document has been deleted")
	ErrIndexExists      = errors.New("index already exists")
	ErrEmptyIndex       = errors.New("cannot create empty index")
	ErrIndexNotFound    = errors.New("index does not exist")
	ErrStreamClosed     = errors.New("stream closed")
	ErrStoreClosed      = errors.New("store closed")
	ErrInvalidDocument  = errors.New("invalid document")
)

// DocumentSnapshot is an immutable, versioned snapshot of a document's data.
// It uses reference counting to manage memory lifecycle safely across concurrent operations.
type DocumentSnapshot struct {
	data     map[string]any // The actual document data
	version  uint64   // Version number for optimistic concurrency control
	refCount int64    // Reference count for memory management
}

// release decrements the snapshot's reference count and clears data when no longer referenced.
// This helps prevent memory leaks in concurrent environments.
func (ds *DocumentSnapshot) release() {
	if atomic.AddInt64(&ds.refCount, -1) == 0 {
		// Clear the data map to help garbage collection
		clear(ds.data)
	}
}

// DocumentHandle provides thread-safe access to the current state of a document.
// It manages the lifecycle of document snapshots and ensures atomic updates.
type DocumentHandle struct {
	id      string                           // Unique document identifier
	current atomic.Pointer[DocumentSnapshot] // Current snapshot pointer
	mu      sync.RWMutex                     // Protects snapshot transitions
}

// read gets the current snapshot and increments its reference count.
// Returns nil if the document has been deleted.
func (dh *DocumentHandle) read() *DocumentSnapshot {
	if snap := dh.current.Load(); snap != nil {
		atomic.AddInt64(&snap.refCount, 1)
		return snap
	}
	return nil
}

// write replaces the current snapshot with a new one and returns the previous snapshot.
// The caller is responsible for releasing the returned snapshot.
func (dh *DocumentHandle) write(data map[string]any, version uint64) *DocumentSnapshot {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	old := dh.current.Load()
	newSnap := &DocumentSnapshot{
		data:     data,
		version:  version,
		refCount: 1, // Start with one reference (the handle itself)
	}
	dh.current.Store(newSnap)
	return old
}

// delete sets the current snapshot to nil and returns the previous snapshot.
// The caller is responsible for releasing the returned snapshot.
func (dh *DocumentHandle) delete() *DocumentSnapshot {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	old := dh.current.Load()
	dh.current.Store(nil)
	return old
}

// indexKey represents a composite key for index entries.
// It implements btree.Item for efficient B-tree operations.
type indexKey struct {
	values []any
}

// Less implements btree.Item interface for ordering index keys.
func (ik indexKey) Less(other btree.Item) bool {
	otherKey := other.(indexKey)

	// Compare values element by element
	minLen := len(ik.values)
	if len(otherKey.values) < minLen {
		minLen = len(otherKey.values)
	}

	for i := 0; i < minLen; i++ {
		if cmp := compareValues(ik.values[i], otherKey.values[i]); cmp != 0 {
			return cmp < 0
		}
	}

	// If all compared values are equal, shorter key comes first
	return len(ik.values) < len(otherKey.values)
}

// indexEntry stores a key and the set of documents that match it.
type indexEntry struct {
	key     indexKey
	docRefs map[string]*DocumentHandle // Maps document ID to handle
}

// Less implements btree.Item interface for ordering index entries.
func (ie indexEntry) Less(other btree.Item) bool {
	return ie.key.Less(other.(indexEntry).key)
}

// fieldIndex is a B-tree based index on one or more document fields.
// It maintains a sorted structure for efficient range queries and lookups.
type fieldIndex struct {
	name   string       // Index name
	fields []string     // Fields included in this index
	tree   *btree.BTree // B-tree for efficient range operations
	mu     sync.RWMutex // Protects concurrent access to the tree
}

// newFieldIndex creates a new field index with the specified name and fields.
func newFieldIndex(name string, fields []string) *fieldIndex {
	return &fieldIndex{
		name:   name,
		fields: fields,
		tree:   btree.New(32), // Degree of 32 provides good performance for most use cases
	}
}

// insertDocument adds a document to the index if it has values for all indexed fields.
func (fi *fieldIndex) insertDocument(docRef *DocumentHandle, doc *DocumentSnapshot) {
	keyValues := fi.extractKeyValues(doc.data)
	if keyValues == nil {
		return // Document doesn't have all required fields
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.addToIndex(docRef, keyValues)
}

// updateDocument updates a document's position in the index.
// It removes the old entry and adds a new one if the indexed fields changed.
func (fi *fieldIndex) updateDocument(docRef *DocumentHandle, oldDoc, newDoc *DocumentSnapshot) {
	oldKeyValues := fi.extractKeyValues(oldDoc.data)
	newKeyValues := fi.extractKeyValues(newDoc.data)

	// Optimization: if indexed fields haven't changed, no work needed
	if reflect.DeepEqual(oldKeyValues, newKeyValues) {
		return
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()

	// Remove old entry if it existed
	if oldKeyValues != nil {
		fi.removeFromIndex(docRef, oldKeyValues)
	}

	// Add new entry if document has all required fields
	if newKeyValues != nil {
		fi.addToIndex(docRef, newKeyValues)
	}
}

// deleteDocument removes a document from the index.
func (fi *fieldIndex) deleteDocument(docRef *DocumentHandle, doc *DocumentSnapshot) {
	keyValues := fi.extractKeyValues(doc.data)
	if keyValues == nil {
		return // Document wasn't indexed
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.removeFromIndex(docRef, keyValues)
}

// removeFromIndex removes a document reference from an index entry.
// If the entry becomes empty, it's removed from the tree.
func (fi *fieldIndex) removeFromIndex(docRef *DocumentHandle, keyValues []any) {
	searchEntry := indexEntry{key: indexKey{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(indexEntry)
		delete(entry.docRefs, docRef.id)

		// Clean up empty entries
		if len(entry.docRefs) == 0 {
			fi.tree.Delete(searchEntry)
		}
	}
}

// addToIndex adds a document reference to an index entry.
// Creates a new entry if one doesn't exist for the key.
func (fi *fieldIndex) addToIndex(docRef *DocumentHandle, keyValues []any) {
	searchEntry := indexEntry{key: indexKey{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		// Add to existing entry
		entry := item.(indexEntry)
		entry.docRefs[docRef.id] = docRef
	} else {
		// Create new entry
		entry := indexEntry{
			key:     indexKey{values: keyValues},
			docRefs: map[string]*DocumentHandle{docRef.id: docRef},
		}
		fi.tree.ReplaceOrInsert(entry)
	}
}

// extractKeyValues extracts the values for indexed fields from a document.
// Returns nil if any required field is missing or nil.
func (fi *fieldIndex) extractKeyValues(data map[string]any) []any {
	values := make([]any, 0, len(fi.fields))

	for _, field := range fi.fields {
		value, exists := data[field]
		if !exists || value == nil {
			return nil // Skip documents with missing or nil indexed fields
		}
		values = append(values, value)
	}

	return values
}

// lookup finds documents that exactly match the given values.
func (fi *fieldIndex) lookup(values []any) []*DocumentHandle {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	searchEntry := indexEntry{key: indexKey{values: values}}
	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(indexEntry)
		result := make([]*DocumentHandle, 0, len(entry.docRefs))
		for _, docRef := range entry.docRefs {
			result = append(result, docRef)
		}
		return result
	}

	return nil
}

// lookupRange finds documents within a given range of values (inclusive).
func (fi *fieldIndex) lookupRange(minValues, maxValues []any) []*DocumentHandle {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	var result []*DocumentHandle
	minEntry := indexEntry{key: indexKey{values: minValues}}
	maxEntry := indexEntry{key: indexKey{values: maxValues}}

	fi.tree.AscendRange(minEntry, maxEntry, func(item btree.Item) bool {
		entry := item.(indexEntry)
		for _, docRef := range entry.docRefs {
			result = append(result, docRef)
		}
		return true // Continue iteration
	})

	return result
}

// DocumentResult holds the data and metadata for a document returned from a query.
type DocumentResult struct {
	ID      string   // Document identifier
	Data    map[string]any // Document data (deep copy)
	Version uint64   // Document version
}

// DocumentStream provides an iterator-like interface for streaming documents.
// It supports cancellation and buffering for efficient processing of large result sets.
type DocumentStream struct {
	results chan DocumentResult
	errors  chan error
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDocumentStream creates a new document stream with the specified buffer size.
// A buffer size of 0 creates an unbuffered channel.
func NewDocumentStream(bufferSize int) *DocumentStream {
	ctx, cancel := context.WithCancel(context.Background())

	var results chan DocumentResult
	if bufferSize > 0 {
		results = make(chan DocumentResult, bufferSize)
	} else {
		results = make(chan DocumentResult)
	}

	err := make(chan error, 1) // Small buffer for errors

	return &DocumentStream{
		ctx:     ctx,
		cancel:  cancel,
		results: results,
		errors:  err,
	}
}

// Next returns the next document from the stream.
// It blocks until a document is available, an error occurs, or the stream is closed.
func (ds *DocumentStream) Next() (DocumentResult, error) {
	select {
	case result, ok := <-ds.results:
		if !ok {
			return DocumentResult{}, ErrStreamClosed
		}
		return result, nil

	case err, ok := <-ds.errors:
		if !ok {
			// Error channel closed, check if there are remaining results
			select {
			case result, ok := <-ds.results:
				if ok {
					return result, nil
				}
			default:
			}
			return DocumentResult{}, ErrStreamClosed
		}
		return DocumentResult{}, err

	case <-ds.ctx.Done():
		return DocumentResult{}, ds.ctx.Err()
	}
}

// Close cancels the stream and releases resources.
func (ds *DocumentStream) Close() {
	ds.cancel()
}

// Store is an in-memory document database with indexing capabilities.
// It provides ACID-like properties for document operations and supports
// concurrent access with proper synchronization.
type Store struct {
	documents map[string]*DocumentHandle // Maps document ID to handle
	indexes   map[string]*fieldIndex     // Maps index name to index
	mu        sync.RWMutex               // Protects maps and coordinates operations
	version   uint64                     // Global version counter
	closed    atomic.Bool                // Indicates if store is closed
}

// NewStore creates a new, empty document store.
func NewStore() *Store {
	return &Store{
		documents: make(map[string]*DocumentHandle),
		indexes:   make(map[string]*fieldIndex),
	}
}

// Insert adds a new document to the store and updates all indexes.
// Returns the generated document ID or an error.
func (s *Store) Insert(doc map[string]any) (string, error) {
	if s.closed.Load() {
		return "", ErrStoreClosed
	}

	if doc == nil {
		return "", ErrInvalidDocument
	}

	// Generate unique ID and prepare document
	docID := uuid.New().String()
	docData := copyDocument(doc)
	version := atomic.AddUint64(&s.version, 1)

	// Create document handle and initial snapshot
	docHandle := &DocumentHandle{id: docID}
	snapshot := &DocumentSnapshot{
		data:     docData,
		version:  version,
		refCount: 1,
	}
	docHandle.current.Store(snapshot)

	// Add to store and update indexes atomically
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents[docID] = docHandle

	// Update all indexes
	for _, idx := range s.indexes {
		idx.insertDocument(docHandle, snapshot)
	}

	return docID, nil
}

// Update modifies an existing document and updates all affected indexes.
func (s *Store) Update(docID string, doc map[string]any) error {
	if s.closed.Load() {
		return ErrStoreClosed
	}

	if doc == nil {
		return ErrInvalidDocument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	docHandle, exists := s.documents[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	// Atomically replace the document snapshot
	oldSnapshot := docHandle.write(copyDocument(doc), atomic.AddUint64(&s.version, 1))
	if oldSnapshot == nil {
		return ErrDocumentDeleted
	}

	// Update all indexes with old and new versions
	newSnapshot := docHandle.read()
	if newSnapshot != nil {
		for _, idx := range s.indexes {
			idx.updateDocument(docHandle, oldSnapshot, newSnapshot)
		}
		newSnapshot.release()
	}

	// Release the old snapshot
	oldSnapshot.release()

	return nil
}

// Delete removes a document from the store and all indexes.
func (s *Store) Delete(docID string) error {
	if s.closed.Load() {
		return ErrStoreClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	docHandle, exists := s.documents[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	// Remove from document map and get final snapshot
	delete(s.documents, docID)
	deletedSnapshot := docHandle.delete()

	if deletedSnapshot != nil {
		// Remove from all indexes
		for _, idx := range s.indexes {
			idx.deleteDocument(docHandle, deletedSnapshot)
		}

		// Release the final snapshot
		deletedSnapshot.release()
	}

	return nil
}

// Get retrieves a single document by its ID.
func (s *Store) Get(docID string) (*DocumentResult, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	docHandle, exists := s.documents[docID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrDocumentNotFound
	}

	snapshot := docHandle.read()
	if snapshot == nil {
		return nil, ErrDocumentDeleted
	}
	defer snapshot.release()

	return &DocumentResult{
		ID:      docID,
		Data:    copyDocument(snapshot.data),
		Version: snapshot.version,
	}, nil
}

// Stream returns a stream of all documents currently in the store.
// The stream provides a consistent snapshot of documents at the time it was created.
func (s *Store) Stream(bufferSize int) *DocumentStream {
    ds := NewDocumentStream(bufferSize)

    if s.closed.Load() {
        s.closeStreamWithError(ds, ErrStoreClosed)
        return ds
    }

    // Capture snapshot immediately, not in the goroutine
    var docHandles []*DocumentHandle
    s.mu.RLock()
    for _, docHandle := range s.documents {
        if docHandle != nil && docHandle.id != "" {
            docHandles = append(docHandles, docHandle)
        }
    }
    s.mu.RUnlock()

    // Start streaming with the captured snapshot
    go s.streamDocuments(ds, docHandles)
    return ds
}

// streamDocuments is a helper that runs the actual streaming logic in a goroutine.
func (s *Store) streamDocuments(ds *DocumentStream, docHandles []*DocumentHandle) {
	defer close(ds.results)
	defer close(ds.errors)

	// Stream documents
	for _, docHandle := range docHandles {
		select {
		case <-ds.ctx.Done():
			return
		default:
			if !s.streamSingleDocument(ds, docHandle) {
				return
			}
		}
	}
}

// streamSingleDocument streams a single document and returns false if streaming should stop.
func (s *Store) streamSingleDocument(ds *DocumentStream, docHandle *DocumentHandle) bool {
	snapshot := docHandle.read()
	if snapshot == nil {
		return true // Skip deleted documents
	}
	defer snapshot.release()

	result := DocumentResult{
		ID:      docHandle.id,
		Data:    copyDocument(snapshot.data),
		Version: snapshot.version,
	}

	select {
	case ds.results <- result:
		return true
	case <-ds.ctx.Done():
		return false
	}
}

// closeStreamWithError closes a stream with an error.
func (s *Store) closeStreamWithError(ds *DocumentStream, err error) {
	go func() {
		defer close(ds.results)
		defer close(ds.errors)
		ds.errors <- err
	}()
}

// CreateIndex builds a new index on the specified fields.
func (s *Store) CreateIndex(indexName string, fields []string) error {
	if s.closed.Load() {
		return ErrStoreClosed
	}

	if len(fields) == 0 {
		return ErrEmptyIndex
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.indexes[indexName]; exists {
		return ErrIndexExists
	}

	// Create new index
	index := newFieldIndex(indexName, fields)
	s.indexes[indexName] = index

	// Populate with existing documents
	for _, docHandle := range s.documents {
		if snapshot := docHandle.read(); snapshot != nil {
			index.insertDocument(docHandle, snapshot)
			snapshot.release()
		}
	}

	return nil
}

// DropIndex removes an existing index from the store.
func (s *Store) DropIndex(indexName string) error {
	if s.closed.Load() {
		return ErrStoreClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.indexes[indexName]; !exists {
		return ErrIndexNotFound
	}

	delete(s.indexes, indexName)
	return nil
}

// Lookup finds documents using an exact match on an index.
func (s *Store) Lookup(indexName string, values []any) ([]*DocumentResult, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	index, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrIndexNotFound
	}

	return s.lookupWithIndex(index, values)
}

// LookupRange finds documents within a range using an index.
func (s *Store) LookupRange(indexName string, minValues, maxValues []any) ([]*DocumentResult, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	index, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrIndexNotFound
	}

	return s.lookupRangeWithIndex(index, minValues, maxValues)
}

// lookupWithIndex performs an exact lookup using the specified index.
func (s *Store) lookupWithIndex(index *fieldIndex, values []any) ([]*DocumentResult, error) {
	docRefs := index.lookup(values)
	return s.collectDocumentResults(docRefs), nil
}

// lookupRangeWithIndex performs a range lookup using the specified index.
func (s *Store) lookupRangeWithIndex(index *fieldIndex, minValues, maxValues []any) ([]*DocumentResult, error) {
	docRefs := index.lookupRange(minValues, maxValues)
	return s.collectDocumentResults(docRefs), nil
}

// collectDocumentResults converts document handles to results.
func (s *Store) collectDocumentResults(docRefs []*DocumentHandle) []*DocumentResult {
	results := make([]*DocumentResult, 0, len(docRefs))

	for _, docRef := range docRefs {
		if snapshot := docRef.read(); snapshot != nil {
			results = append(results, &DocumentResult{
				ID:      docRef.id,
				Data:    copyDocument(snapshot.data),
				Version: snapshot.version,
			})
			snapshot.release()
		}
	}

	return results
}

// Close shuts down the store and releases all resources.
func (s *Store) Close() {
	s.closed.Store(true)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear maps to help garbage collection
	clear(s.documents)
	clear(s.indexes)
}

// copyDocument creates a deep copy of a document.
func copyDocument(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = copyValue(v)
	}
	return dst
}

// copyValue creates a deep copy of a value, handling nested structures.
func copyValue(src any) any {
	switch v := src.(type) {
	case map[string]any:
		return copyDocument(v)
	case []any:
		dst := make([]any, len(v))
		for i, elem := range v {
			dst[i] = copyValue(elem)
		}
		return dst
	case []int:
		dst := make([]int, len(v))
		copy(dst, v)
		return dst
	case []string:
		dst := make([]string, len(v))
		copy(dst, v)
		return dst
	default:
		// For primitive types, direct assignment is sufficient
		return v
	}
}

// compareValues compares two values for B-tree ordering.
// It handles different types consistently and provides stable sorting.
func compareValues(a, b any) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Handle numeric types
	if aIsNum, bIsNum := isNumber(a), isNumber(b); aIsNum && bIsNum {
		return compareNumbers(a, b)
	}

	// Handle same types
	if reflect.TypeOf(a) == reflect.TypeOf(b) {
		return compareSameType(a, b)
	}

	// Handle different types by comparing type names
	typeA, typeB := reflect.TypeOf(a).String(), reflect.TypeOf(b).String()
	if typeA < typeB {
		return -1
	} else if typeA > typeB {
		return 1
	}

	return 0
}

// compareNumbers compares two numeric values.
func compareNumbers(a, b any) int {
	valA := toFloat64(a)
	valB := toFloat64(b)

	if valA < valB {
		return -1
	} else if valA > valB {
		return 1
	}
	return 0
}

// compareSameType compares two values of the same type.
func compareSameType(a, b any) int {
	switch va := a.(type) {
	case string:
		vb := b.(string)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0

	case bool:
		vb := b.(bool)
		if va == vb {
			return 0
		}
		if va {
			return 1
		}
		return -1

	default:
		// Fallback to string comparison for other types
		strA := fmt.Sprintf("%v", a)
		strB := fmt.Sprintf("%v", b)
		if strA < strB {
			return -1
		} else if strA > strB {
			return 1
		}
		return 0
	}
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		return 0 // Should not happen if isNumber returned true
	}
}

// isNumber checks if a value is a numeric type.
func isNumber(v any) bool {
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return true
	default:
		return false
	}
}
