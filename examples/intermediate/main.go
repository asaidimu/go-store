package main

import (
	"fmt"
	"log"
	"time"

	store "github.com/asaidimu/go-store"
)

func main() {
	fmt.Println("--- Intermediate Store Usage Example ---")
	s := store.NewStore()
	defer s.Close()

	// Insert several documents
	s.Insert(store.Document{"name": "Alice", "age": 30, "city": "New York"})
	s.Insert(store.Document{"name": "Bob", "age": 25, "city": "London"})
	s.Insert(store.Document{"name": "Charlie", "age": 35, "city": "New York"})
	s.Insert(store.Document{"name": "David", "age": 28, "city": "Paris"})
	s.Insert(store.Document{"name": "Eve", "age": 30, "city": "New York"})

	time.Sleep(100 * time.Millisecond) // Give async index updates time to process

	// 1. Create an Index
	err := s.CreateIndex("by_city", []string{"city"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to create 'by_city' index: %v", err)
	}
	fmt.Println("Intermediate: Created index 'by_city' on field 'city'.")

	err = s.CreateIndex("by_age", []string{"age"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to create 'by_age' index: %v", err)
	}
	fmt.Println("Intermediate: Created index 'by_age' on field 'age'.")

	// 2. Lookup Documents by Exact Match (using "by_city" index)
	fmt.Println("\nIntermediate: Looking up documents in 'New York':")
	nyDocs, err := s.Lookup("by_city", []any{"New York"})
	if err != nil {
		log.Fatalf("Intermediate: Failed to lookup by city: %v", err)
	}
	for _, doc := range nyDocs {
		fmt.Printf("  ID: %s, Name: %s, Age: %.0f\n", doc.ID, doc.Data["name"], doc.Data["age"])
	}

	// 3. Lookup Documents by Range (using "by_age" index)
	fmt.Println("\nIntermediate: Looking up documents with age between 27 and 32:")
	ageRangeDocs, err := s.LookupRange("by_age", []any{27}, []any{32})
	if err != nil {
		log.Fatalf("Intermediate: Failed to lookup age range: %v", err)
	}
	for _, doc := range ageRangeDocs {
		fmt.Printf("  ID: %s, Name: %s, Age: %.0f\n", doc.ID, doc.Data["name"], doc.Data["age"])
	}

	// 4. Stream All Documents
	fmt.Println("\nIntermediate: Streaming all documents:")
	docStream := s.Stream(2) // Small buffer for demonstration
	for {
		docResult, err := docStream.Next()
		if err != nil {
			if err.Error() == "stream closed" {
				fmt.Println("Intermediate: Document stream finished or cancelled.")
				break
			}
			log.Printf("Intermediate: Error reading from stream: %v", err)
			break
		}
		fmt.Printf("  Streamed: ID=%s, Name=%s, City=%s\n",
			docResult.ID, docResult.Data["name"], docResult.Data["city"])
	}
	docStream.Close()
}
