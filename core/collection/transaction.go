package collection

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTransactionClosed   = errors.New("the transaction has been closed")
	ErrReadOnlyTransaction = errors.New("the transaction was opened for read only")
	ErrInvalidData         = errors.New("cannot insert invalid data into collection")
	ErrDocumentNotFound    = errors.New("document not found")
	ErrDocumentDeleted     = errors.New("document has been deleted")
	ErrStreamClosed        = errors.New("stream closed")
)

type TransactionMode int

const (
	TxR TransactionMode = iota
	TxRW
)

type write struct {
	id     string
	delete bool
	doc    document
}

type document struct {
	createdTxnID uint64
	data         map[string]any
}

type Transaction struct {
	TxnID      uint64
	TxnTime    int64 // Snapshot time for this transaction
	closed     bool
	writes     []*write
	collection *Collection
	mode       TransactionMode
}

// StartTransaction creates a new transaction with MVCC support
func (c *Collection) StartTransaction(mode TransactionMode) *Transaction {
	return c.startTransaction(mode)
}

// Commit applies all writes and commits the transaction
func (t *Transaction) Commit() error {
	if t.closed {
		return ErrTransactionClosed
	}

	defer t.close()

	if t.mode == TxR {
		return nil // Read-only transactions don't need to commit anything
	}

	// Apply all writes atomically with current commit time
	commitTime := time.Now().UnixNano()
	return t.collection.write(t.writes, commitTime)
}

// Rollback discards all changes and closes the transaction
func (t *Transaction) Rollback() {
	if t.closed {
		return
	}

	t.close()
}

// close cleans up the transaction
func (t *Transaction) close() {
	if t.closed {
		return
	}

	t.collection.endTransaction(t.TxnID)
	t.writes = nil
	t.closed = true
}

// Create adds a new document to the transaction's write set
func (t *Transaction) Create(data map[string]any) (string, error) {
	if t.closed {
		return "", ErrTransactionClosed
	}

	if t.mode != TxRW {
		return "", ErrReadOnlyTransaction
	}

	if data == nil {
		return "", ErrInvalidData
	}

	id := uuid.Must(uuid.NewV7()).String()

	entry := write{
		id:     id,
		delete: false,
		doc: document{
			data:         data,
			createdTxnID: t.TxnID,
		},
	}

	t.writes = append(t.writes, &entry)
	return id, nil
}

// Update modifies an existing document in the transaction's write set
func (t *Transaction) Update(id string, data map[string]any) error {
	if t.closed {
		return ErrTransactionClosed
	}

	if t.mode != TxRW {
		return ErrReadOnlyTransaction
	}

	if data == nil {
		return ErrInvalidData
	}

	// Check if document exists in current snapshot
	_, err := t.collection.read(id, t.TxnID, t.TxnTime)
	if err != nil {
		return err
	}

	entry := write{
		id:     id,
		delete: false,
		doc: document{
			data:         data,
			createdTxnID: t.TxnID,
		},
	}

	t.writes = append(t.writes, &entry)
	return nil
}

// Delete marks a document for deletion in the transaction's write set
func (t *Transaction) Delete(id string) error {
	if t.closed {
		return ErrTransactionClosed
	}

	if t.mode != TxRW {
		return ErrReadOnlyTransaction
	}

	// Check if document exists in current snapshot
	_, err := t.collection.read(id, t.TxnID, t.TxnTime)
	if err != nil {
		return err
	}

	entry := write{
		id:     id,
		delete: true,
		doc: document{
			createdTxnID: t.TxnID,
		},
	}

	t.writes = append(t.writes, &entry)
	return nil
}

// Read returns the visible version of a document for this transaction
func (t *Transaction) Read(id string) (map[string]any, error) {
	if t.closed {
		return map[string]any{}, ErrTransactionClosed
	}

	// First check if we have a pending write for this document in our transaction
	for i := len(t.writes) - 1; i >= 0; i-- {
		if t.writes[i].id == id {
			if t.writes[i].delete {
				return map[string]any{}, ErrDocumentDeleted
			}
			return t.writes[i].doc.data, nil
		}
	}

	// Read from the collection using our snapshot time
	return t.collection.read(id, t.TxnID, t.TxnTime)
}

// ReadAll returns all visible documents for this transaction
func (t *Transaction) ReadAll() (map[string]map[string]any, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	t.collection.mu.RLock()
	defer t.collection.mu.RUnlock()

	result := make(map[string]map[string]any)

	// Read all documents that are visible to this transaction
	for id, entry := range t.collection.documents {
		entry.mu.RLock()

		// Walk the version chain to find the visible version
		for version := entry.versions; version != nil; version = version.next {
			if t.collection.isVersionVisible(version, t.TxnID, t.TxnTime) {
				if !version.deleted {
					result[id] = version.data
				}
				break
			}
		}

		entry.mu.RUnlock()
	}

	// Apply any pending writes from this transaction
	for _, write := range t.writes {
		if write.delete {
			delete(result, write.id)
		} else {
			result[write.id] = write.doc.data
		}
	}

	return result, nil
}

// Exists checks if a document exists and is visible to this transaction
func (t *Transaction) Exists(id string) (bool, error) {
	if t.closed {
		return false, ErrTransactionClosed
	}

	_, err := t.Read(id)
	if err == ErrDocumentNotFound || err == ErrDocumentDeleted {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// FindByIndex queries documents using an index with exact match
func (t *Transaction) FindByIndex(indexName string, values []any) (map[string]map[string]any, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	t.collection.mu.RLock()
	index, exists := t.collection.indexes[indexName]
	t.collection.mu.RUnlock()

	if !exists {
		return nil, errors.New("index not found")
	}

	// Get candidate document IDs from index
	docIDs := index.lookup(values)

	// Filter by visibility and collect results
	result := make(map[string]map[string]any)
	for _, docID := range docIDs {
		if doc, err := t.Read(docID); err == nil {
			result[docID] = doc
		}
	}

	return result, nil
}

// FindByIndexRange queries documents using an index with range queries
func (t *Transaction) FindByIndexRange(indexName string, minValues, maxValues []any) (map[string]map[string]any, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	t.collection.mu.RLock()
	index, exists := t.collection.indexes[indexName]
	t.collection.mu.RUnlock()

	if !exists {
		return nil, errors.New("index not found")
	}

	// Get candidate document IDs from index
	docIDs := index.lookupRange(minValues, maxValues)

	// Filter by visibility and collect results
	result := make(map[string]map[string]any)
	for _, docID := range docIDs {
		if doc, err := t.Read(docID); err == nil {
			result[docID] = doc
		}
	}

	return result, nil
}

// FindByIndexPrefix queries documents using an index with prefix matching
func (t *Transaction) FindByIndexPrefix(indexName string, prefixValues []any) (map[string]map[string]any, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	t.collection.mu.RLock()
	index, exists := t.collection.indexes[indexName]
	t.collection.mu.RUnlock()

	if !exists {
		return nil, errors.New("index not found")
	}

	// Get candidate document IDs from index
	docIDs := index.lookupPrefix(prefixValues)

	// Filter by visibility and collect results
	result := make(map[string]map[string]any)
	for _, docID := range docIDs {
		if doc, err := t.Read(docID); err == nil {
			result[docID] = doc
		}
	}

	return result, nil
}

// Count returns the number of documents visible to this transaction
func (t *Transaction) Count() (int, error) {
	if t.closed {
		return 0, ErrTransactionClosed
	}

	allDocs, err := t.ReadAll()
	if err != nil {
		return 0, err
	}

	return len(allDocs), nil
}

// CountByIndex returns the number of documents matching an index query
func (t *Transaction) CountByIndex(indexName string, values []any) (int, error) {
	docs, err := t.FindByIndex(indexName, values)
	if err != nil {
		return 0, err
	}
	return len(docs), nil
}
