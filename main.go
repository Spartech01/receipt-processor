package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Item represents a receipt item. Price is stored in cents.
type Item struct {
	ShortDescription string `json:"shortDescription"`
	PriceCents       int    `json:"price"`
}

// Receipt represents the receipt data
type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	TotalCents   int    `json:"total"`
}

// ReceiptPoints holds the calculated points for a receipt.
type ReceiptPoints struct {
	Points int `json:"points"`
}

// Constants for time-based calculations.
var (
	twoPM   time.Time
	fourPM  time.Time
	dateFmt = "2006-01-02"
	timeFmt = "15:04"
)

// init runs once, setting up the time constants.
func init() {
	twoPM, _ = time.Parse(timeFmt, "14:00")
	fourPM, _ = time.Parse(timeFmt, "16:00")
}

// inMemoryStore: a simple key-value store for receipts (ID -> points).
var inMemoryStore = make(map[string]int)

// processReceiptHandler: handles POST requests to /receipts/process.
func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Temporary struct for initial JSON decoding.
	var tempReceipt struct {
		Retailer     string `json:"retailer"`
		PurchaseDate string `json:"purchaseDate"`
		PurchaseTime string `json:"purchaseTime"`
		Items        []struct {
			ShortDescription string `json:"shortDescription"`
			Price            string `json:"price"`
		} `json:"items"`
		Total string `json:"total"`
	}

	// Decode the JSON request body.  If it fails, it's a client error.
	if err := json.NewDecoder(r.Body).Decode(&tempReceipt); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("ERROR: Invalid request body: %v", err)
		return
	}

	// Create the final Receipt struct, cleaning up strings.
	receipt := Receipt{
		Retailer:     strings.TrimSpace(tempReceipt.Retailer),
		PurchaseDate: strings.TrimSpace(tempReceipt.PurchaseDate),
		PurchaseTime: strings.TrimSpace(tempReceipt.PurchaseTime),
		Items:        make([]Item, len(tempReceipt.Items)),
	}

	// Process each item, converting price to cents.
	for i, tempItem := range tempReceipt.Items {
		cents, err := parseCents(tempItem.ShortDescription, tempItem.Price)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid item price: %v", err), http.StatusBadRequest)
			log.Printf("ERROR: Invalid item price: %v", err)
			return
		}
		receipt.Items[i] = Item{
			ShortDescription: strings.TrimSpace(tempItem.ShortDescription),
			PriceCents:       cents,
		}
	}

	// Convert the total to cents.
	totalCents, err := parseCents("total", tempReceipt.Total)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid total price: %v", err), http.StatusBadRequest)
		log.Printf("ERROR: Invalid total price: %v", err)
		return
	}
	receipt.TotalCents = totalCents

	receiptID := uuid.NewString()
	points := calculatePoints(receipt)
	inMemoryStore[receiptID] = points

	// If encode fails, it's a server error.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"id": receiptID}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("ERROR: Failed to encode response: %v", err)
		return
	}
}

// getPointsHandler: handles GET requests to /receipts/{id}/points.
func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	receiptID := mux.Vars(r)["id"]            // Get ID from URL.
	points, found := inMemoryStore[receiptID] // Lookup receipt.
	if !found {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		log.Printf("INFO: Receipt not found: %s", receiptID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ReceiptPoints{Points: points}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("ERROR: Failed to encode points response: %v", err)
		return
	}
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// Add points for retailer name.
	points += countAlphanumeric(receipt.Retailer)

	// Add points for round dollar total.
	if receipt.TotalCents%100 == 0 {
		points += 50
	}
	// Add points if total is a multiple of 0.25.
	if receipt.TotalCents%25 == 0 {
		points += 25
	}

	// Add points from items.
	points += calculateItemPoints(receipt.Items)

	// Add points for odd purchase day.
	purchaseDate, err := time.Parse(dateFmt, receipt.PurchaseDate)
	if err == nil {
		if purchaseDate.Day()%2 != 0 {
			points += 6
		}
		// Add points for purchase time between 2 PM and 4 PM.
		purchaseTime, err := time.Parse(timeFmt, receipt.PurchaseTime)
		if err == nil && isTimeBetween(purchaseTime, twoPM, fourPM) {
			points += 10
		}
	}

	return points
}

func isTimeBetween(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

func countAlphanumeric(s string) int {
	count := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			count++
		}
	}
	return count
}

func calculateItemPoints(items []Item) int {
	totalPoints := 0
	totalPoints += (len(items) / 2) * 5 // Add points for every two items.

	// Add points for item descriptions with trimmed length as a multiple of 3.
	for _, item := range items {
		trimmedLen := len(strings.TrimSpace(item.ShortDescription))
		if trimmedLen%3 == 0 {
			totalPoints += int(math.Ceil(float64(item.PriceCents) * 0.2 / 100))
		}
	}
	return totalPoints
}

// parseCents converts a dollar string (e.g., "12.34") to cents (1234).
func parseCents(description string, dollarAmount interface{}) (int, error) {
	dollarString, ok := dollarAmount.(string)
	if !ok {
		return 0, fmt.Errorf("invalid input: expected string, got %T", dollarAmount)
	}

	parts := strings.Split(dollarString, ".")

	// Parse dollar part.
	dollars, err := parseNum(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid dollar amount in %s: %w", description, err)
	}

	cents := 0
	if len(parts) > 1 {
		cents, err = parseNum(parts[1])
		if err != nil || len(parts[1]) > 2 {
			return 0, fmt.Errorf("invalid cents format in %s: %w", description, err)
		}
		if len(parts[1]) == 1 {
			cents *= 10
		}
	}

	return dollars*100 + cents, nil
}

// parseNum parses a string containing only digits into an integer.
func parseNum(s string) (int, error) {
	num := 0
	for _, char := range s {
		digit := int(char - '0')
		if digit < 0 || digit > 9 {
			return 0, fmt.Errorf("invalid digit")
		}
		num = num*10 + digit
	}
	return num, nil
}

// main sets up the router and starts the HTTP server.
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/receipts/process", processReceiptHandler).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", getPointsHandler).Methods("GET")
	r.Use(loggingMiddleware)
	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r)) // Use log.Fatal for consistency
}

// loggingMiddleware logs incoming requests (middleware).
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r) // Call the next handler.
	})
}
