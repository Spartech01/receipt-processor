package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestProcessReceiptHandler_ExampleReceipt tests the /receipts/process endpoint.
func TestProcessReceiptHandler_ExampleReceipt(t *testing.T) {
	receiptJSON := `{
		"retailer": "Target",
		"purchaseDate": "2022-01-01",
		"purchaseTime": "13:01",
		"items": [
			{"shortDescription": "Mountain Dew 12PK", "price": "6.49"},
			{"shortDescription": "Emils Cheese Pizza", "price": "12.25"},
			{"shortDescription": "Knorr Creamy Chicken", "price": "1.26"},
			{"shortDescription": "Doritos Nacho Cheese", "price": "3.35"},
			{"shortDescription": "Klarbrunn 12-PK 12 FL OZ", "price": "12.00"}
		],
		"total": "35.35"
	}`

	req, err := http.NewRequest("POST", "/receipts/process", bytes.NewBuffer([]byte(receiptJSON)))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/receipts/process", processReceiptHandler).Methods("POST")
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	receiptID, ok := response["id"]
	if !ok {
		t.Error("response body does not contain an 'id' field")
	}

	reqGet, err := http.NewRequest("GET", "/receipts/"+receiptID+"/points", nil)
	if err != nil {
		t.Fatal(err)
	}
	rrGet := httptest.NewRecorder()
	r.HandleFunc("/receipts/{id}/points", getPointsHandler).Methods("GET") // Add handler
	r.ServeHTTP(rrGet, reqGet)

	if status := rrGet.Code; status != http.StatusOK {
		t.Errorf("GET handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var pointsResp ReceiptPoints
	if err := json.Unmarshal(rrGet.Body.Bytes(), &pointsResp); err != nil {
		t.Fatalf("failed to unmarshal points response: %v", err)
	}
	expectedPoints := 28 // The *correct* expected value.
	if pointsResp.Points != expectedPoints {
		t.Errorf("GET handler returned unexpected points: got %v want %v", pointsResp.Points, expectedPoints)
	}
}

func TestGetPointsHandler_NotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/receipts/nonexistent-id/points", nil)
	if err != nil {
		t.Fatal(err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/receipts/{id}/points", getPointsHandler).Methods("GET")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	expected := "Receipt not found\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestCalculatePoints(t *testing.T) {
	tests := []struct {
		name           string
		receipt        Receipt
		expectedPoints int
	}{
		{
			name: "Example Receipt",
			receipt: Receipt{
				Retailer:     "Target",
				PurchaseDate: "2022-01-01",
				PurchaseTime: "13:01",
				Items: []Item{
					{ShortDescription: "Mountain Dew 12PK", PriceCents: 649},
					{ShortDescription: "Emils Cheese Pizza", PriceCents: 1225},
					{ShortDescription: "Knorr Creamy Chicken", PriceCents: 126},
					{ShortDescription: "Doritos Nacho Cheese", PriceCents: 335},
					{ShortDescription: "Klarbrunn 12-PK 12 FL OZ", PriceCents: 1200},
				},
				TotalCents: 3535,
			},
			expectedPoints: 28,
		},
		{
			name: "Round Dollar Total",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02", // Even day
				PurchaseTime: "15:30",      // Between 2 PM and 4 PM
				Items:        []Item{},
				TotalCents:   1000, // Round dollar amount
			},
			expectedPoints: 50 + 10 + 12 + 25, // 50 (round) + 10 (time) + 12 (retailer) + 25 (multiple of 0.25) = 97 //CORRECT
		},
		{
			name: "Multiple of 0.25",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02", // Even day
				PurchaseTime: "10:00",      // NOT between 2 PM and 4 PM
				Items:        []Item{},
				TotalCents:   75, // Multiple of 0.25
			},
			expectedPoints: 25 + 12, // 25 (multiple of 0.25) + 12 (retailer) = 37 //CORRECT
		},
		{
			name: "Odd Day",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-03", // Odd day
				PurchaseTime: "10:00",
				Items:        []Item{},
				TotalCents:   1234,
			},
			expectedPoints: 6 + 12, // 6 (odd day) + 12 (retailer) = 18 //CORRECT
		},
		{
			name: "Between 2 PM and 4 PM",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02", // Even day
				PurchaseTime: "15:00",      // 3 PM (between 2 and 4)
				Items:        []Item{},
				TotalCents:   1234,
			},
			expectedPoints: 10 + 12, // 10 (time) + 12 (retailer) = 22 //CORRECT
		},
		{
			name: "Empty Retailer",
			receipt: Receipt{
				Retailer:     "",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items:        []Item{},
				TotalCents:   100,
			},
			expectedPoints: 50 + 10 + 0 + 25, // 50 (round) + 10 (time) + 0 (retailer) + 25 (multiple of 0.25) = 85 //CORRECT
		},
		{
			name: "No Items",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items:        []Item{},
				TotalCents:   1234,
			},
			expectedPoints: 10 + 12, // 10 (time) + 0 (items) + 12 (retailer) = 22 //CORRECT
		},
		{
			name: "One Item",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items: []Item{
					{ShortDescription: "Test Item", PriceCents: 100}, // Multiple of 3
				},
				TotalCents: 1234,
			},
			expectedPoints: 10 + 1 + 12, // 10 (time) + 1 (item points) + 12 (retailer) = 23 //CORRECT
		},
		{
			name: "Many Items",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items: []Item{
					{ShortDescription: "Item 1", PriceCents: 100}, // Multiple of 3
					{ShortDescription: "Item 2", PriceCents: 200}, // Multiple of 3
					{ShortDescription: "Item 3", PriceCents: 300}, // Multiple of 3
					{ShortDescription: "Item 4", PriceCents: 400}, // Multiple of 3
					{ShortDescription: "Item 5", PriceCents: 500}, // Multiple of 3
					{ShortDescription: "Item 6", PriceCents: 600}, // Multiple of 3
				},
				TotalCents: 1234,
			},
			expectedPoints: 10 + 15 + 12 + 7, // 10 (time) + 15 (pairs) + 12 (retailer) + 7 (Descriptions)= 44 //CORRECT
		},
		{
			name: "Item Description Multiple of 3",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items: []Item{
					{ShortDescription: "ABC", PriceCents: 151}, // 151 * 0.2 = 30.  Ceil(30.2) = 31 / 100 = 0
				},
				TotalCents: 1234,
			},
			expectedPoints: 10 + 12 + 1, // 10 (time) + 12 (retailer) + 1 item = 23 //CORRECT
		},
		{
			name: "Item Description Not Multiple of 3",
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2022-01-02",
				PurchaseTime: "15:00",
				Items: []Item{
					{ShortDescription: "ABCD", PriceCents: 100},
				},
				TotalCents: 1234,
			},
			expectedPoints: 10 + 12, // 10 + 12 //CORRECT
		},
		{
			name: "Unicode Retailer",
			receipt: Receipt{
				Retailer:     "Tárgét", // Accented characters, not alphanumeric
				PurchaseDate: "2022-01-01",
				PurchaseTime: "13:01",
				Items:        []Item{},
				TotalCents:   100,
			},
			expectedPoints: 50 + 6 + 4 + 25, // 50 (round) + 6 (odd day) + 6 (retailer) + 25 (multiple of 25)= 85 //CORRECT
		},
		{
			name: "Invalid Date", //Test case for an invalid date
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "invalid-date",
				PurchaseTime: "13:01",
				Items:        []Item{},
				TotalCents:   10,
			},
			expectedPoints: 12, // 12 (retailer) //CORRECT
		},
		{
			name: "Invalid Time", //Test case for an invalid time
			receipt: Receipt{
				Retailer:     "TestRetailer",
				PurchaseDate: "2024-01-01",
				PurchaseTime: "invalid-time",
				Items:        []Item{},
				TotalCents:   10,
			},
			expectedPoints: 6 + 12, // 6 (odd day) + 12 (retailer) //CORRECT
		},
		{
			name: "M&M Corner Market",
			receipt: Receipt{
				Retailer:     "M&M Corner Market", //14 points
				PurchaseDate: "2022-03-20",        //0 points
				PurchaseTime: "14:33",             //10 points
				Items: []Item{
					{ShortDescription: "Gatorade", PriceCents: 225},
					{ShortDescription: "Gatorade", PriceCents: 225},
					{ShortDescription: "Gatorade", PriceCents: 225},
					{ShortDescription: "Gatorade", PriceCents: 225},
				},
				TotalCents: 900, //50 points + 25 points
			},
			expectedPoints: 50 + 25 + 10 + 10 + 14, // 109 //CORRECT
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPoints := calculatePoints(tt.receipt)
			if actualPoints != tt.expectedPoints {
				t.Errorf("calculatePoints() = %v, want %v", actualPoints, tt.expectedPoints)
			}
		})
	}
}
