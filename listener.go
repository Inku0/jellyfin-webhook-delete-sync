package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const port = ":666"

type Deletion struct {
	Name         string `json:"Name"`
	Username     string `json:"Username"`
	Deleted      string `json:"Deleted"`
	Timestamp    string `json:"Timestamp"`
	Year         string `json:"Year"`
	PremiereDate string `json:"PremiereDate"`
	ItemType     string `json:"ItemType"`
}

func main() {

	http.HandleFunc("/webhook", listener)
	log.Fatal(http.ListenAndServe(port, nil))

}

func listener(w http.ResponseWriter, r *http.Request) {
	// Source: https://medium.com/@rajamanohar.mummidi/implement-webhooks-in-go-d34f67196b43
	// Check if the request is a POST request
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Parse the payload JSON
	var payload Deletion
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Deleted == "true" {
		log.Printf("received deletion for: %s from user %s", payload.Name, payload.Username)
	}

	err, s := start()
	if err != nil {
		log.Fatal(err)
		return
	}

	s.find(payload)
	// Do something with the payload data...
}

func save(str string) {

}
