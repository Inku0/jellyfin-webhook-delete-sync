package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"sync"

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

func (h *WebhookHandler) findMovie(name string) ([]*radarr.Movie, error) {
	lookup, err := h.Radarr.Lookup(name)
	if err != nil {
		return nil, err
	}
	if len(lookup) == 0 {
		return nil, fmt.Errorf("found 0 results for \"%s\" in Radarr", name)
	}

	return lookup, nil
}

func (h *WebhookHandler) findSeries(name string) ([]*sonarr.Series, error) {
	all, err := h.Sonarr.GetAllSeries()
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("found 0 results for \"%s\" in Sonarr", name)
	}

	names := make([]string, 0, len(all))
	for _, series := range all {
		names = append(names, series.Title)
	}

	matches := Find(name, names)
	if len(matches) == 0 {
		return nil, fmt.Errorf("matched 0 results for \"%s\" in Sonarr", name)
	}

	serieses := make([]*sonarr.Series, 0, len(matches))
	for _, match := range matches {
		index := slices.Index(names, match.Str)
		serieses = append(serieses, all[index])
	}

	return serieses, nil
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

	var titles []string
	if service == "Radarr" {
		log.Printf("finding %s", p.Name+" "+p.Year)
		n, err := h.findMovie(p.Name + " " + p.Year)
		if err != nil {
			return err
		}
		//log.Printf("found %s as", p.Name+" "+p.Year)
		var ids []int64
		for _, movie := range n {
			ids = append(ids, movie.ID)
			titles = append(titles, movie.Title)
			fmt.Printf("%s\n", movie.Title)
		}

		tag, err := h.findTag("marked-for-death", service)
		if err != nil {
			return err
		}

		if os.Getenv("DRY_RUN") == "false" {
			_, err = h.Radarr.EditMovies(&radarr.BulkEdit{
				Monitored: starr.False(),
				MovieIDs:  ids,
				Tags:      []int{tag},
			})
			if err != nil {
				return err
			}
		}

	} else if service == "Sonarr" {
		log.Printf("finding %s", p.Name)
		n, err := h.findSeries(p.Name)
		if err != nil {
			return err
		}
		//log.Printf("found %s as %+v", p.Name)

		tag, err := h.findTag("marked-for-death", service)
		if err != nil {
			return err
		}

		for _, series := range n {
			tags := append(series.Tags, tag)
			titles = append(titles, series.Title)

			input := sonarr.AddSeriesInput{
				Monitored:         false,
				SeasonFolder:      series.SeasonFolder,
				UseSceneNumbering: series.UseSceneNumbering,
				ID:                series.ID,
				LanguageProfileID: series.LanguageProfileID,
				QualityProfileID:  series.QualityProfileID,
				TvdbID:            series.TvdbID,
				ImdbID:            series.ImdbID,
				TvMazeID:          series.TvMazeID,
				TvRageID:          series.TvRageID,
				Path:              series.Path,
				SeriesType:        series.SeriesType,
				Title:             series.Title,
				TitleSlug:         series.TitleSlug,
				RootFolderPath:    series.RootFolderPath,
				Tags:              tags,
				Seasons:           series.Seasons,
				Images:            series.Images,
			}
			//fmt.Printf("%+v", input)
			if os.Getenv("DRY_RUN") == "false" {
				_, err = h.Sonarr.UpdateSeries(&input, false)
				if err != nil {
					return err
				}
			}
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

	// source: https://gobyexample.com/worker-pools
	numJobs := len(torrents)

	nameHashes := make(map[string][]string, len(torrents))

	type result struct {
		title string
		hash  string
		err   error
	}

	jobs := make(chan qbt.TorrentInfo, numJobs)
	results := make(chan result, numJobs)

	workers := 8
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for torrent := range jobs {
				parsed, err := ptn.Parse(torrent.Name)
				if err != nil {
					results <- result{err: err}
					continue
				}
				results <- result{title: parsed.Title, hash: torrent.Hash}
			}
		}()
	}

	// wrap in go func() so that they can run concurrently
	// feed jobs
	go func() {
		for _, torrent := range torrents {
			jobs <- torrent
		}
		close(jobs)
	}()

	// close results after workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// this also run concurrently so a mutex is needed
	var mu sync.Mutex
	for res := range results {
		if res.err != nil {
			return res.err
		}
		mu.Lock()
		nameHashes[res.title] = append(nameHashes[res.title], res.hash)
		mu.Unlock()
	}

	log.Printf("parsed torrent names")

	names := slices.Sorted(maps.Keys(nameHashes))

	var matches Matches
	for _, title := range titles {
		matches = append(matches, Find(title, names)...) // is this legal? (...)
	}
	//matches := Find(name, names)
	if len(matches) == 0 {
		return fmt.Errorf("found no matches for %+v in qBittorrent", titles)
	}

	hashes := make([]string, 0)

	for _, match := range matches {
		log.Printf("matched %+v to %s with score %d", titles, match.Str, match.Score)
		hashes = append(hashes, nameHashes[match.Str]...)
	}

	log.Printf("adding tags to torrents")
	if os.Getenv("DRY_RUN") == "false" {
		result, err := h.QB.AddTorrentTags(hashes, []string{"marked-for-death"})
		if err != nil {
			return err
		} else if result == false {
			return fmt.Errorf("failed to add tags for hashes %+v, because: %v", hashes, err)
		}
	}

	log.Printf("success")
	return nil
}
