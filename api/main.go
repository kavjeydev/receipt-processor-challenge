package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price string `json:"price"`
}

type Reciept struct {
	ID string `json:"id"`
	Retailer string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Total string `json:"total"`
	Items []Item `json:"items"`
}

var receipts = []Reciept{}

func generateId() string {
	id := uuid.New()
	return id.String()
}

func getAlphaNumPoints(retailer string) int{
	points := 0
	for _, r := range retailer {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			points += 1
		}
	}
	return points
}

func checkRoundNumber(total string) int{
	cents := strings.Split(total, ".")[1]
	if (cents ==  "00") {
		return 50;
	}
	return 0;
}

func checkMultiple(total string) int {
	cents := strings.Split(total, ".")[1]

	num, err := strconv.Atoi(cents)

	if err != nil {
		fmt.Println("Error converting string:", err)
	}

	if (num % 25 == 0) {
		return 25
	}
	return 0
}

func descriptionPoints(items []Item) int {
	var total_points int = 0
	for _, item := range items {
		description := strings.TrimSpace(item.ShortDescription)
		price := item.Price
		if (len(description) % 3 == 0){
			priceNum, err := strconv.ParseFloat(price, 64)
			if err != nil {
				fmt.Println("Error converting string:", err)
			}
			total_points += int(math.Ceil((priceNum * 0.2)))
		}
	}

	return total_points
}

func purchaseDatePoints(purchaseDate string) int {
	dateAsNum, err := strconv.Atoi(strings.Split(purchaseDate, "-")[2])
	if err != nil {
		fmt.Println("Error converting string:", err)
	}
	if (dateAsNum % 2 == 1) {
		return 6
	}
	return 0
}

func purchaseTimePoints(purchaseTime string) int {
	hour, herr:= strconv.Atoi(strings.Split(purchaseTime, ":")[0])
	minute, merr := strconv.Atoi(strings.Split(purchaseTime, ":")[1])

	if herr != nil || merr != nil {
		fmt.Println("Error converting string")
	}

	if ((hour >= 14 && minute >= 1) && (hour < 16 && minute <= 59) ) {
		return 10
	}
	return 0
}

func processReciept(w http.ResponseWriter, r *http.Request) {
	if (r.Method != "POST") {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var recieptPayload Reciept
	var recieptId string = generateId()
	err := json.NewDecoder(r.Body).Decode(&recieptPayload)

	if err != nil {
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"id": recieptId,
	}

	recieptPayload.ID = recieptId
	receipts = append(receipts, recieptPayload)

	json.NewEncoder(w).Encode(response)
}

func countItemsPoints(items []Item) float64 {
	points := math.Floor(float64(len(items)) / 2) * 5
	return points
}

func getTotalPoints(reciept Reciept) int {
	totalPoints := 0
	totalPoints += getAlphaNumPoints(reciept.Retailer)
	totalPoints += checkRoundNumber(reciept.Total)
	totalPoints += checkMultiple(reciept.Total)
	totalPoints += descriptionPoints(reciept.Items)
	totalPoints += purchaseDatePoints(reciept.PurchaseDate)
	totalPoints += purchaseTimePoints(reciept.PurchaseTime)
	totalPoints += int(countItemsPoints(reciept.Items))

	return totalPoints
}

func getPoints (w http.ResponseWriter, r *http.Request) {
	if (r.Method != "GET") {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}

	vars := mux.Vars(r)
    receiptID := vars["id"]

	if receiptID == "" {
		http.Error(w, "No ID found", http.StatusBadRequest)
		return
	}

	var foundReceipt *Reciept
    for _, receipt := range receipts {
        if receipt.ID == receiptID {
            foundReceipt = &receipt
            break
        }
    }

    if foundReceipt == nil {
        http.Error(w, "Receipt not found", http.StatusNotFound)
        return
    }

	totalPoints := getTotalPoints(*foundReceipt)

	response := map[string]int{
		"points": totalPoints,
	}

	json.NewEncoder(w).Encode(response)

}

func main() {
	r := mux.NewRouter()
	http.HandleFunc("/reciepts/process", processReciept)
	r.HandleFunc("/receipts/{id}/points", getPoints).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}