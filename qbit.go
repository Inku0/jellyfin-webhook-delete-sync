package main

import (
	"os"

	"github.com/superturkey650/go-qbittorrent/qbt"

	"github.com/joho/godotenv"
)

type qbittorrentInfo struct {
	username string
	password string
	host     string
	port     string
}

func qbit() (error, *qbt.Client) {
	err := godotenv.Load()
	if err != nil {
		return err, nil
	}

	qb := qbt.NewClient(os.Getenv("QBIT_HOST") + ":" + os.Getenv("QBIT_PORT"))

	err = qb.Login(os.Getenv("QBIT_USERNAME"), os.Getenv("QBIT_PASSWORD"))
	if err != nil {
		return err, nil
	}

	return nil, qb
}
