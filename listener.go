package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/superturkey650/go-qbittorrent/qbt"
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

	s, err := start()
	if err != nil {
		log.Fatal(err)
		return
	}

	var name string

	switch payload.ItemType {
	case "Movie":
		lookup, err := s.Radarr.Lookup(payload.Name + " " + payload.Year)
		if err != nil {
			log.Fatalf("failed to look up movie: %s: %s", payload.Name, err)
		}
		//log.Printf("%+v", lookup[0])

		tags, err := s.Radarr.GetTags()
		if err != nil {
			log.Fatalf("failed to get tags from Radarr: %s", err)
		}

		var tagID int
		for _, tag := range tags {
			if tag.Label == "marked-for-death" {
				tagID = tag.ID
				log.Printf("found ID for marked-for-death: %d", tagID)
			}
		}

		_, err = s.Radarr.EditMovies(&radarr.BulkEdit{
			MovieIDs:  []int64{lookup[0].ID},
			Monitored: starr.False(),
			Tags:      []int{tagID},
		})
		if err != nil {
			log.Fatalf("failed to unmonitor movie: %s: %s", payload.Name, err)
		}

		name = lookup[0].Title

	case "Series":
		lookup, err := s.Sonarr.Lookup(payload.Name + " " + payload.Year)
		if err != nil {
			log.Fatalf("failed to look up series: %s", err)
		}
		//log.Printf("%+v", lookup[0])

		tags, err := s.Sonarr.GetTags()
		if err != nil {
			log.Fatalf("failed to get tags from Sonarr: %s", err)
		}

		var tagID int
		for _, tag := range tags {
			if tag.Label == "marked-for-death" {
				log.Printf("found ID for marked-for-death: %d", tagID)
				tagID = tag.ID
			}
		}

		_, err = s.Sonarr.UpdateSeries(&sonarr.AddSeriesInput{
			Monitored: false,
			ID:        lookup[0].ID,
			Tags:      []int{tagID},
		}, false)
		if err != nil {
			log.Fatalf("failed to unmonitor series: %s: %s", payload.Name, err)
		}

		name = lookup[0].Title
	}

	qb, err := qbit()
	if err != nil {
		log.Fatal(err)
		return
	}

	var category string
	if payload.ItemType == "Movie" {
		category = "movies"
	} else if payload.ItemType == "Series" {
		category = "tv"
	}
	log.Printf("%s from %s", name, category)

	torrents, err := qb.Torrents(qbt.TorrentsOptions{
		Filter:   &name,
		Category: &category,
	})

	var hashes []string
	for _, torrent := range torrents {
		log.Printf("%s", torrent.Name)
		hashes = append(hashes, torrent.Hash)
	}

	result, err := qb.AddTorrentTags(hashes, []string{"marked-for-death"})
	if err != nil || result == false {
		log.Fatalf("failed to add tags for hashes %v, because: %s", hashes, err)
		return
	}
}

func save(str string) {

}
