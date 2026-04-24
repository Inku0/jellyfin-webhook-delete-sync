package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"golift.io/starr"
	"golift.io/starr/radarr"
	"golift.io/starr/sonarr"
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

	switch payload.ItemType {
	case "Movie":
		lookup, err := s.Radarr.Lookup(payload.Name + " " + payload.Year)
		if err != nil {
			log.Fatalf("failed to look up movie: %s: %s", payload.Name, err)
		}
		log.Printf("%+v", lookup[0])
		_, err = s.Radarr.EditMovies(&radarr.BulkEdit{
			MovieIDs:  []int64{lookup[0].ID},
			Monitored: starr.False(),
		})
		if err != nil {
			log.Fatalf("failed to unmonitor movie: %s: %s", payload.Name, err)
		}

	case "Series":
		lookup, err := s.Sonarr.Lookup(payload.Name + " " + payload.Year)
		if err != nil {
			log.Fatalf("failed to look up series: %s", err)
		}
		log.Printf("%+v", lookup[0])
		_, err = s.Sonarr.UpdateSeries(&sonarr.AddSeriesInput{
			Monitored: false,
			ID:        lookup[0].ID,
		}, false)
		if err != nil {
			log.Fatalf("failed to unmonitor movie: %s: %s", payload.Name, err)
		}
	}

	// Do something with the payload data...
}

func save(str string) {

}
