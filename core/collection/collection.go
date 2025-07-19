package collection

import (
	"errors"
	"sync"
	"time"

	"github.com/asaidimu/go-store/v3/core/utils"
)

const maxTxnID uint64 = ^uint64(0)

// documentVersion represents a single version of a document
type documentVersion struct {
	createdTxnID  uint64
	committedTime int64  // When this version was committed (0 if not committed)
	data          map[string]any
	deleted       bool
	next          *documentVersion // Pointer to previous version (version chain)
}

// documentEntry represents the head of a version chain for a document
type documentEntry struct {
	id       string
	versions *documentVersion // Head of version chain (newest first)
	mu       sync.RWMutex
}

// activeTransaction tracks information about active transactions
type activeTransaction struct {
	txnID     uint64
	startTime int64
	mode      TransactionMode
}

// Collection manages stable document storage with MVCC
type Collection struct {
	documents    map[string]*documentEntry
	indexes      map[string]*fieldIndex // Maps index name to index
	currentTxnID uint64
	mu           sync.RWMutex

	// MVCC specific fields
	activeTxns   map[uint64]*activeTransaction // Track active transactions
	txmu         sync.RWMutex                   // Protects activeTxns and currentTxnID
	gcMu         sync.Mutex                     // Protects garbage collection
	lastGCTime   int64                          // Last garbage collection time
}

// NewCollection creates a new document collection with MVCC support
func NewCollection() *Collection {
	return &Collection{
		documents:    make(map[string]*documentEntry),
		indexes:      map[string]*fieldIndex{},
		activeTxns:   make(map[uint64]*activeTransaction),
		currentTxnID: 0,
		lastGCTime:   time.Now().UnixNano(),
	}
}

// read returns the visible version of a document for a given transaction
func (c *Collection) read(id string, txnID uint64, snapshotTime int64) (map[string]any, error) {
	c.mu.RLock()
	entry, exists := c.documents[id]
	c.mu.RUnlock()

	if !exists {
		return map[string]any{}, ErrDocumentNotFound
	}

	entry.mu.RLock()
	defer entry.mu.RUnlock()

	// Walk the version chain to find the visible version
	for version := entry.versions; version != nil; version = version.next {
		if c.isVersionVisible(version, txnID, snapshotTime) {
			if version.deleted {
				return map[string]any{}, ErrDocumentDeleted
			}
			return utils.CopyDocument(version.data), nil
		}
	}

	return map[string]any{}, ErrDocumentNotFound
}

// isVersionVisible determines if a version is visible to a transaction
func (c *Collection) isVersionVisible(version *documentVersion, txnID uint64, snapshotTime int64) bool {
	// If this version was created by the current transaction, it's visible
	if version.createdTxnID == txnID {
		return true
	}

	// If the version is not yet committed, it's not visible to other transactions
	if version.committedTime == 0 {
		return false
	}

	// Version is visible if it was committed before the transaction's snapshot time
	return version.committedTime < snapshotTime
}

// write applies a batch of writes atomically
func (c *Collection) write(writes []*write, commitTime int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, w := range writes {
		if err := c.applyWrite(w, commitTime); err != nil {
			return err
		}
	}

	return nil
}

// applyWrite applies a single write operation
func (c *Collection) applyWrite(w *write, commitTime int64) error {
	entry, exists := c.documents[w.id]

	if !exists {
		if w.delete {
			return ErrDocumentNotFound
		}

		// Create new document entry
		newVersion := &documentVersion{
			createdTxnID:  w.doc.createdTxnID,
			committedTime: commitTime,
			data:          w.doc.data,
			deleted:       false,
			next:          nil,
		}

		entry = &documentEntry{
			id:       w.id,
			versions: newVersion,
		}
		c.documents[w.id] = entry

		// Update indexes for new document
		c.updateIndexesForNewVersion(w.id, nil, newVersion)
		return nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Get the previous version for index updates
	prevVersion := entry.versions

	// Create new version and add to front of chain
	newVersion := &documentVersion{
		createdTxnID:  w.doc.createdTxnID,
		committedTime: commitTime,
		data:          w.doc.data,
		deleted:       w.delete,
		next:          entry.versions,
	}

	entry.versions = newVersion

	// Update indexes with the new version
	c.updateIndexesForNewVersion(w.id, prevVersion, newVersion)
	return nil
}

// startTransaction creates a new transaction and tracks it
func (c *Collection) startTransaction(mode TransactionMode) *Transaction {
	c.txmu.Lock()
	defer c.txmu.Unlock()

	c.currentTxnID++
	txnID := c.currentTxnID
	startTime := time.Now().UnixNano()

	// Track active transaction
	c.activeTxns[txnID] = &activeTransaction{
		txnID:     txnID,
		startTime: startTime,
		mode:      mode,
	}

	return &Transaction{
		TxnID:      txnID,
		TxnTime:    startTime,
		closed:     false,
		writes:     make([]*write, 0),
		collection: c,
		mode:       mode,
	}
}

// endTransaction removes transaction from active tracking
func (c *Collection) endTransaction(txnID uint64) {
	c.txmu.Lock()
	delete(c.activeTxns, txnID)
	c.txmu.Unlock()

	// Trigger garbage collection periodically
	c.maybeGarbageCollect()
}

// maybeGarbageCollect runs garbage collection if enough time has passed
func (c *Collection) maybeGarbageCollect() {
	now := time.Now().UnixNano()
	if now-c.lastGCTime > 10*time.Second.Nanoseconds() {
		go c.garbageCollect()
	}
}

// garbageCollect removes old versions that are no longer visible to any transaction
func (c *Collection) garbageCollect() {
	c.gcMu.Lock()
	defer c.gcMu.Unlock()

	now := time.Now().UnixNano()
	c.lastGCTime = now

	// Find the oldest active transaction snapshot time
	c.txmu.RLock()
	oldestSnapshotTime := now
	for _, txn := range c.activeTxns {
		if txn.startTime < oldestSnapshotTime {
			oldestSnapshotTime = txn.startTime
		}
	}
	c.txmu.RUnlock()

	// Clean up old versions
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range c.documents {
		entry.mu.Lock()
		c.cleanupVersionChain(entry, oldestSnapshotTime)
		entry.mu.Unlock()
	}
}

// cleanupVersionChain removes versions that are no longer visible
func (c *Collection) cleanupVersionChain(entry *documentEntry, oldestSnapshotTime int64) {
	if entry.versions == nil {
		return
	}

	// Keep at least one version and any versions that might still be visible
	var _ *documentVersion
	current := entry.versions

	for current != nil && current.next != nil {
		// If the next version is older than the oldest active transaction
		// and we have at least one version to keep, we can remove it
		if current.next.committedTime > 0 && current.next.committedTime < oldestSnapshotTime {
			// Clean up indexes for the versions we're about to remove
			c.cleanupIndexesForOldVersions(entry.id, current.next)
			// Remove the rest of the chain
			current.next = nil
			break
		}
		_ = current
		current = current.next
	}
}

// updateIndexesForNewVersion updates all indexes when a new version is created
func (c *Collection) updateIndexesForNewVersion(docID string, prevVersion, newVersion *documentVersion) {
	for _, index := range c.indexes {
		var prevData map[string]any
		if prevVersion != nil && !prevVersion.deleted {
			prevData = prevVersion.data
		}

		if newVersion.deleted {
			// Document is being deleted
			if prevData != nil {
				index.deleteDocument(docID, prevData)
			}
		} else if prevData != nil {
			// Document is being updated
			index.updateDocument(docID, prevData, newVersion.data)
		} else {
			// Document is being created
			index.insertDocument(docID, newVersion.data)
		}
	}
}

// cleanupIndexesForOldVersions removes index entries for old versions being garbage collected
func (c *Collection) cleanupIndexesForOldVersions(docID string, versionChain *documentVersion) {
	// This is a simplified cleanup - in a production system you'd want more sophisticated
	// tracking of which versions are indexed to avoid redundant cleanup
	for version := versionChain; version != nil; version = version.next {
		if !version.deleted {
			for _, index := range c.indexes {
				index.deleteDocument(docID, version.data)
			}
		}
	}
}

// CreateIndex creates a new index on the specified fields
func (c *Collection) CreateIndex(name string, fields []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.indexes[name]; exists {
		return errors.New("index already exists")
	}

	index := newFieldIndex(name, fields)
	c.indexes[name] = index

	// Build the index with current visible documents
	// For simplicity, we'll index the latest committed version of each document
	for _, entry := range c.documents {
		entry.mu.RLock()
		// Find the latest committed version
		for version := entry.versions; version != nil; version = version.next {
			if version.committedTime > 0 && !version.deleted {
				index.insertDocument(entry.id, version.data)
				break
			}
		}
		entry.mu.RUnlock()
	}

	return nil
}

// DropIndex removes an index
func (c *Collection) DropIndex(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.indexes[name]; !exists {
		return errors.New("index does not exist")
	}

	delete(c.indexes, name)
	return nil
}

// GetIndexes returns the names of all indexes
func (c *Collection) GetIndexes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.indexes))
	for name := range c.indexes {
		names = append(names, name)
	}
	return names
}
