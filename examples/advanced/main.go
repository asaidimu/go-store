package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	store "github.com/asaidimu/go-store/v3"
)

func main() {
	fmt.Println("--- Advanced Store Usage Example ---")
	s := store.NewStore()
	defer s.Close()

	// Insert initial document
	id1, _ := s.Insert(map[string]any{"name": "Original", "value": 100})
	fmt.Printf("Advanced: Initial document ID: %s\n", id1)

	var wg sync.WaitGroup
	const numConcurrentUpdates = 5

	// 1. Concurrent Updates to a single document
	fmt.Printf("\nAdvanced: Performing %d concurrent updates on document %s...\n", numConcurrentUpdates, id1)
	for i := range numConcurrentUpdates {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			newVal := 100 + iteration + 1
			err := s.Update(id1, map[string]any{"name": fmt.Sprintf("Updated %d", iteration+1), "value": newVal})
			if err != nil {
				fmt.Printf("Advanced: Concurrent update %d failed: %v\n", iteration+1, err)
			} else {
				// In a real scenario, you might add a small delay here to make concurrency more evident
				// time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("Advanced: Concurrent updates finished.")

	// Verify the final state and version
	finalDoc, err := s.Get(id1)
	if err != nil {
		log.Fatalf("Advanced: Failed to get final document after concurrent updates: %v", err)
	}
	fmt.Printf("Advanced: Final document state: ID=%s, Name='%s', Value=%.0f, Version=%d\n",
		finalDoc.ID, finalDoc.Data["name"], finalDoc.Data["value"], finalDoc.Version)
	if finalDoc.Version != uint64(numConcurrentUpdates+1) {
		fmt.Printf("Advanced: Warning: Expected version %d, got %d. (Async updates can cause non-sequential versions from a single goroutine's perspective, but total versions should be correct).\n", numConcurrentUpdates+1, finalDoc.Version)
	}

	// 2. Using a theorized Functional Index (if implemented as discussed)
	// This part requires the CreateFunctionalIndex and Filter methods to be added to your store.go
	// For demonstration, we'll assume they exist and run a small example.
	fmt.Println("\nAdvanced: Demonstrating (theorized) Functional Index usage...")
	s.Insert(map[string]any{"product": "Laptop", "price": 1200, "in_stock": true})
	s.Insert(map[string]any{"product": "Mouse", "price": 25, "in_stock": true})
	s.Insert(map[string]any{"product": "Monitor", "price": 300, "in_stock": false})
	s.Insert(map[string]any{"product": "Keyboard", "price": 75, "in_stock": true})

	time.Sleep(100 * time.Millisecond) // Allow updates to process


	fmt.Println("Advanced: (Simulated) Expensive In-Stock Items (price > 100 and in_stock == true):")
	streamAll := s.Stream(10)
	filteredCount := 0
	for {
		docRes, err := streamAll.Next()
		if err != nil {
			break // Stream closed or error
		}
		price, priceOK := docRes.Data["price"].(int)
		inStock, stockOK := docRes.Data["in_stock"].(bool)
		if priceOK && stockOK && price > 100 && inStock {
			fmt.Printf("  Product: %s, Price: %d\n", docRes.Data["product"], price)
			filteredCount++
		}
	}
	streamAll.Close()
	if filteredCount == 0 {
		fmt.Println("  No expensive in-stock items found (simulated, or if functional index not implemented).")
	}

	// 3. Dropping an Index
	fmt.Println("\nAdvanced: Attempting to drop 'by_city' index (created in intermediate example).")
	// This relies on "by_city" having been created in a previous run or being re-created.
	// For this isolated example, it might fail if intermediate_example.go wasn't run first.
	err = s.DropIndex("by_city")
	if err != nil {
		fmt.Printf("Advanced: Failed to drop index 'by_city' (might not exist, run intermediate_example.go first): %v\n", err)
	} else {
		fmt.Println("Advanced: Successfully dropped 'by_city' index.")
	}
}
