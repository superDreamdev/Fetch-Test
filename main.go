package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/mux"
    "github.com/google/uuid"
)

type Receipt struct {
    Retailer     string  `json:"retailer"`
    PurchaseDate string  `json:"purchaseDate"`
    PurchaseTime string  `json:"purchaseTime"`
    Items        []Item  `json:"items"`
    Total        string  `json:"total"`
}

type Item struct {
    ShortDescription string `json:"shortDescription"`
    Price            string `json:"price"`
}

type ReceiptResponse struct {
    ID string `json:"id"`
}

type PointsResponse struct {
    Points int `json:"points"`
}

var receipts = make(map[string]Receipt)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/receipts/process", processReceipt).Methods("POST")
    r.HandleFunc("/receipts/{id}/points", getPoints).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func processReceipt(w http.ResponseWriter, r *http.Request) {
    var receipt Receipt
    if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
        http.Error(w, "Invalid receipt", http.StatusBadRequest)
        return
    }

    id := uuid.New().String()
    receipts[id] = receipt

    response := ReceiptResponse{ID: id}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getPoints(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    receipt, exists := receipts[id]
    if !exists {
        http.Error(w, "No receipt found for that ID", http.StatusNotFound)
        return
    }

    points := calculatePoints(receipt)
    response := PointsResponse{Points: points}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func calculatePoints(receipt Receipt) int {
    points := 0

    // One point for every alphanumeric character in the retailer name.
    points += len(regexp.MustCompile(`[a-zA-Z0-9]`).FindAllString(receipt.Retailer, -1))

    // 50 points if the total is a round dollar amount with no cents.
    if strings.HasSuffix(receipt.Total, ".00") {
        points += 50
    }

    // 25 points if the total is a multiple of 0.25.
    total, _ := strconv.ParseFloat(receipt.Total, 64)
    if total*100%25 == 0 {
        points += 25
    }

    // 5 points for every two items on the receipt.
    points += (len(receipt.Items) / 2) * 5

    // If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer.
    for _, item := range receipt.Items {
        descLen := len(strings.TrimSpace(item.ShortDescription))
        if descLen%3 == 0 {
            price, _ := strconv.ParseFloat(item.Price, 64)
            points += int(price*0.2 + 0.9999)
        }
    }

    // 6 points if the day in the purchase date is odd.
    date, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
    if date.Day()%2 != 0 {
        points += 6
    }

    // 10 points if the time of purchase is after 2:00pm and before 4:00pm.
    time, _ := time.Parse("15:04", receipt.PurchaseTime)
    if time.Hour() == 14 {
        points += 10
    }

    return points
}