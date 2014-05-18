package main

import (
	"encoding/json"
	"errors"
	"os"
)

const portVar = "PORT"
const servicesVar = "VCAP_SERVICES"

var port string
var mgoURL string

func loadMongoDBUrlFromEnv() (string, error) {
	if s := os.Getenv(servicesVar); s != "" {

		var srv struct {
			UserProvied []struct {
				Credentials struct {
					Uri string
				}
			} `json:"user-provided"`
		}

		json.Unmarshal([]byte(s), &srv)

		return srv.UserProvied[0].Credentials.Uri, nil
	}
	return "", errors.New("mongoDB config not found")
}

func init() {
	if port = os.Getenv(portVar); port == "" {
		port = "8080"
	}
	var err error
	mgoURL, err = loadMongoDBUrlFromEnv()
	if err != nil {
		mgoURL = "localhost"
	}

}
