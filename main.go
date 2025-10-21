package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// URL struct maps to DB table
type URL struct {
	ID          string    `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	CreatedAt   time.Time `json:"created_at"` 
}

// generate short hash
func generateShortURL(originalURL string) string {
	hasher := md5.New()
	hasher.Write([]byte(originalURL))
	data := hasher.Sum(nil)
	hash := hex.EncodeToString(data)
	return hash[:8]
}

// create new short URL (insert into DB)

func createURL(originalURL string) string {
	shortURL := generateShortURL(originalURL)
	id := shortURL

	_, err := DB.Exec(
		"INSERT INTO urls (id, original_url, short_url, created_at) VALUES (?, ?, ?, ?)",
		id, originalURL, shortURL, time.Now(),
	)
	if err != nil {
		fmt.Printf("DB Insert failed: %v\n", err)
		return ""
	}

	return shortURL
}

// fetch original URL from DB - FIXED: changed creation_date to created_at
func getURL(id string) (URL, error) {
	var url URL
	row := DB.QueryRow("SELECT id, original_url, short_url, created_at FROM urls WHERE id = ?", id)
	err := row.Scan(&url.ID, &url.OriginalURL, &url.ShortURL, &url.CreatedAt) // fixed
	if err != nil {
		if err == sql.ErrNoRows {
			return URL{}, errors.New("URL not found")
		}
		return URL{}, err
	}
	return url, nil
}

// Root handler
func RootPageURL(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, world! This is your URL shortener 🚀")
}

// Shorten handler
func ShortURLHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		URL string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("📥 Received URL to shorten: %s\n", data.URL)

	shortURL := createURL(data.URL)
	if shortURL == "" {
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	response := struct {
		ShortURL string `json:"short_url"`
		FullURL  string `json:"full_url"`
	}{
		ShortURL: shortURL,
		FullURL:  fmt.Sprintf("http://localhost:3000/redirect/%s", shortURL),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Redirect handler - ADDED: debug logging
func redirectURLHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/redirect/"):]
	fmt.Printf("🔍 Looking for ID: '%s'\n", id) // Debug log
	
	url, err := getURL(id)
	if err != nil {
		fmt.Printf("❌ Error getting URL: %v\n", err) // Debug log
		http.Error(w, "Invalid request", http.StatusNotFound)
		return
	}
	
	fmt.Printf("✅ Found URL, redirecting to: %s\n", url.OriginalURL) // Debug log
	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

func main() {
	// init DB
	InitDB()
	defer DB.Close()

	// register routes
	http.HandleFunc("/", RootPageURL)
	http.HandleFunc("/shorten", ShortURLHandler)
	http.HandleFunc("/redirect/", redirectURLHandler)

	// start server
	fmt.Println("🌍 Starting server on port 3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		fmt.Println("Error on starting server:", err)
	}
}
