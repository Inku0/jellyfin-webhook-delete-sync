package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

const port = ":6666"

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
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("%v", err)
	}

	q, err := qbit()
	if err != nil {
		log.Fatalf("%v", err)
	}

	r, err := createRadarr()
	if err != nil {
		log.Fatalf("%v", err)
	}

	s, err := createSonarr()
	if err != nil {
		log.Fatalf("%v", err)
	}

	h := WebhookHandler{
		Radarr: r,
		Sonarr: s,
		QB:     q,
	}

	http.HandleFunc("/webhook", h.Handle)
	log.Fatal(http.ListenAndServe(port, nil))
}
