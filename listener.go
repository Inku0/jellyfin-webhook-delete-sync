package main

import (
	"encoding/json"
	"io"
	"log"
	"maps"
	"net/http"
	"slices"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/razsteinmetz/go-ptn"
	"github.com/superturkey650/go-qbittorrent/qbt"
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
		if len(lookup) == 0 {
			log.Fatalf("failed to find match for %s in Radarr", payload.Name)
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

		//_, err = s.Radarr.EditMovies(&radarr.BulkEdit{
		//	MovieIDs:  []int64{lookup[0].ID},
		//	Monitored: starr.False(),
		//	Tags:      []int{tagID},
		//})
		//if err != nil {
		//	log.Fatalf("failed to unmonitor movie: %s: %s", payload.Name, err)
		//}
		name = lookup[0].Title

	case "Series":
		lookup, err := s.Sonarr.Lookup(payload.Name + " " + payload.Year)
		if err != nil {
			log.Fatalf("failed to look up series: %s", err)
		}
		if len(lookup) == 0 {
			log.Fatalf("failed to find match for %s in Sonarr", payload.Name)
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
		//
		//_, err = s.Sonarr.UpdateSeries(&sonarr.AddSeriesInput{
		//	Monitored: false,
		//	ID:        lookup[0].ID,
		//	Tags:      []int{tagID},
		//}, false)
		//if err != nil {
		//	log.Fatalf("failed to unmonitor series: %s: %s", payload.Name, err)
		//}

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
		Category: &category,
	})

	nameHashes := make(map[string]string, len(torrents))
	for _, torrent := range torrents {
		parsed, err := ptn.Parse(torrent.Name)
		if err != nil {
			log.Fatalf("failed to parse %s", torrent.Name)
		}
		nameHashes[parsed.Title] = torrent.Hash
	}

	names := slices.Sorted(maps.Keys(nameHashes))
	log.Printf("matching for %s in %v", name, names)
	for _, match := range fuzzy.RankFindNormalizedFold(name, names) {
		log.Printf("matched %s with distance %d", match.Target, match.Distance)
	}

	//result, err := qb.AddTorrentTags(hashes, []string{"marked-for-death"})
	//if err != nil || result == false {
	//	log.Fatalf("failed to add tags for hashes %v, because: %s", hashes, err)
	//	return
	//}
}

func save(str string) {

}
