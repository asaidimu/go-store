package gostore

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
)

// TestNewStore tests the initialization of the store.
func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.documents == nil {
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

	_, err := s.Insert(Document{"test": "data"})
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}
}

// TestInsert tests inserting documents.
func TestInsert(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc := Document{"name": "Test Doc", "value": 123}
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

	doc := Document{"key": "value"}
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

	originalDoc := Document{"name": "Old Name", "age": 20}
	id, _ := s.Insert(originalDoc)

	updatedDocData := Document{"name": "New Name", "age": 25, "city": "NYC"}
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
	err = s.Update("non_existent_id", Document{"a": 1})
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound for non-existent update, got %v", err)
	}
}

// TestDelete tests deleting documents.
func TestDelete(t *testing.T) {
	s := NewStore()
	defer s.Close()

	doc := Document{"data": "to be deleted"}
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
	_, _ = s.Insert(Document{"user": "Alice", "country": "USA", "age": 30})
	_, _ = s.Insert(Document{"user": "Bob", "country": "Canada", "age": 25})
	_, _ = s.Insert(Document{"user": "Charlie", "country": "USA", "age": 35})
	_, _ = s.Insert(Document{"user": "Alice", "country": "UK", "age": 28}) // Another Alice

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
		_, _ = s.Insert(Document{"item": fmt.Sprintf("Item%d", i), "score": i})
	}
	_, _ = s.Insert(Document{"item": "Special", "score": 5.5}) // Float value

	// Test range lookup [3, 6) -> should include 3, 4, 5, 5.5
	results, err := s.LookupRange("by_score", []any{3}, []any{6})
	if err != nil {
		t.Fatalf("LookupRange failed: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 documents for range [3, 6), got %d", len(results))
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
		id, _ := s.Insert(Document{"num": i, "data": fmt.Sprintf("Doc %d", i)})
		insertedIDs[id] = true
	}

	stream := s.Stream(10) // Buffer size 10. <-- culprit?
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
				doc := Document{"goroutine": gID, "index": i}
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
		id, _ := s.Insert(Document{"counter": 0, "state": "active"})
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
					err := s.Update(docID, Document{"counter": j + 1, "state": "updated"})
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
			_, err := s.Insert(Document{"id": fmt.Sprintf("doc_%d", i), "category": "B"})
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
			_, _ = s.Insert(Document{"id": fmt.Sprintf("initial_%d", i), "category": "A"})
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

