package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"golift.io/starr"
	"golift.io/starr/radarr"
	"golift.io/starr/sonarr"
)

type missingServiceError struct {
	missing string
}

func (e *missingServiceError) Error() string {
	return fmt.Sprintf("missing from .env: %s", e.missing)
}

type sonarrInfo struct {
	apiKey string
	host   string
	port   string
}

type radarrInfo struct {
	apiKey string
	host   string
	port   string
}

type Services struct {
	Radarr *radarr.Radarr
	Sonarr *sonarr.Sonarr
}

func start() (error, *Services) {
	err := godotenv.Load()
	if err != nil {
		return &missingServiceError{"file itself"}, nil
	}

	var (
		sonarrHost   = os.Getenv("SONARR_HOST")
		sonarrPort   = os.Getenv("SONARR_PORT")
		sonarrApiKey = os.Getenv("SONARR_API_KEY")
	)

	var (
		radarrHost   = os.Getenv("RADARR_HOST")
		radarrPort   = os.Getenv("RADARR_PORT")
		radarrApiKey = os.Getenv("RADARR_API_KEY")
	)

	switch "" {
	case sonarrHost:
		return &missingServiceError{"SONARR_HOST"}, nil
	case sonarrPort:
		return &missingServiceError{"SONARR_PORT"}, nil
	case sonarrApiKey:
		return &missingServiceError{"SONARR_API_KEY"}, nil
	case radarrHost:
		return &missingServiceError{"RADARR_HOST"}, nil
	case radarrPort:
		return &missingServiceError{"RADARR_PORT"}, nil
	case radarrApiKey:
		return &missingServiceError{"RADARR_API_KEY"}, nil
	}

	// port could be unset...
	c := starr.New(sonarrApiKey, sonarrHost+":"+sonarrPort, 0)
	s := sonarr.New(c)
	c = starr.New(radarrApiKey, radarrHost+":"+radarrPort, 0)
	r := radarr.New(c)

	return nil, &Services{
		Radarr: r,
		Sonarr: s,
	}
}

func (s *Services) findPath(item Deletion) (error, []string) {
	fmt.Printf("%v", item)
	var path []string
	if item.ItemType == "Movie" {
		lookup, err := s.Radarr.Lookup(item.Name + " " + item.Year)
		if err != nil {
			return err, path
		}
		for _, movie := range lookup {
			path = append(path, movie.Path)
		}
	} else if item.ItemType == "Movie" {
		lookup, err := s.Sonarr.Lookup(item.Name + " " + item.Year)
		if err != nil {
			return err, path
		}
		for _, series := range lookup {
			path = append(path, series.Path)
		}
	}

	return nil, path
}
