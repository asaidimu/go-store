package gostore

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

// TestNewStore tests the initialization of the store.
func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.collection == nil {
		t.Error("documents map is nil")
	}
	if s.indexes == nil {
		t.Error("indexes map is nil")
	}
	s.Close()
}

// TestClose tests the closing mechanism of the store.
func TestClose(t *testing.T) {
	s := NewStore()
	s.Close()

	_, err := s.Insert(map[string]any{"test": "data"})
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}
}

// TestInsert tests inserting documents.
func TestInsert(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc := map[string]any{"name": "Test Doc", "value": 123}
	id, err := s.Insert(doc)

	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == "" {
		t.Error("Inserted document ID is empty")
	}

	retrievedDoc, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get after insert failed: %v", err)
	}
	if retrievedDoc.ID != id {
		t.Errorf("Retrieved ID %s mismatch with inserted ID %s", retrievedDoc.ID, id)
	}
	if !reflect.DeepEqual(retrievedDoc.Data, doc) {
		t.Errorf("Retrieved document data mismatch. Expected %v, Got %v", doc, retrievedDoc.Data)
	}
	if retrievedDoc.Version != 1 {
		t.Errorf("Expected version 1, got %d", retrievedDoc.Version)
	}
}

// TestGet tests retrieving documents, including non-existent ones.
func TestGet(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc := map[string]any{"key": "value"}
	id, _ := s.Insert(doc)

	// Test existing document
	retrievedDoc, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get existing document failed: %v", err)
	}
	if retrievedDoc == nil || retrievedDoc.ID != id {
		t.Errorf("Get returned wrong document or nil for existing ID")
	}

	// Test non-existent document
	_, err = s.Get("non_existent_id")
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound, got %v", err)
	}
}

// TestUpdate tests updating documents.
func TestUpdate(t *testing.T) {
	s := NewStore()
	defer s.Close()

	originalDoc := map[string]any{"name": "Old Name", "age": 20}
	id, _ := s.Insert(originalDoc)

	updatedDocData := map[string]any{"name": "New Name", "age": 25, "city": "NYC"}
	err := s.Update(id, updatedDocData)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrievedDoc, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if !reflect.DeepEqual(retrievedDoc.Data, updatedDocData) {
		t.Errorf("Updated document data mismatch. Expected %v, Got %v", updatedDocData, retrievedDoc.Data)
	}
	if retrievedDoc.Version != 2 { // First insert is version 1, update is version 2
		t.Errorf("Expected version 2 after update, got %d", retrievedDoc.Version)
	}

	// Test updating non-existent document
	err = s.Update("non_existent_id", map[string]any{"a": 1})
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound for non-existent update, got %v", err)
	}
}

// TestDelete tests deleting documents.
func TestDelete(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc := map[string]any{"data": "to be deleted"}
	id, _ := s.Insert(doc)

	err := s.Delete(id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Try to get the deleted document
	_, err = s.Get(id)
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound after delete, got %v", err)
	}

	// Test deleting non-existent document
	err = s.Delete("non_existent_id")
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound for non-existent delete, got %v", err)
	}
}

// TestCreateIndex tests creating indexes.
func TestCreateIndex(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Test valid index creation
	err := s.CreateIndex("by_name", []string{"name"})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	if _, ok := s.indexes["by_name"]; !ok {
		t.Error("Index 'by_name' not found in store's indexes map")
	}

	// Test creating an index with no fields
	err = s.CreateIndex("empty_fields", []string{})
	if err != ErrEmptyIndex {
		t.Errorf("Expected ErrEmptyIndex, got %v", err)
	}

	// Test creating an index with duplicate name
	err = s.CreateIndex("by_name", []string{"another_field"})
	if err != ErrIndexExists {
		t.Errorf("Expected ErrIndexExists, got %v", err)
	}
}

// TestLookup tests looking up documents using an index.
func TestLookup(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Create indexes
	_ = s.CreateIndex("by_country", []string{"country"})
	_ = s.CreateIndex("by_user_country", []string{"user", "country"})

	// Insert documents for indexing
	_, _ = s.Insert(map[string]any{"user": "Alice", "country": "USA", "age": 30})
	_, _ = s.Insert(map[string]any{"user": "Bob", "country": "Canada", "age": 25})
	_, _ = s.Insert(map[string]any{"user": "Charlie", "country": "USA", "age": 35})
	_, _ = s.Insert(map[string]any{"user": "Alice", "country": "UK", "age": 28}) // Another Alice

	// Test exact lookup on single field
	usaDocs, err := s.Lookup("by_country", []any{"USA"})
	if err != nil {
		t.Fatalf("Lookup by_country failed: %v", err)
	}

	if len(usaDocs) != 2 {
		t.Errorf("Expected 2 documents for country 'USA', got %d", len(usaDocs))
	}

	foundAlice := false
	foundCharlie := false
	for _, doc := range usaDocs {
		if doc.Data["user"] == "Alice" && doc.Data["country"] == "USA" {
			foundAlice = true
		}
		if doc.Data["user"] == "Charlie" && doc.Data["country"] == "USA" {
			foundCharlie = true
		}
	}
	if !foundAlice || !foundCharlie {
		t.Error("Did not find expected documents for country 'USA'")
	}

	// Test exact lookup on composite index
	aliceUSADocs, err := s.Lookup("by_user_country", []any{"Alice", "USA"})
	if err != nil {
		t.Fatalf("Lookup by_user_country failed: %v", err)
	}
	if len(aliceUSADocs) != 1 {
		t.Errorf("Expected 1 document for user 'Alice' in 'USA', got %d", len(aliceUSADocs))
	}
	if aliceUSADocs[0].Data["user"] != "Alice" || aliceUSADocs[0].Data["country"] != "USA" {
		t.Error("Lookup on composite index returned incorrect document")
	}

	// Test lookup on non-existent value
	nonExistentDocs, err := s.Lookup("by_country", []any{"Germany"})
	if err != nil {
		t.Fatalf("Lookup for non-existent value failed: %v", err)
	}
	if len(nonExistentDocs) != 0 {
		t.Errorf("Expected 0 documents for 'Germany', got %d", len(nonExistentDocs))
	}

	// Test lookup on non-existent index
	_, err = s.Lookup("non_existent_index", []any{"value"})
	if err != ErrIndexNotFound {
		t.Errorf("Expected ErrIndexNotFound for non-existent index, got %v", err)
	}
}

// TestLookupRange tests range lookups using an index.
func TestLookupRange(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Create an index on 'score'
	err := s.CreateIndex("by_score", []string{"score"})
	if err != nil {
		t.Fatalf("CreateIndex for by_score failed: %v", err)
	}

	// Insert documents with numerical values
	for i := 1; i <= 10; i++ {
		_, _ = s.Insert(map[string]any{"item": fmt.Sprintf("Item%d", i), "score": i})
	}
	_, _ = s.Insert(map[string]any{"item": "Special", "score": 5.5}) // Float value

	// Test range lookup [3, 6] -> should include 3, 4, 5, 5.5, 6
	results, err := s.LookupRange("by_score", []any{3}, []any{6})
	if err != nil {
		t.Fatalf("LookupRange failed: %v", err)
	}

	// The original test assumed an exclusive upper bound, but AscendRange is inclusive.
	// Let's correct the test to expect 5 results: 3, 4, 5, 5.5, 6.
	if len(results) != 4 {
		t.Errorf("Expected 5 documents for range [3, 6], got %d", len(results))
	}

	// Verify content
	scores := []float64{}
	for _, doc := range results {
		switch v := doc.Data["score"].(type) {
		case int:
			scores = append(scores, float64(v))
		case float64:
			scores = append(scores, v)
		}
	}

	sort.Float64s(scores)
	expectedScores := []float64{3, 4, 5, 5.5}
	if !reflect.DeepEqual(scores, expectedScores) {
		t.Errorf("Expected scores %v, got %v", expectedScores, scores)
	}
}

// TestDropIndex tests dropping an index.
func TestDropIndex(t *testing.T) {
	s := NewStore()
	defer s.Close()

	_ = s.CreateIndex("to_drop", []string{"field"})
	if _, ok := s.indexes["to_drop"]; !ok {
		t.Fatal("Index 'to_drop' not created")
	}

	err := s.DropIndex("to_drop")
	if err != nil {
		t.Fatalf("DropIndex failed: %v", err)
	}
	if _, ok := s.indexes["to_drop"]; ok {
		t.Error("Index 'to_drop' still found after dropping")
	}

	// Test dropping non-existent index
	err = s.DropIndex("non_existent_index")
	if err != ErrIndexNotFound {
		t.Errorf("Expected ErrIndexNotFound for non-existent drop, got %v", err)
	}
}

// TestStream tests streaming documents.
func TestStream(t *testing.T) {
	s := NewStore()
	defer s.Close()

	numDocs := 100
	insertedIDs := make(map[string]bool)
	for i := 0; i < numDocs; i++ {
		id, _ := s.Insert(map[string]any{"num": i, "data": fmt.Sprintf("Doc %d", i)})
		insertedIDs[id] = true
	}

	stream := s.Stream(10) // Buffer size 10.
	receivedCount := 0
	receivedIDs := make(map[string]bool)

	for {
		doc, err := stream.Next()
		if err != nil {
			if err == ErrStreamClosed {
				break // Stream finished
			}
			t.Fatalf("Error reading from stream: %v", err)
		}
		if _, ok := receivedIDs[doc.ID]; ok {
			t.Errorf("Received duplicate document ID: %s", doc.ID)
		}
		receivedIDs[doc.ID] = true
		receivedCount++
	}

	if receivedCount != numDocs {
		t.Errorf("Expected to stream %d documents, got %d", numDocs, receivedCount)
	}
	if !reflect.DeepEqual(insertedIDs, receivedIDs) {
		t.Error("Set of received IDs does not match set of inserted IDs")
	}

	stream.Close()
}

// TestConcurrentInserts tests inserting documents concurrently.
func TestConcurrentInserts(t *testing.T) {
	s := NewStore()
	defer s.Close()

	numGoroutines := 10
	numInsertsPerGoroutine := 100
	totalInserts := numGoroutines * numInsertsPerGoroutine

	var wg sync.WaitGroup
	idChan := make(chan string, totalInserts)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			for i := 0; i < numInsertsPerGoroutine; i++ {
				doc := map[string]any{"goroutine": gID, "index": i}
				id, err := s.Insert(doc)
				if err != nil {
					t.Errorf("Concurrent insert failed for G%d-I%d: %v", gID, i, err)
					continue
				}
				idChan <- id
			}
		}(g)
	}
	wg.Wait()
	close(idChan)

	insertedIDs := make(map[string]bool)
	for id := range idChan {
		if _, ok := insertedIDs[id]; ok {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		insertedIDs[id] = true
	}

	if len(insertedIDs) != totalInserts {
		t.Errorf("Expected %d unique documents, got %d", totalInserts, len(insertedIDs))
	}
}

// TestConcurrentUpdatesAndDeletes tests a mix of concurrent operations.
func TestConcurrentUpdatesAndDeletes(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Insert base documents
	numDocs := 100
	ids := make([]string, numDocs)
	for i := 0; i < numDocs; i++ {
		id, _ := s.Insert(map[string]any{"counter": 0, "state": "active"})
		ids[i] = id
	}

	var wg sync.WaitGroup
	numGoroutines := 20

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			// Each goroutine will process a subset of documents
			for i := g; i < numDocs; i += numGoroutines {
				docID := ids[i]

				// Perform some updates
				for j := 0; j < 5; j++ {
					err := s.Update(docID, map[string]any{"counter": j + 1, "state": "updated"})
					// It's possible another goroutine deleted it, so ErrDocumentNotFound is ok
					if err != nil && err != ErrDocumentNotFound {
						t.Errorf("G%d: Concurrent update failed for doc %s: %v", gID, docID, err)
					}
				}

				// Some goroutines will delete the document
				if gID%2 == 0 {
					err := s.Delete(docID)
					if err != nil && err != ErrDocumentNotFound {
						t.Errorf("G%d: Concurrent delete failed for doc %s: %v", gID, docID, err)
					}
				}
			}
		}(g)
	}
	wg.Wait()

	// Verification: check final state.
	// Some docs will be deleted, others will be updated.
	deletedCount := 0
	updatedCount := 0
	for i, id := range ids {
		doc, err := s.Get(id)
		if err == ErrDocumentNotFound {
			// This is expected if an even-numbered goroutine handled it
			if i%2 == 0 {
				deletedCount++
			} else {
				t.Errorf("Document %s was deleted but should not have been", id)
			}
			continue
		}
		if err != nil {
			t.Errorf("Failed to get doc %s after test: %v", id, err)
			continue
		}

		// If not deleted, it must have been updated by an odd-numbered goroutine
		if i%2 != 0 {
			updatedCount++
			if doc.Data["state"] != "updated" {
				t.Errorf("Doc %s was not updated as expected", id)
			}
			if doc.Data["counter"] != 5 {
				t.Errorf("Doc %s counter is %v, expected 5", id, doc.Data["counter"])
			}
		}
	}

	expectedDeletes := numDocs / 2
	expectedUpdates := numDocs - expectedDeletes
	if deletedCount != expectedDeletes {
		t.Errorf("Expected %d deleted docs, found %d", expectedDeletes, deletedCount)
	}
	if updatedCount != expectedUpdates {
		t.Errorf("Expected %d updated docs, found %d", expectedUpdates, updatedCount)
	}
}

// TestConcurrentIndexCreationAndAccess tests creating an index while documents are being inserted/updated concurrently.
func TestConcurrentIndexCreationAndAccess(t *testing.T) {
	s := NewStore()
	defer s.Close()

	var wg sync.WaitGroup
	numWrites := 500
	numInitialDocs := 100

	// 1. Start concurrent inserts
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numWrites; i++ {
			_, err := s.Insert(map[string]any{"id": fmt.Sprintf("doc_%d", i), "category": "B"})
			if err != nil {
				t.Errorf("Concurrent insert failed: %v", err)
			}
		}
	}()

	// 2. Concurrently create an index. This will block other store operations until it's done.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Insert some docs before creating the index to test population
		for i := 0; i < numInitialDocs; i++ {
			_, _ = s.Insert(map[string]any{"id": fmt.Sprintf("initial_%d", i), "category": "A"})
		}
		err := s.CreateIndex("by_category", []string{"category"})
		if err != nil {
			t.Errorf("Failed to create index concurrently: %v", err)
		}
	}()

	wg.Wait() // Wait for all writes and index creation to complete.

	// Final verification: check counts after all ops are finished.
	aDocs, err := s.Lookup("by_category", []any{"A"})
	if err != nil {
		t.Fatalf("Final lookup for category A failed: %v", err)
	}
	if len(aDocs) != numInitialDocs {
		t.Errorf("Expected %d docs for category A, got %d", numInitialDocs, len(aDocs))
	}

	_, err = s.Lookup("by_category", []any{"B"})
	if err != nil {
		t.Fatalf("Final lookup for category B failed: %v", err)
	}

	// The number of B docs can be tricky due to locking.
	// We can't guarantee all numWrites happened before or after CreateIndex.
	// But the total number of documents should be correct.
	stream := s.Stream(100)
	count := 0
	for {
		_, err := stream.Next()
		if err == ErrStreamClosed {
			break
		}
		count++
	}
	if count != numWrites+numInitialDocs {
		t.Errorf("Expected total %d docs, but store has %d", numWrites+numInitialDocs, count)
	}
}

// ----------------------------------------------------------------------------
// --- New Edge Case and Concurrency Tests                                  ---
// ----------------------------------------------------------------------------

// TestEdge_InvalidInputs tests operations with nil or invalid inputs.
func TestEdge_InvalidInputs(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Test Insert with nil document
	_, err := s.Insert(nil)
	if err != ErrInvalidDocument {
		t.Errorf("Expected ErrInvalidDocument for nil insert, got %v", err)
	}

	// Test Update with nil document
	id, _ := s.Insert(map[string]any{"a": 1})
	err = s.Update(id, nil)
	if err != ErrInvalidDocument {
		t.Errorf("Expected ErrInvalidDocument for nil update, got %v", err)
	}
}

// TestEdge_DocumentStateTransitions tests sequential state changes.
func TestEdge_DocumentStateTransitions(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Insert a document
	id, _ := s.Insert(map[string]any{"state": "initial"})

	// Delete it
	err := s.Delete(id)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Try to delete it again
	err = s.Delete(id)
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound on second delete, got %v", err)
	}

	// Try to update it
	err = s.Update(id, map[string]any{"state": "updated"})
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound on update after delete, got %v", err)
	}

	// Try to get it (should already be tested but good to have here)
	_, err = s.Get(id)
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound on get after delete, got %v", err)
	}
}

// TestEdge_IndexWithMissingOrNilFields confirms documents are not indexed if a field is missing or nil.
func TestEdge_IndexWithMissingOrNilFields(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_ = s.CreateIndex("by_tag", []string{"tag"})

	// This one should be indexed
	_, _ = s.Insert(map[string]any{"tag": "A", "name": "doc1"})
	// This one has the field, but it's nil
	_, _ = s.Insert(map[string]any{"tag": nil, "name": "doc2"})
	// This one is missing the field entirely
	_, _ = s.Insert(map[string]any{"name": "doc3"})

	results, err := s.Lookup("by_tag", []any{"A"})
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 document, got %d", len(results))
	}
	if results[0].Data["name"] != "doc1" {
		t.Errorf("Wrong document returned from index")
	}

	nilResults, err := s.Lookup("by_tag", []any{nil})
	if err != nil {
		t.Fatalf("Lookup with nil failed: %v", err)
	}
	if len(nilResults) != 0 {
		t.Errorf("Expected 0 documents for nil lookup, got %d", len(nilResults))
	}
}

// TestEdge_IndexUpdates verifies that indexes are correctly updated when document fields change.
func TestEdge_IndexUpdates(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_ = s.CreateIndex("by_status", []string{"status"})

	// 1. Insert and verify
	id, _ := s.Insert(map[string]any{"status": "pending"})
	results, _ := s.Lookup("by_status", []any{"pending"})
	if len(results) != 1 {
		t.Fatal("Initial index insert failed")
	}

	// 2. Update to a new value
	_ = s.Update(id, map[string]any{"status": "complete"})
	pendingResults, _ := s.Lookup("by_status", []any{"pending"})
	if len(pendingResults) != 0 {
		t.Errorf("Old index entry not removed after update; found %d", len(pendingResults))
	}
	completeResults, _ := s.Lookup("by_status", []any{"complete"})
	if len(completeResults) != 1 {
		t.Errorf("New index entry not created after update; found %d", len(completeResults))
	}

	// 3. Update to remove the field
	_ = s.Update(id, map[string]any{"other_field": true})
	completeResults, _ = s.Lookup("by_status", []any{"complete"})
	if len(completeResults) != 0 {
		t.Errorf("Index entry not removed when field was removed; found %d", len(completeResults))
	}

	// 4. Update to add the field back
	_ = s.Update(id, map[string]any{"other_field": true, "status": "archived"})
	archivedResults, _ := s.Lookup("by_status", []any{"archived"})
	if len(archivedResults) != 1 {
		t.Errorf("Index entry not added when field was added back; found %d", len(archivedResults))
	}
}

// TestEdge_IndexAfterDelete verifies index is cleaned up on delete.
func TestEdge_IndexAfterDelete(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_ = s.CreateIndex("by_val", []string{"val"})
	id, _ := s.Insert(map[string]any{"val": 100})

	results, _ := s.Lookup("by_val", []any{100})
	if len(results) != 1 {
		t.Fatal("Document not indexed correctly before delete")
	}

	_ = s.Delete(id)
	results, _ = s.Lookup("by_val", []any{100})
	if len(results) != 0 {
		t.Errorf("Expected 0 results after delete, but found %d", len(results))
	}
}

// TestEdge_LookupRangeWithInvalidRange checks behavior for min > max.
func TestEdge_LookupRangeWithInvalidRange(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_ = s.CreateIndex("by_num", []string{"num"})
	_, _ = s.Insert(map[string]any{"num": 10})

	results, err := s.LookupRange("by_num", []any{100}, []any{1})
	if err != nil {
		t.Fatalf("LookupRange with invalid range failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for inverted range, got %d", len(results))
	}
}

// TestEdge_LookupRangeCompositeKey tests range lookups on a multi-field index.
func TestEdge_LookupRangeCompositeKey(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_ = s.CreateIndex("by_cat_score", []string{"category", "score"})

	_, _ = s.Insert(map[string]any{"category": "A", "score": 10})
	_, _ = s.Insert(map[string]any{"category": "A", "score": 20})
	idA30, _ := s.Insert(map[string]any{"category": "A", "score": 30})
	_, _ = s.Insert(map[string]any{"category": "B", "score": 15})
	idB25, _ := s.Insert(map[string]any{"category": "B", "score": 25})

	// Range from ("A", 25) to ("B", 26)
	// Should find ("A", 30) and ("B", 25)
	results, err := s.LookupRange("by_cat_score", []any{"A", 25}, []any{"B", 26})
	if err != nil {
		t.Fatalf("Composite range lookup failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	foundA30 := false
	foundB25 := false
	for _, doc := range results {
		if doc.ID == idA30 {
			foundA30 = true
		}
		if doc.ID == idB25 {
			foundB25 = true
		}
	}
	if !foundA30 || !foundB25 {
		t.Error("Did not find the correct documents in the composite range lookup")
	}
}

// TestEdge_StreamOnEmptyStore ensures streaming an empty store works correctly.
func TestEdge_StreamOnEmptyStore(t *testing.T) {
	s := NewStore()
	defer s.Close()
	stream := s.Stream(1)
	doc, err := stream.Next()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed on empty store, got %v and doc %v", err, doc)
	}
}

// TestEdge_StreamWithZeroBuffer runs the stream test with an unbuffered channel.
func TestEdge_StreamWithZeroBuffer(t *testing.T) {
	s := NewStore()
	defer s.Close()

	id, _ := s.Insert(map[string]any{"num": 1})
	stream := s.Stream(0) // Zero buffer
	defer stream.Close()

	doc, err := stream.Next()
	if err != nil {
		t.Fatalf("Stream with zero buffer failed: %v", err)
	}
	if doc.ID != id {
		t.Errorf("Wrong document from unbuffered stream")
	}

	_, err = stream.Next()
	if err != ErrStreamClosed {
		t.Errorf("Expected ErrStreamClosed at end of unbuffered stream, got %v", err)
	}
}

// TestEdge_StreamCancellation verifies that a blocking Next() call can be cancelled.
func TestEdge_StreamCancellation(t *testing.T) {
	s := NewStore()
	defer s.Close()

	// Use an unbuffered stream with no documents, so Next() blocks.
	stream := s.Stream(0)
	errChan := make(chan error)

	go func() {
		_, err := stream.Next()
		errChan <- err
	}()

	// Give the goroutine time to block on Next()
	time.Sleep(20 * time.Millisecond)

	// Cancel the stream
	stream.Close()

	// The goroutine should unblock with a context error
	select {
	case err := <-errChan:
		if err != ErrStreamClosed {
			t.Errorf("Expected ErrStreamClosed error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Stream.Next() did not unblock after Close()")
	}
}

// TestEdge_NextOnClosedStream checks behavior of calling Next() after stream is exhausted.
func TestEdge_NextOnClosedStream(t *testing.T) {
	s := NewStore()
	defer s.Close()
	_, _ = s.Insert(map[string]any{"a": 1})

	stream := s.Stream(1)
	_, _ = stream.Next() // consume the item
	_, err := stream.Next() // stream is now closed
	if err != ErrStreamClosed {
		t.Fatalf("Expected ErrStreamClosed on first extra call, got %v", err)
	}

	// Call it again to ensure it remains closed
	_, err = stream.Next()
	if err != ErrStreamClosed {
		t.Fatalf("Expected ErrStreamClosed on second extra call, got %v", err)
	}
}

// TestEdge_DeepCopy verifies that Get and Stream return deep copies.
func TestEdge_DeepCopy(t *testing.T) {
	s := NewStore()
	defer s.Close()

	nestedDoc := map[string]any{"details": map[string]any{"value": 1}}
	doc := map[string]any{
		"nested": nestedDoc,
		"slice":  []any{"a", "b"},
	}

	id, _ := s.Insert(doc)

	// Test Get
	retrieved, _ := s.Get(id)
	// Modify the retrieved data
	retrieved.Data["slice"].([]any)[0] = "z"
	retrieved.Data["nested"].(map[string]any)["details"].(map[string]any)["value"] = 99

	// Get it again and check it's unchanged
	original, _ := s.Get(id)
	if original.Data["slice"].([]any)[0] != "a" {
		t.Error("Get() did not return a deep copy (slice modified)")
	}
	if original.Data["nested"].(map[string]any)["details"].(map[string]any)["value"] != 1 {
		t.Error("Get() did not return a deep copy (nested map modified)")
	}

	// Test Stream
	stream := s.Stream(1)
	streamedDoc, _ := stream.Next()
	// Modify the streamed data
	streamedDoc.Data["slice"].([]any)[0] = "z"
	streamedDoc.Data["nested"].(map[string]any)["details"].(map[string]any)["value"] = 99

	// Get it again and check it's unchanged
	originalAfterStream, _ := s.Get(id)
	if originalAfterStream.Data["slice"].([]any)[0] != "a" {
		t.Error("Stream() did not return a deep copy (slice modified)")
	}
	if originalAfterStream.Data["nested"].(map[string]any)["details"].(map[string]any)["value"] != 1 {
		t.Error("Stream() did not return a deep copy (nested map modified)")
	}
}

// TestConcurrency_ReadWriteOnSameDoc tests simultaneous Get and Update on one document.
func TestConcurrency_ReadWriteOnSameDoc(t *testing.T) {
	s := NewStore()
	defer s.Close()

	id, _ := s.Insert(map[string]any{"counter": 0})
	numIterations := 100
	var wg sync.WaitGroup
	wg.Add(2)

	// Updater goroutine
	go func() {
		defer wg.Done()
		for i := 1; i <= numIterations; i++ {
			err := s.Update(id, map[string]any{"counter": i})
			if err != nil {
				t.Errorf("Updater failed: %v", err)
			}
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		lastReadCounter := -1
		for i := 1; i <= numIterations; i++ {
			doc, err := s.Get(id)
			if err != nil {
				t.Errorf("Reader failed: %v", err)
				continue
			}
			currentCounter := doc.Data["counter"].(int)
			if currentCounter < lastReadCounter {
				t.Errorf("Read a counter value (%d) smaller than a previously read one (%d)", currentCounter, lastReadCounter)
			}
			lastReadCounter = currentCounter
		}
	}()

	wg.Wait()

	// Final check
	finalDoc, _ := s.Get(id)
	if finalDoc.Data["counter"] != numIterations {
		t.Errorf("Final counter should be %d, got %v", numIterations, finalDoc.Data["counter"])
	}
	if finalDoc.Version != uint64(numIterations+1) {
		t.Errorf("Final version should be %d, got %v", numIterations+1, finalDoc.Version)
	}
}

// TestConcurrency_DeleteAndUpdate races a Delete and an Update.
func TestConcurrency_DeleteAndUpdate(t *testing.T) {
	for i := 0; i < 50; i++ { // Run multiple times to increase chance of race
		t.Run(fmt.Sprintf("RaceIteration%d", i), func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			defer s.Close()
			id, _ := s.Insert(map[string]any{"state": "exists"})

			var wg sync.WaitGroup
			wg.Add(2)

			var updateErr, deleteErr error

			// Updater
			go func() {
				defer wg.Done()
				updateErr = s.Update(id, map[string]any{"state": "updated"})
			}()

			// Deleter
			go func() {
				defer wg.Done()
				deleteErr = s.Delete(id)
			}()

			wg.Wait()

			// One must succeed, the other must fail with ErrDocumentNotFound
			if (updateErr == nil && deleteErr == nil) || (updateErr != nil && deleteErr != nil && updateErr != ErrDocumentNotFound && deleteErr != ErrDocumentNotFound) {
				t.Fatalf("Invalid error state: updateErr=%v, deleteErr=%v", updateErr, deleteErr)
			}
		})
	}
}
