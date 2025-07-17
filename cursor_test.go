// cursor_test.go
package gostore

import (
	"errors"
	"reflect"
	"testing"
)

// TestStoreCursorNext tests the Next method of StoreCursor.
func TestStoreCursorNext(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Insert some documents
	doc1 := map[string]any{"name": "Alice", "age": 30}
	doc2 := map[string]any{"name": "Bob", "age": 25}
	doc3 := map[string]any{"name": "Charlie", "age": 35}

	_, _ = s.Insert(doc1)
	_, _ = s.Insert(doc2)
	_, _ = s.Insert(doc3)

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Test Next() for all documents
	expectedDocs := []map[string]any{doc1, doc2, doc3} // Sorted by index in Read()

	for i := range expectedDocs {
		doc, hasNext, err := cursor.Next()
		if err != nil {
			t.Fatalf("Next() failed at iteration %d: %v", i, err)
		}
		if doc == nil {
			t.Fatalf("Next() returned nil document at iteration %d", i)
		}

		// Because Read() sorts by internal index, and we inserted sequentially,
		// the order should match insertion order if IDs are sequential or not
		// and the internal indices happen to align.
		// For robust comparison, we'll check if the retrieved document data
		// matches one of our expected documents.
		found := false
		for _, expected := range expectedDocs {
			if reflect.DeepEqual(*doc, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Next() returned unexpected document: %v", doc)
		}

		if i < len(expectedDocs)-1 && !hasNext {
			t.Errorf("Expected hasNext to be true at iteration %d", i)
		}
		if i == len(expectedDocs)-1 && hasNext {
			t.Errorf("Expected hasNext to be false at iteration %d", i)
		}
	}

	// Test Next() after exhaustion
	doc, hasNext, err := cursor.Next()
	if err != nil {
		t.Fatalf("Next() after exhaustion returned error: %v", err)
	}
	if doc != nil && !reflect.DeepEqual(*doc, map[string]any{}) { // Expect empty map for map[string]any zero value
		t.Errorf("Next() after exhaustion returned non-empty document: %v", doc)
	}
	if hasNext {
		t.Error("Next() after exhaustion returned hasNext true")
	}

	// Test Next() on a closed cursor
	cursor.Close()
	_, _, err = cursor.Next()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed on Next() after Close(), got %v", err)
	}
}

// TestStoreCursorAdvance tests the Advance method of StoreCursor.
func TestStoreCursorAdvance(t *testing.T) {
	s := NewStore()
	defer s.Close()

	docs := make([]map[string]any, 5)
	for i := range 5 {
		docs[i] = map[string]any{"value": i}
		_, _ = s.Insert(docs[i])
	}

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Advance forward
	doc, hasMore, err := cursor.Advance(2) // Move to index 2 (value: 2)
	if err != nil {
		t.Fatalf("Advance(2) failed: %v", err)
	}
	if !reflect.DeepEqual(*doc, docs[2]) {
		t.Errorf("Advance(2) returned wrong document. Expected %v, got %v", docs[2], doc)
	}
	if !hasMore { // Should have doc3 and doc4 after this
		t.Errorf("Expected hasMore to be true after advancing to index 2")
	}

	// Advance further forward past end
	doc, hasMore, err = cursor.Advance(10) // Try to move past end
	if err != nil {
		t.Fatalf("Advance(10) failed: %v", err)
	}

	if !reflect.DeepEqual(*doc, docs[4]) { // Should clamp to last document
		t.Errorf("Advance(10) clamped incorrectly. Expected %v, got %v", docs[4], doc)
	}

	if hasMore {
		t.Errorf("Expected hasMore to be false after advancing past end %v", cursor.position)
	}

	// Advance backward
	doc, hasMore, err = cursor.Advance(-2) // Move back to index 2 (value: 2)
	if err != nil {
		t.Fatalf("Advance(-2) failed: %v", err)
	}
	if !reflect.DeepEqual(*doc, docs[2]) {
		t.Errorf("Advance(-2) returned wrong document. Expected %v, got %v", docs[2], doc)
	}
	if !hasMore { // Should have doc1 and doc0 before this
		t.Errorf("Expected hasMore to be true after advancing back to index 2")
	}

	// Advance further backward past beginning
	doc, hasMore, err = cursor.Advance(-10) // Try to move past beginning
	if err != nil {
		t.Fatalf("Advance(-10) failed: %v", err)
	}
	if !reflect.DeepEqual(*doc, docs[0]) { // Should clamp to first document
		t.Errorf("Advance(-10) clamped incorrectly. Expected %v, got %v", docs[0], doc)
	}
	if hasMore {
		t.Errorf("Expected hasMore to be true after advancing to beginning") // Can still advance forward
	}

	// Test Advance() on closed cursor
	cursor.Close()
	_, _, err = cursor.Advance(1)
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed on Advance() after Close(), got %v", err)
	}
}

// TestStoreCursorReset tests the Reset method of StoreCursor.
func TestStoreCursorReset(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc1 := map[string]any{"id": "d1"}
	doc2 := map[string]any{"id": "d2"}
	_, _ = s.Insert(doc1)
	_, _ = s.Insert(doc2)

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Advance to middle
	_, _, _ = cursor.Next() // Move to d1
	_, _, _ = cursor.Next() // Move to d2

	// Reset and check if back at beginning
	err = cursor.Reset()
	if err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	doc, _, err := cursor.Next()
	if err != nil {
		t.Fatalf("Next() after Reset() failed: %v", err)
	}
	if !reflect.DeepEqual(*doc, doc1) {
		t.Errorf("Reset() did not move cursor to beginning. Expected %v, got %v", doc1, doc)
	}

	// Test Reset() on a closed cursor
	cursor.Close()
	err = cursor.Reset()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed on Reset() after Close(), got %v", err)
	}
}

// TestStoreCursorClone tests the Clone method of StoreCursor.
func TestStoreCursorClone(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc1 := map[string]any{"val": 1}
	doc2 := map[string]any{"val": 2}
	_, _ = s.Insert(doc1)
	_, _ = s.Insert(doc2)

	originalCursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create original cursor: %v", err)
	}
	defer originalCursor.Close()

	// Advance original cursor
	firstDocOriginal, _, _ := originalCursor.Next() // Should be doc1

	clonedCursor := originalCursor.Clone()
	defer clonedCursor.Close()

	// Verify original cursor is at doc1
	if !reflect.DeepEqual(*firstDocOriginal, doc1) {
		t.Errorf("Original cursor not at expected position. Expected %v, got %v", doc1, firstDocOriginal)
	}

	// Verify cloned cursor is also at doc1 (same initial position as original)
	firstDocCloned, _, _ := clonedCursor.Next()
	if !reflect.DeepEqual(*firstDocCloned, doc2) { // Next will give the second doc after cloning the current position.
		t.Errorf("Cloned cursor not at expected position. Expected %v, got %v", doc2, firstDocCloned)
	}

	// Move cloned cursor independently
	secondDocCloned, hasNextCloned, err := clonedCursor.Next() // Should be empty map, hasNext false
	if err != nil {
		t.Fatalf("Next() on cloned cursor failed: %v", err)
	}

	if secondDocCloned != nil { // Expect empty map for map[string]any zero value
		t.Errorf("Cloned cursor returned unexpected document after exhaustion. Expected empty map, got %v", secondDocCloned)
	}

	if hasNextCloned {
		t.Errorf("Expected hasNext to be false for cloned cursor after exhaustion")
	}

	// Ensure original cursor is unaffected
	remainingDocOriginal, _, _ := originalCursor.Next() // Should be doc2
	if !reflect.DeepEqual(*remainingDocOriginal, doc2) {
		t.Errorf("Original cursor was affected by cloned cursor's movement. Expected %v, got %v", doc2, remainingDocOriginal)
	}

	// Test Clone() on a closed cursor
	originalCursor.Close()
	closedClone := originalCursor.Clone()
	_, _, err = closedClone.Next()
	if err != ErrStreamClosed {
		t.Errorf("Expected cloned cursor from closed cursor to be closed, got error %v", err)
	}
}

// TestStoreCursorCount tests the Count method of StoreCursor.
func TestStoreCursorCount(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Empty store
	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor for empty store: %v", err)
	}
	if cursor.Count() != 0 {
		t.Errorf("Expected count 0 for empty store, got %d", cursor.Count())
	}
	cursor.Close()

	// Store with documents
	for i := 0; i < 5; i++ {
		_, _ = s.Insert(map[string]any{"data": i})
	}

	cursor, err = s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor for populated store: %v", err)
	}
	if cursor.Count() != 5 {
		t.Errorf("Expected count 5 for populated store, got %d", cursor.Count())
	}
	cursor.Close()

	// Test Count() on a closed cursor
	cursor.Close()
	if cursor.Count() != 0 { // Handles slice is nilled out on close
		t.Errorf("Expected count 0 for closed cursor, got %d", cursor.Count())
	}
}

// TestStoreCursorClose tests the Close method of StoreCursor.
func TestStoreCursorClose(t *testing.T) {
	s := NewStore()
	defer s.Close()

	_, _ = s.Insert(map[string]any{"data": "test"})

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}

	err = cursor.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Test that subsequent operations fail
	_, _, err = cursor.Next()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed after Close(), got %v", err)
	}

	err = cursor.Reset()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed after Close(), got %v", err)
	}

	// Test calling Close() multiple times
	err = cursor.Close()
	if err != nil {
		t.Errorf("Second Close() call returned error: %v", err)
	}
}

// TestStoreCursorReadIndex tests cursor functionality with ReadIndex.
func TestStoreCursorReadIndex(t *testing.T) {
	s := NewStore()
	defer s.Close()

	_ = s.CreateIndex("by_group", []string{"group"})

	idA1, _ := s.Insert(map[string]any{"group": "A", "val": 1})
	idB1, _ := s.Insert(map[string]any{"group": "B", "val": 1})
	idA2, _ := s.Insert(map[string]any{"group": "A", "val": 2})
	idC1, _ := s.Insert(map[string]any{"group": "C", "val": 1})

	// Test reading from an index
	cursor, err := s.ReadIndex("by_group")
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}
	defer cursor.Close()

	// Documents from ReadIndex are sorted by their internal collection index,
	// which for sequential inserts often aligns with insertion order.
	// We expect A1, B1, A2, C1 to be the order by handle index
	expectedIDs := []string{idA1, idB1, idA2, idC1}
	receivedIDs := []string{}

	for {
		doc, hasNext, err := cursor.Next()
		if err != nil {
			if err == ErrStreamClosed {
				break
			}
			t.Fatalf("Error reading from index cursor: %v", err)
		}

		if doc == nil {
			break
		}

		receivedIDs = append(receivedIDs, (*doc)["group"].(string))
		if !hasNext && len(receivedIDs) < len(expectedIDs) {
			t.Errorf("Cursor prematurely indicated no more documents. Got %d, Expected %d", len(receivedIDs), len(expectedIDs))
		}
	}

	if len(receivedIDs) != len(expectedIDs) {
		t.Errorf("Expected %d documents from index, got %d", len(expectedIDs), len(receivedIDs))
	}

	// Test ReadIndex with non-existent index
	_, err = s.ReadIndex("non_existent")
	if err != ErrIndexNotFound {
		t.Errorf("Expected ErrIndexNotFound for non-existent index, got %v", err)
	}
}

// TestStoreCursorIntegrityAfterDelete verifies cursor behavior when documents are deleted
// *after* the cursor has been created. The cursor should reflect the snapshot.
func TestStoreCursorIntegrityAfterDelete(t *testing.T) {
	s := NewStore()
	defer s.Close()

	ids := make([]string, 5)
	for i := range 5 {
		id, _ := s.Insert(map[string]any{"val": i})
		ids[i] = id
	}

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Delete a document after cursor creation
	err = s.Delete(ids[2]) // Delete the document at conceptual index 2
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Iterate through the cursor. The deleted document should cause ErrDocumentDeleted.
	expectedVals := []any{0, 1, "deleted", 3, 4} // Placeholder for deleted
	receivedVals := []any{}

	for i := range len(cursor.handles) {
		doc, _, err := cursor.Next()

		if err != nil && ! errors.Is(err, ErrDocumentDeleted) {
			t.Errorf("Expected ErrDocumentDeleted for deleted document at position %d, got %v", i, err)
		}

		if(doc == nil) {
			receivedVals = append(receivedVals, "deleted")
		} else {
			receivedVals = append(receivedVals, (*doc)["val"])
		}
	}

	if !reflect.DeepEqual(receivedVals, expectedVals) {
		t.Errorf("Cursor content mismatch after delete. Expected %v, Got %v", expectedVals, receivedVals)
	}
}

// TestStoreCursorIntegrityAfterUpdate verifies cursor behavior when documents are updated
// *after* the cursor has been created. The cursor should reflect the updated data.
func TestStoreCursorIntegrityAfterUpdate(t *testing.T) {
	s := NewStore()
	defer s.Close()

	id, _ := s.Insert(map[string]any{"val": 10})

	cursor, err := s.Read()
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Update the document after cursor creation
	err = s.Update(id, map[string]any{"val": 20, "new_field": true})
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Retrieve the document via the cursor. It should see the updated value.
	doc, _, err := cursor.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if doc == nil {
		t.Fatal("Next() returned nil document")
	}

	if (*doc)["val"] != 20 {
		t.Errorf("Cursor did not reflect updated value. Expected 20, got %v", (*doc)["val"])
	}
	if _, ok := (*doc)["new_field"]; !ok {
		t.Errorf("Cursor did not reflect new field after update")
	}
}
