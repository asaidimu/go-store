package main

import (
	"fmt"
	"log"

	store "github.com/asaidimu/go-store/v2"
)

func main() {
	fmt.Println("--- Basic Store Usage Example ---")
	s := store.NewStore()
	defer s.Close()

	// 1. Insert Document
	doc1 := map[string]any{"title": "My First Document", "author": "Alice"}
	id1, err := s.Insert(doc1)
	if err != nil {
		log.Fatalf("Basic: Failed to insert document: %v", err)
	}
	fmt.Printf("Basic: Inserted document with ID: %s\n", id1)

	// 2. Get Document
	retrievedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to get document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Retrieved document: ID=%s, Title='%s', Version=%d\n",
		retrievedDoc.ID, retrievedDoc.Data["title"], retrievedDoc.Version)

	// 3. Update Document
	updatedDoc1 := map[string]any{"title": "My First Document (Revised)", "author": "Alice Smith", "pages": 150}
	err = s.Update(id1, updatedDoc1)
	if err != nil {
		log.Fatalf("Basic: Failed to update document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Updated document with ID: %s\n", id1)

	retrievedUpdatedDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to get updated document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Retrieved updated document: ID=%s, Title='%s', Pages=%.0f, Version=%d\n",
		retrievedUpdatedDoc.ID, retrievedUpdatedDoc.Data["title"], retrievedUpdatedDoc.Data["pages"], retrievedUpdatedDoc.Version)

	// 4. Delete Document
	err = s.Delete(id1)
	if err != nil {
		log.Fatalf("Basic: Failed to delete document %s: %v", id1, err)
	}
	fmt.Printf("Basic: Deleted document with ID: %s\n", id1)

	// 5. Try to Get Deleted Document (should fail)
	_, err = s.Get(id1)
	if err != nil {
		fmt.Printf("Basic: Attempted to get deleted document %s: %v (Expected error)\n", id1, err)
	}
}
