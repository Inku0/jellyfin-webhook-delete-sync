package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"golift.io/starr"
	"golift.io/starr/radarr"
	"golift.io/starr/sonarr"
)

type Services struct {
	Radarr *radarr.Radarr
	Sonarr *sonarr.Sonarr
}

func start() (*Services, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
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

	// port could be unset...
	c := starr.New(sonarrApiKey, sonarrHost+":"+sonarrPort, 0)
	s := sonarr.New(c)
	c = starr.New(radarrApiKey, radarrHost+":"+radarrPort, 0)
	r := radarr.New(c)

	return &Services{
		Radarr: r,
		Sonarr: s,
	}, nil
}

func (s *Services) find(item Deletion) (error, int64) {
	// deprecated
	return fmt.Errorf("didn't find %s %s", item.ItemType, item.Name), 0
}
