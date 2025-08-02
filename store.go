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

// Document represents a stable document in the collection
type Document struct {
	id      string
	data    map[string]any
	version uint64
	deleted bool
}

// Collection manages stable document storage with auto-scaling
type Collection struct {
	documents []*Document
	freeSlots []int // Indices of deleted documents available for reuse
	mu        sync.RWMutex
}

// NewCollection creates a new document collection
func NewCollection() *Collection {
	return &Collection{
		documents: make([]*Document, 0),
		freeSlots: make([]int, 0),
	}
}

// Insert adds a new document to the collection and returns its stable index
func (c *Collection) Insert(id string, data map[string]any, version uint64) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc := &Document{
		id:      id,
		data:    copyDocument(data),
		version: version,
		deleted: false,
	}

	// Reuse a free slot if available
	if len(c.freeSlots) > 0 {
		index := c.freeSlots[len(c.freeSlots)-1]
		c.freeSlots = c.freeSlots[:len(c.freeSlots)-1]
		c.documents[index] = doc
		return index
	}

	// Otherwise append to the end
	c.documents = append(c.documents, doc)
	return len(c.documents) - 1
}

// Update modifies an existing document in place
func (c *Collection) Update(index int, data map[string]any, version uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index < 0 || index >= len(c.documents) {
		return false
	}

	doc := c.documents[index]
	if doc == nil || doc.deleted {
		return false
	}

	// Update in place - this is the key optimization
	doc.data = copyDocument(data)
	doc.version = version
	return true
}

// Get retrieves a document by index
func (c *Collection) Get(index int) (*Document, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if index < 0 || index >= len(c.documents) {
		return nil, false
	}

	doc := c.documents[index]
	if doc == nil || doc.deleted {
		return nil, false
	}

	// Return a copy to maintain immutability for callers
	return &Document{
		id:      doc.id,
		data:    copyDocument(doc.data),
		version: doc.version,
		deleted: doc.deleted,
	}, true
}

// Delete marks a document as deleted and reclaims memory immediately
func (c *Collection) Delete(index int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index < 0 || index >= len(c.documents) {
		return false
	}

	doc := c.documents[index]
	if doc == nil || doc.deleted {
		return false
	}

	// Mark as deleted and clear data immediately
	doc.deleted = true
	doc.data = nil
	c.documents[index] = nil
	c.freeSlots = append(c.freeSlots, index)
	return true
}

// GetAllValid returns all non-deleted documents
func (c *Collection) GetAllValid() []*Document {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*Document
	for _, doc := range c.documents {
		if doc != nil && !doc.deleted {
			result = append(result, &Document{
				id:      doc.id,
				data:    copyDocument(doc.data),
				version: doc.version,
				deleted: doc.deleted,
			})
		}
	}
	return result
}

// DocumentHandle provides a versioned reference to a stable document location.
// It tracks the current version and provides atomic access to document state
// without requiring complex reference counting.
type DocumentHandle struct {
	id       string
	version  uint64
	index    int // Stable index in the collection
	document *Document
	mu       sync.RWMutex
}

// HandleEntry consolidates handle management with index membership tracking
type HandleEntry struct {
	handle  *DocumentHandle
	indexes []string
}

// indexKey represents a composite key for index entries.
type indexKey struct {
	values []any
}

// Less implements btree.Item interface for ordering index keys.
func (ik indexKey) Less(other btree.Item) bool {
	otherKey := other.(indexKey)

	// Compare values element by element
	minLen := min(len(otherKey.values), len(ik.values))

	for i := range minLen {
		if cmp := compareValues(ik.values[i], otherKey.values[i]); cmp != 0 {
			return cmp < 0
		}
	}

	// If all compared values are equal, shorter key comes first
	return len(ik.values) < len(otherKey.values)
}

// indexEntry stores a key and the set of document IDs that match it.
type indexEntry struct {
	key    indexKey
	docIDs map[string]struct{} // Changed from handles to document IDs
}

// Less implements btree.Item interface for ordering index entries.
func (ie indexEntry) Less(other btree.Item) bool {
	return ie.key.Less(other.(indexEntry).key)
}

// fieldIndex is a B-tree based index on one or more document fields.
type fieldIndex struct {
	name       string
	fields     []string
	tree       *btree.BTree
	collection *Collection // Reference to the stable collection
	mu         sync.RWMutex
}

// newFieldIndex creates a new field index with the specified name and fields.
func newFieldIndex(name string, fields []string, collection *Collection) *fieldIndex {
	return &fieldIndex{
		name:       name,
		fields:     fields,
		tree:       btree.New(32),
		collection: collection,
	}
}

// insertDocument adds a document to the index if it has values for all indexed fields.
func (fi *fieldIndex) insertDocument(handle *DocumentHandle) bool {
	doc, exists := fi.collection.Get(handle.index)
	if !exists {
		return false
	}

	keyValues := fi.extractKeyValues(doc.data)
	if keyValues == nil {
		return false // Document doesn't have all required fields
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.addToIndex(handle.id, keyValues)
	return true
}

// updateDocument updates a document's position in the index.
func (fi *fieldIndex) updateDocument(handle *DocumentHandle, oldData map[string]any) bool {
	doc, exists := fi.collection.Get(handle.index)
	if !exists {
		return false
	}

	oldKeyValues := fi.extractKeyValues(oldData)
	newKeyValues := fi.extractKeyValues(doc.data)

	// Optimization: if indexed fields haven't changed, no work needed
	if reflect.DeepEqual(oldKeyValues, newKeyValues) {
		return oldKeyValues != nil // Return true if document was/is indexed
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()

	// Remove old entry if it existed
	wasIndexed := false
	if oldKeyValues != nil {
		wasIndexed = true
		fi.removeFromIndex(handle.id, oldKeyValues)
	}

	// Add new entry if document has all required fields
	nowIndexed := false
	if newKeyValues != nil {
		nowIndexed = true
		fi.addToIndex(handle.id, newKeyValues)
	}

	return wasIndexed || nowIndexed
}

// deleteDocument removes a document from the index.
func (fi *fieldIndex) deleteDocument(docID string, data map[string]any) bool {
	keyValues := fi.extractKeyValues(data)
	if keyValues == nil {
		return false // Document wasn't indexed
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.removeFromIndex(docID, keyValues)
	return true
}

// removeFromIndex removes a document ID from an index entry.
func (fi *fieldIndex) removeFromIndex(docID string, keyValues []any) {
	searchEntry := indexEntry{key: indexKey{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(indexEntry)
		delete(entry.docIDs, docID)

		// Clean up empty entries
		if len(entry.docIDs) == 0 {
			fi.tree.Delete(searchEntry)
		}
	}
}

// addToIndex adds a document ID to an index entry.
func (fi *fieldIndex) addToIndex(docID string, keyValues []any) {
	searchEntry := indexEntry{key: indexKey{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		// Add to existing entry
		entry := item.(indexEntry)
		entry.docIDs[docID] = struct{}{}
	} else {
		// Create new entry
		entry := indexEntry{
			key:    indexKey{values: keyValues},
			docIDs: map[string]struct{}{docID:{}},
		}
		fi.tree.ReplaceOrInsert(entry)
	}
}

// extractKeyValues extracts the values for indexed fields from a document.
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

// lookup finds document IDs that exactly match the given values.
func (fi *fieldIndex) lookup(values []any) []string {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	searchEntry := indexEntry{key: indexKey{values: values}}
	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(indexEntry)
		result := make([]string, 0, len(entry.docIDs))
		for docID := range entry.docIDs {
			result = append(result, docID)
		}
		return result
	}

	return nil
}

// lookupRange finds document IDs within a given range of values.
func (fi *fieldIndex) lookupRange(minValues, maxValues []any) []string {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	var result []string
	minEntry := indexEntry{key: indexKey{values: minValues}}
	maxEntry := indexEntry{key: indexKey{values: maxValues}}

	fi.tree.AscendRange(minEntry, maxEntry, func(item btree.Item) bool {
		entry := item.(indexEntry)
		for docID := range entry.docIDs {
			result = append(result, docID)
		}
		return true // Continue iteration
	})

	return result
}

// DocumentResult holds the data and metadata for a document returned from a query.
type DocumentResult struct {
	ID      string
	Data    map[string]any
	Version uint64
}

// DocumentStream provides an iterator-like interface for streaming documents.
type DocumentStream struct {
	results chan DocumentResult
	errors  chan error
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDocumentStream creates a new document stream with the specified buffer size.
func NewDocumentStream(bufferSize int) *DocumentStream {
	ctx, cancel := context.WithCancel(context.Background())

	var results chan DocumentResult
	if bufferSize > 0 {
		results = make(chan DocumentResult, bufferSize)
	} else {
		results = make(chan DocumentResult)
	}

	err := make(chan error, 1)

	return &DocumentStream{
		ctx:     ctx,
		cancel:  cancel,
		results: results,
		errors:  err,
	}
}

// Next returns the next document from the stream.
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
type Store struct {
	collection *Collection
	handles    map[string]HandleEntry // Centralized handle management
	indexes    map[string]*fieldIndex // Maps index name to index
	mu         sync.RWMutex           // Protects handles and indexes maps
	version    uint64                 // Global version counter
	closed     atomic.Bool            // Indicates if store is closed
}

// NewStore creates a new, empty document store.
func NewStore() *Store {
	collection := NewCollection()
	return &Store{
		collection: collection,
		handles:    make(map[string]HandleEntry),
		indexes:    make(map[string]*fieldIndex),
	}
}

// Insert adds a new document to the store and updates all indexes.
func (s *Store) Insert(doc map[string]any) (string, error) {
	if s.closed.Load() {
		return "", ErrStoreClosed
	}

	if doc == nil {
		return "", ErrInvalidDocument
	}

	// Generate unique ID and version
	docID := uuid.Must(uuid.NewV7()).String()
	version := atomic.AddUint64(&s.version, 1)

	// Insert into collection to get stable index
	index := s.collection.Insert(docID, doc, version)

	// Create handle
	handle := &DocumentHandle{
		id:    docID,
		index: index,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create handle entry
	entry := HandleEntry{
		handle:  handle,
		indexes: make([]string, 0, len(s.indexes)),
	}

	// Update all indexes synchronously
	for idxName, idx := range s.indexes {
		if idx.insertDocument(handle) {
			entry.indexes = append(entry.indexes, idxName)
		}
	}

	// Add handle entry to store
	s.handles[docID] = entry

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

	entry, exists := s.handles[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	// Get old data for index updates
	currentDoc, exists := s.collection.Get(entry.handle.index)
	if !exists {
		return ErrDocumentDeleted
	}

	currentData := copyDocument(currentDoc.data)

	// Update in collection
	version := atomic.AddUint64(&s.version, 1)
	if !s.collection.Update(entry.handle.index, doc, version) {
		return ErrDocumentDeleted
	}

	// Update indexes and track new membership
	newIndexes := make([]string, 0, len(s.indexes))
	for idxName, idx := range s.indexes {
		if idx.updateDocument(entry.handle, currentData) {
			newIndexes = append(newIndexes, idxName)
		}
	}

	// Update handle entry with new index membership
	entry.indexes = newIndexes
	s.handles[docID] = entry

	return nil
}

// Delete removes a document from the store and all indexes.
func (s *Store) Delete(docID string) error {
	if s.closed.Load() {
		return ErrStoreClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.handles[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	// Get document data for index cleanup
	doc, exists := s.collection.Get(entry.handle.index)
	if !exists {
		return ErrDocumentDeleted
	}
	docData := copyDocument(doc.data)

	// Remove from only the indexes this document is actually in
	for _, indexName := range entry.indexes {
		if idx, exists := s.indexes[indexName]; exists {
			idx.deleteDocument(docID, docData)
		}
	}

	// Remove from collection and handles
	s.collection.Delete(entry.handle.index)
	delete(s.handles, docID)

	return nil
}

// Get retrieves a single document by its ID.
func (s *Store) Get(docID string) (*DocumentResult, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	entry, exists := s.handles[docID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrDocumentNotFound
	}

	doc, exists := s.collection.Get(entry.handle.index)
	if !exists {
		return nil, ErrDocumentDeleted
	}

	return &DocumentResult{
		ID:      docID,
		Data:    doc.data,
		Version: doc.version,
	}, nil
}

// Stream returns a stream of all documents currently in the store.
func (s *Store) Stream(bufferSize int) *DocumentStream {
	ds := NewDocumentStream(bufferSize)

	if s.closed.Load() {
		s.closeStreamWithError(ds, ErrStoreClosed)
		return ds
	}

	// Get all documents from collection
	documents := s.collection.GetAllValid()

	// Start streaming
	go s.streamDocuments(ds, documents)
	return ds
}

// Clone creates a deep copy of the store with all documents and indexes.
// The cloned store is completely independent - changes to one store will not affect the other.
// Returns an error if the store is closed.
func (s *Store) Clone() (*Store, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	// Lock the source store for reading during cloning
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create new store instance
	newStore := NewStore()

	// Set the version counter to match the source
	atomic.StoreUint64(&newStore.version, atomic.LoadUint64(&s.version))

	// Clone all valid documents
	documents := s.collection.GetAllValid()
	for _, doc := range documents {
		// Insert document into new store's collection
		index := newStore.collection.Insert(doc.id, copyDocument(doc.data), doc.version)

		// Create handle for the new store
		handle := &DocumentHandle{
			id:    doc.id,
			index: index,
		}

		// Create handle entry (indexes will be populated when we recreate indexes)
		entry := HandleEntry{
			handle:  handle,
			indexes: make([]string, 0),
		}

		newStore.handles[doc.id] = entry
	}

	// Recreate all indexes with the same configuration
	for indexName, sourceIndex := range s.indexes {
		// Create the index (this will automatically populate it with existing documents)
		err := newStore.CreateIndex(indexName, sourceIndex.fields)
		if err != nil {
			// This shouldn't happen since we're creating with unique names,
			// but handle it gracefully
			return nil, fmt.Errorf("failed to recreate index %s: %w", indexName, err)
		}
	}

	return newStore, nil
}

// CloneWithCallback creates a deep copy of the store with an optional callback
// that gets called for each document during cloning. This allows for selective
// cloning or document transformation during the clone operation.
// The callback receives the document and should return true to include it in the clone.
func (s *Store) CloneWithCallback(callback func(*DocumentResult) bool) (*Store, error) {
	if s.closed.Load() {
		return nil, ErrStoreClosed
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create new store instance
	newStore := NewStore()

	// Clone documents with callback filtering
	documents := s.collection.GetAllValid()
	for _, doc := range documents {
		docResult := &DocumentResult{
			ID:      doc.id,
			Data:    copyDocument(doc.data),
			Version: doc.version,
		}

		// Apply callback filter
		if callback != nil && !callback(docResult) {
			continue // Skip this document
		}

		// Insert document into new store's collection
		index := newStore.collection.Insert(doc.id, docResult.Data, doc.version)

		// Create handle for the new store
		handle := &DocumentHandle{
			id:    doc.id,
			index: index,
		}

		// Create handle entry
		entry := HandleEntry{
			handle:  handle,
			indexes: make([]string, 0),
		}

		newStore.handles[doc.id] = entry
	}

	// Recreate all indexes with the same configuration
	for indexName, sourceIndex := range s.indexes {
		err := newStore.CreateIndex(indexName, sourceIndex.fields)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate index %s: %w", indexName, err)
		}
	}

	// Set version counter based on what was actually cloned
	if len(newStore.handles) > 0 {
		atomic.StoreUint64(&newStore.version, atomic.LoadUint64(&s.version))
	}

	return newStore, nil
}

// streamDocuments runs the actual streaming logic in a goroutine.
func (s *Store) streamDocuments(ds *DocumentStream, documents []*Document) {
	defer close(ds.results)
	defer close(ds.errors)

	for _, doc := range documents {
		select {
		case <-ds.ctx.Done():
			return
		default:
			result := DocumentResult{
				ID:      doc.id,
				Data:    doc.data,
				Version: doc.version,
			}

			select {
			case ds.results <- result:
			case <-ds.ctx.Done():
				return
			}
		}
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
	index := newFieldIndex(indexName, fields, s.collection)
	s.indexes[indexName] = index

	// Populate with existing documents and update handle entries
	for docID, entry := range s.handles {
		if index.insertDocument(entry.handle) {
			// Update handle entry to include new index
			entry.indexes = append(entry.indexes, indexName)
			s.handles[docID] = entry
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

	// Remove index from all handle entries
	for docID, entry := range s.handles {
		newIndexes := make([]string, 0, len(entry.indexes))
		for _, idxName := range entry.indexes {
			if idxName != indexName {
				newIndexes = append(newIndexes, idxName)
			}
		}
		entry.indexes = newIndexes
		s.handles[docID] = entry
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
	docIDs := index.lookup(values)
	return s.collectDocumentResults(docIDs), nil
}

// lookupRangeWithIndex performs a range lookup using the specified index.
func (s *Store) lookupRangeWithIndex(index *fieldIndex, minValues, maxValues []any) ([]*DocumentResult, error) {
	docIDs := index.lookupRange(minValues, maxValues)
	return s.collectDocumentResults(docIDs), nil
}

// collectDocumentResults converts document IDs to results.
func (s *Store) collectDocumentResults(docIDs []string) []*DocumentResult {
	results := make([]*DocumentResult, 0, len(docIDs))

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, docID := range docIDs {
		if entry, exists := s.handles[docID]; exists {
			if doc, exists := s.collection.Get(entry.handle.index); exists {
				results = append(results, &DocumentResult{
					ID:      docID,
					Data:    doc.data,
					Version: doc.version,
				})
			}
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
	clear(s.handles)
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
