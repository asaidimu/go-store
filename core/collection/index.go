package collection

import (
	"reflect"
	"sync"

	"github.com/asaidimu/go-store/v3/core/utils"
	"github.com/google/btree"
)

// indexKey represents a composite key for key entries.
type key struct {
	values []any
}

// Less implements btree.Item interface for ordering index keys.
func (k key) Less(other btree.Item) bool {
	otherKey := other.(key)

	// Compare values element by element
	minLen := min(len(otherKey.values), len(k.values))

	for i := range minLen {
		if cmp := utils.CompareValues(k.values[i], otherKey.values[i]); cmp != 0 {
			return cmp < 0
		}
	}

	// If all compared values are equal, shorter key comes first
	return len(k.values) < len(otherKey.values)
}

// index stores a key and the set of document IDs that match it.
type index struct {
	key    key
	docIDs map[string]struct{} // Document IDs that match this key
}

// Less implements btree.Item interface for ordering index entries.
func (ie index) Less(other btree.Item) bool {
	return ie.key.Less(other.(index).key)
}

// fieldIndex is a B-tree based index on one or more document fields.
type fieldIndex struct {
	name   string
	fields []string
	tree   *btree.BTree
	mu     sync.RWMutex
}

// newFieldIndex creates a new field index with the specified name and fields.
func newFieldIndex(name string, fields []string) *fieldIndex {
	return &fieldIndex{
		name:   name,
		fields: fields,
		tree:   btree.New(32),
	}
}

// insertDocument adds a document to the index if it has values for all indexed fields.
func (fi *fieldIndex) insertDocument(id string, doc map[string]any) bool {
	keyValues := fi.extractKeyValues(doc)
	if keyValues == nil {
		return false
	}
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.addToIndex(id, keyValues)
	return true
}

// updateDocument updates a document's position in the index.
func (fi *fieldIndex) updateDocument(id string, prev, cur map[string]any) bool {
	oldKeyValues := fi.extractKeyValues(prev)
	newKeyValues := fi.extractKeyValues(cur)

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
		fi.removeFromIndex(id, oldKeyValues)
	}

	// Add new entry if document has all required fields
	nowIndexed := false
	if newKeyValues != nil {
		nowIndexed = true
		fi.addToIndex(id, newKeyValues)
	}

	return wasIndexed || nowIndexed
}

// deleteDocument removes a document from the index.
func (fi *fieldIndex) deleteDocument(docID string, doc map[string]any) bool {
	keyValues := fi.extractKeyValues(doc)
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
	searchEntry := index{key: key{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(index)
		delete(entry.docIDs, docID)

		// Clean up empty entries
		if len(entry.docIDs) == 0 {
			fi.tree.Delete(searchEntry)
		}
	}
}

// addToIndex adds a document ID to an index entry.
func (fi *fieldIndex) addToIndex(docID string, keyValues []any) {
	searchEntry := index{key: key{values: keyValues}}

	if item := fi.tree.Get(searchEntry); item != nil {
		// Add to existing entry
		entry := item.(index)
		entry.docIDs[docID] = struct{}{}
	} else {
		// Create new entry
		entry := index{
			key:    key{values: keyValues},
			docIDs: map[string]struct{}{docID: {}},
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

	searchEntry := index{key: key{values: values}}
	if item := fi.tree.Get(searchEntry); item != nil {
		entry := item.(index)
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
	minEntry := index{key: key{values: minValues}}
	maxEntry := index{key: key{values: maxValues}}

	fi.tree.AscendRange(minEntry, maxEntry, func(item btree.Item) bool {
		entry := item.(index)
		for docID := range entry.docIDs {
			result = append(result, docID)
		}
		return true
	})
	return result
}

// lookupPrefix finds document IDs where the indexed fields start with the given values.
func (fi *fieldIndex) lookupPrefix(prefixValues []any) []string {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	var result []string
	prefixEntry := index{key: key{values: prefixValues}}

	fi.tree.AscendGreaterOrEqual(prefixEntry, func(item btree.Item) bool {
		entry := item.(index)

		// Check if this entry matches the prefix
		if len(entry.key.values) >= len(prefixValues) {
			matches := true
			for i, prefixValue := range prefixValues {
				if utils.CompareValues(entry.key.values[i], prefixValue) != 0 {
					matches = false
					break
				}
			}

			if matches {
				for docID := range entry.docIDs {
					result = append(result, docID)
				}
				return true // Continue iteration
			}
		}

		// Stop if we've gone beyond the prefix
		return false
	})

	return result
}

// GetStats returns statistics about the index
func (fi *fieldIndex) GetStats() map[string]any {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	totalDocs := 0
	fi.tree.Ascend(func(item btree.Item) bool {
		entry := item.(index)
		totalDocs += len(entry.docIDs)
		return true
	})

	return map[string]any{
		"name":         fi.name,
		"fields":       fi.fields,
		"entries":      fi.tree.Len(),
		"totalDocs":    totalDocs,
		"avgDocsPerEntry": float64(totalDocs) / float64(max(fi.tree.Len(), 1)),
	}
}
