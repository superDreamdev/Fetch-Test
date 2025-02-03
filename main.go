package main

import (
	// "encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

var (
	receipts sync.Map
	mu       sync.Mutex
)

func main() {
	r := gin.Default()

	r.POST("/receipts/process", processReceipt)
	r.GET("/receipts/:id/points", getPoints)

	r.Run(":8080")
}

func processReceipt(c *gin.Context) {
	var receipt Receipt
	if err := c.ShouldBindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt format"})
		return
	}

	points := calculatePoints(receipt)
	id := uuid.New().String()

	mu.Lock()
	receipts.Store(id, points)
	mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func getPoints(c *gin.Context) {
	id := c.Param("id")
	points, ok := receipts.Load(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"points": points})
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// Rule 1: Alphanumeric characters in retailer name
	points += len(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(receipt.Retailer, ""))

	// Rule 2: Round dollar amount
	total, _ := parseMoney(receipt.Total)
	if total == math.Trunc(total) {
		points += 50
	}

	// Rule 3: Multiple of 0.25
	if math.Mod(total, 0.25) == 0 {
		points += 25
	}

	// Rule 4: 5 points per two items
	points += (len(receipt.Items) / 2) * 5

	// Rule 5: Item description length multiple of 3
	for _, item := range receipt.Items {
		desc := strings.TrimSpace(item.ShortDescription)
		if len(desc)%3 == 0 {
			price, _ := parseMoney(item.Price)
			points += int(math.Ceil(price * 0.2))
		}
	}

	// Rule 6: LLM-generated code bonus
	if total > 10.0 {
		points += 5
	}

	// Rule 7: Odd purchase day
	purchaseDate, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if purchaseDate.Day()%2 != 0 {
		points += 6
	}

	// Rule 8: Purchase time between 2pm and 4pm
	purchaseTime, _ := time.Parse("15:04", receipt.PurchaseTime)
	start, _ := time.Parse("15:04", "14:00")
	end, _ := time.Parse("15:04", "16:00")
	if purchaseTime.After(start) && purchaseTime.Before(end) {
		points += 10
	}

	return points
}

func parseMoney(s string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}