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
	Radarr *radarr.Radarr
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
		err = processDeletion(h, p)
		if err != nil {
			log.Printf("error processing p: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("received a request which was not for a deletion from %s", p.Username)
	}
}

func (h *WebhookHandler) findMovie(name string) (*radarr.Movie, error) {
	lookup, err := h.Radarr.Lookup(name)
	if err != nil {
		return nil, err
	}
	if len(lookup) == 0 {
		return nil, fmt.Errorf("found 0 results for \"%s\" in Radarr", name)
	}

	return lookup[0], nil
}

func (h *WebhookHandler) findSeries(name string) (*sonarr.Series, error) {
	lookup, err := h.Sonarr.Lookup(name)
	if err != nil {
		return nil, err
	}
	if len(lookup) == 0 {
		return nil, fmt.Errorf("found 0 results for \"%s\" in Sonarr", name)
	}

	return lookup[0], nil
}

func (h *WebhookHandler) findTag(name string, service string) (int, error) {
	// -1 indicates that the tag wasn't found
	var tags []*starr.Tag
	if service == "Radarr" {
		t, err := h.Radarr.GetTags()
		if err != nil {
			return -1, err
		}
		tags = t
	} else if service == "Sonarr" {
		t, err := h.Sonarr.GetTags()
		if err != nil {
			return -1, err
		}
		tags = t
	}

	for _, tag := range tags {
		if tag.Label == name {
			log.Printf("found ID for %s: %d", name, tag.ID)
			return tag.ID, nil
		}
	}

	return -1, fmt.Errorf("found no results for tag \"%s\"", name)
}

func (h *WebhookHandler) processp(p Deletion) error {
	var service string
	if p.ItemType == "Movie" {
		service = "Radarr"
	} else if p.ItemType == "Series" {
		service = "Sonarr"
	} else {
		return fmt.Errorf("unsupported ItemType: %s", p.ItemType)
	}

	var name string
	if service == "Radarr" {
		log.Printf("finding %s", p.Name+" "+p.Year)
		n, err := h.findMovie(p.Name + " " + p.Year)
		if err != nil {
			return err
		}
		name = n.Title
		log.Printf("found %s as %s", p.Name+" "+p.Year, name)

		tag, err := h.findTag("marked-for-death", service)
		if err != nil {
			return err
		}

		_, err = h.Radarr.EditMovies(&radarr.BulkEdit{
			Monitored: starr.False(),
			MovieIDs:  []int64{n.ID},
			Tags:      []int{tag},
		})
		if err != nil {
			return err
		}

	} else if service == "Sonarr" {
		log.Printf("finding %s", p.Name+" "+p.Year)
		n, err := h.findSeries(p.Name + " " + p.Year)
		if err != nil {
			return err
		}
		name = n.Title
		log.Printf("found %s as %s", p.Name+" "+p.Year, name)

		tag, err := h.findTag("marked-for-death", service)
		if err != nil {
			return err
		}

		tags := append(n.Tags, tag)

		input := sonarr.AddSeriesInput{
			Monitored:         false,
			SeasonFolder:      n.SeasonFolder,
			UseSceneNumbering: n.UseSceneNumbering,
			ID:                n.ID,
			LanguageProfileID: n.LanguageProfileID,
			QualityProfileID:  n.QualityProfileID,
			TvdbID:            n.TvdbID,
			ImdbID:            n.ImdbID,
			TvMazeID:          n.TvMazeID,
			TvRageID:          n.TvRageID,
			Path:              n.Path,
			SeriesType:        n.SeriesType,
			Title:             n.Title,
			TitleSlug:         n.TitleSlug,
			RootFolderPath:    n.RootFolderPath,
			Tags:              tags,
			Seasons:           n.Seasons,
			Images:            n.Images,
		}
		fmt.Printf("%+v", input)
		_, err = h.Sonarr.UpdateSeries(&input, false)
		if err != nil {
			return err
		}
	}

	var category string
	if service == "Radarr" {
		category = "movies"
	} else if service == "Sonarr" {
		category = "tv"
	}

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
		nameHashes[parsed.Title] = append(nameHashes[parsed.Title], torrent.Hash)
	}
	log.Printf("parsed torrent names")

	names := slices.Sorted(maps.Keys(nameHashes))

	matches := Find(name, names)
	if len(matches) == 0 {
		return fmt.Errorf("found no matches for %s in qBittorrent", name)
	}

	hashes := make([]string, 0)

	for _, match := range matches {
		log.Printf("matched %s to %s with score %d", name, match.Str, match.Score)
		hashes = append(hashes, nameHashes[match.Str]...)
	}

	log.Printf("adding tags to torrents")
	result, err := h.QB.AddTorrentTags(hashes, []string{"marked-for-death"})
	if err != nil {
		return err
	} else if result == false {
		return fmt.Errorf("failed to add tags for hashes %+v, because: %v", hashes, err)
	}

	log.Printf("success")
	return nil
}
