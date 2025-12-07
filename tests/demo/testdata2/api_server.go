package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type User struct {
	ID       int    
	Username string 
	Email    string 
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := User{
		ID:       1,
		Username: "john_doe",
		Email:    "john@example.com",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func main() {
	http.HandleFunc("/api/user", getUserHandler)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
