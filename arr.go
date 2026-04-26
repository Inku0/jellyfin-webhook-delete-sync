package main

import (
	"os"

	"golift.io/starr"
	"golift.io/starr/radarr"
	"golift.io/starr/sonarr"
)

func createRadarr() (*radarr.Radarr, error) {
	var (
		radarrHost   = os.Getenv("RADARR_HOST")
		radarrPort   = os.Getenv("RADARR_PORT")
		radarrApiKey = os.Getenv("RADARR_API_KEY")
	)

	c := starr.New(radarrApiKey, radarrHost+":"+radarrPort, 0)
	r := radarr.New(c)

	return r, nil
}

func createSonarr() (*sonarr.Sonarr, error) {
	var (
		sonarrHost   = os.Getenv("SONARR_HOST")
		sonarrPort   = os.Getenv("SONARR_PORT")
		sonarrApiKey = os.Getenv("SONARR_API_KEY")
	)

	// port could be unset...
	c := starr.New(sonarrApiKey, sonarrHost+":"+sonarrPort, 0)
	s := sonarr.New(c)

	return s, nil
}
