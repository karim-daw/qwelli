package main

// just test the db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/db"
)

func main() {
	// Test 1: In-memory database
	fmt.Println("Test 1: In-memory database")
	database, err := db.OpenProjectDB("", 1536)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()
	fmt.Println("✓ In-memory database opened successfully")

	// Test 2: File-based database with directory creation
	fmt.Println("\nTest 2: File-based database (with directory creation)")
	testDBPath := "testdata/test.db"
	database2, err := db.OpenProjectDB(testDBPath, 1536)
	if err != nil {
		log.Fatalf("Failed to open file database: %v", err)
	}
	defer database2.Close()
	fmt.Printf("✓ File database opened successfully at %s\n", testDBPath)

	// Use the file database for the rest of the tests
	database = database2

	// Test: Create a table if it doesn't exist
	_, err = database.Conn().Exec(`CREATE TABLE IF NOT EXISTS people (id INTEGER, name VARCHAR)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created successfully")

	// Test: Insert data
	_, err = database.Conn().Exec(`INSERT INTO people VALUES (12, 'Steve')`)
	if err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	}
	fmt.Println("Data inserted successfully")

	// Test: Query data
	var (
		id   int
		name string
	)
	row := database.Conn().QueryRow(`SELECT id, name FROM people WHERE id = ?`, 42)
	err = row.Scan(&id, &name)
	if errors.Is(err, sql.ErrNoRows) {
		log.Println("No rows found")
	} else if err != nil {
		log.Fatalf("Failed to query data: %v", err)
	}

	fmt.Printf("Query result: id=%d, name=%s\n", id, name)
	fmt.Println("All tests passed!")
}
