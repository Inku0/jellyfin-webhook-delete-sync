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

func qbit() (*qbt.Client, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	var (
		qbitHost     = os.Getenv("QBIT_HOST")
		qbitPort     = os.Getenv("QBIT_PORT")
		qbitUsername = os.Getenv("QBIT_USERNAME")
		qbitPassword = os.Getenv("QBIT_PASSWORD")
	)

	qb := qbt.NewClient(qbitHost + ":" + qbitPort)

	err = qb.Login(qbitUsername, qbitPassword)
	if err != nil {
		return nil, err
	}

	return qb, nil
}
