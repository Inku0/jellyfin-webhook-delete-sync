package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"slices"

	"github.com/razsteinmetz/go-ptn"
	"github.com/superturkey650/go-qbittorrent/qbt"
	"golift.io/starr"
	"golift.io/starr/radarr"
	"golift.io/starr/sonarr"
)

type WebhookHandler struct {
	Radarr *radarr.Radarr // or whatever type start() returns
	Sonarr *sonarr.Sonarr
	QB     *qbt.Client
}

var processDeletion = func(h *WebhookHandler, p Deletion) error {
	return h.processp(p)
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Source: https://medium.com/@rajamanohar.mummidi/implement-webhooks-in-go-d34f67196b43
	if r.Method != http.MethodPost {
		// request has to be a POST request
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var p Deletion
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if p.Deleted == "true" {
		log.Printf("received deletion for: %s from user %s", p.Name, p.Username)
	}

	err = processDeletion(h, p)
	if err != nil {
		log.Printf("error processing p: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *WebhookHandler) processp(p Deletion) error {
	var name string

	switch p.ItemType {
	case "Movie":
		lookup, err := h.Radarr.Lookup(p.Name + " " + p.Year)
		if err != nil {
			return err
		}
		if len(lookup) == 0 {
			return fmt.Errorf("failed to find match for %s in Radarr", p.Name)
		}
		//log.Printf("%+v", lookup[0])

		tags, err := h.Radarr.GetTags()
		if err != nil {
			return err
		}

		var tagID int
		for _, tag := range tags {
			if tag.Label == "marked-for-death" {
				tagID = tag.ID
				log.Printf("found ID for marked-for-death: %d", tagID)
			}
		}

		_, err = h.Radarr.EditMovies(&radarr.BulkEdit{
			MovieIDs:  []int64{lookup[0].ID},
			Monitored: starr.False(),
			Tags:      []int{tagID},
		})
		if err != nil {
			return err
		}
		name = lookup[0].Title

	case "Series":
		lookup, err := h.Sonarr.Lookup(p.Name + " " + p.Year)
		if err != nil {
			return err
		}
		if len(lookup) == 0 {
			return fmt.Errorf("failed to find match for %s in Sonarr", p.Name)
		}
		//log.Printf("%+v", lookup[0])

		tags, err := h.Sonarr.GetTags()
		if err != nil {
			return err
		}

		var tagID int
		for _, tag := range tags {
			if tag.Label == "marked-for-death" {
				log.Printf("found ID for marked-for-death: %d", tagID)
				tagID = tag.ID
			}
		}

		_, err = h.Sonarr.UpdateSeries(&sonarr.AddSeriesInput{
			Monitored: false,
			ID:        lookup[0].ID,
			Tags:      []int{tagID},
		}, false)
		if err != nil {
			return err
		}

		name = lookup[0].Title
	default:
		return fmt.Errorf("unsupported ItemType: %s", p.ItemType)
	}

	var category string
	if p.ItemType == "Movie" {
		category = "movies"
	} else if p.ItemType == "Series" {
		category = "tv"
	}
	log.Printf("%s from %s", name, category)

	torrents, err := h.QB.Torrents(qbt.TorrentsOptions{
		Category: &category,
	})
	if err != nil {
		return err
	}

	nameHashes := make(map[string][]string, len(torrents))
	for _, torrent := range torrents {
		parsed, err := ptn.Parse(torrent.Name)
		if err != nil {
			return err
		}
		//fmt.Printf("%v for %v\n", parsed.Title, torrent.Name)
		nameHashes[parsed.Title] = append(nameHashes[parsed.Title], torrent.Hash)
	}

	names := slices.Sorted(maps.Keys(nameHashes))
	//log.Printf("matching for %s in", name)
	//for _, n := range names {
	//	fmt.Printf("%s\n", n)
	//}

	matches := Find(name, names)
	if len(matches) == 0 {
		return fmt.Errorf("matched nothing for %s", name)
	}

	hashes := make([]string, 0)

	for _, match := range matches {
		log.Printf("matched %s with score %d", match.Str, match.Score)
		for _, p := range nameHashes[match.Str] {
			hashes = append(hashes, p)
		}
	}

	result, err := h.QB.AddTorrentTags(hashes, []string{"marked-for-death"})
	if err != nil {
		return err
	} else if result == false {
		return fmt.Errorf("failed to add tags for hashes %v, because: %v", hashes, err)
	}

	return nil
}
