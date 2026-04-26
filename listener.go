package main

import (
	"log"
	"net/http"
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
	q, err := qbit()
	if err != nil {
		log.Fatalf("%v", err)
	}

	s, err := start()
	if err != nil {
		log.Fatalf("%v", err)
	}

	h := WebhookHandler{
		Radarr: s.Radarr,
		Sonarr: s.Sonarr,
		QB:     q,
	}

	http.HandleFunc("/webhook", h.Handle)
	log.Fatal(http.ListenAndServe(port, nil))
}
