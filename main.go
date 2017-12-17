package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/zmb3/spotify"
	"io"
	"log"
	"net/http"
	"os"
)

type Options struct {
	WebServer bool
	Transfer  string
}

func Devices(sc *spotify.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices, err := sc.PlayerDevices()
		if err != nil {
			log.Printf("Error listing devices: %v", err)
		}

		b, _ := json.Marshal(devices)
		w.Write(b)
	}
}

type IdType struct {
	Id string
}

func Transfer(sc *spotify.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var t IdType
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("Error parsing request: %v", err)
		}

		log.Printf("Transfering %s...", t)
		err = sc.TransferPlayback(spotify.ID(t.Id), true)
		if err != nil {
			log.Fatalf("Error transfering playback: %v", err)
		}

		devices, err := sc.PlayerDevices()
		if err != nil {
			log.Fatalf("Error listing devices: %v", err)
		}

		b, _ := json.Marshal(devices)
		w.Write(b)
	}
}

func main() {
	o := Options{}

	flag.BoolVar(&o.WebServer, "web-server", true, "start web server")
	flag.StringVar(&o.Transfer, "transfer", "", "transfer playback to the specified device")

	flag.Parse()

	logFile, err := os.OpenFile("play-zones.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer logFile.Close()

	buffer := new(bytes.Buffer)

	multi := io.MultiWriter(logFile, buffer, os.Stdout)

	log.SetOutput(multi)

	sc, _ := AuthenticateSpotify()

	playerState, err := sc.PlayerState()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Found your %+v", playerState.Device)

	devices, err := sc.PlayerDevices()
	if err != nil {
		log.Fatalf("Error listing devices: %v", err)
	}

	for _, d := range devices {
		log.Printf("%+v", d)

		if o.Transfer != "" && d.Name == o.Transfer {
			if d.ID != playerState.Device.ID {
				err = sc.TransferPlayback(d.ID, true)
				if err != nil {
					log.Fatalf("Error transfering playback: %v", err)
				}
			}
		}
	}

	if o.WebServer {
		http.Handle("/spotify/", http.StripPrefix("/spotify", http.FileServer(http.Dir("./static"))))
		http.HandleFunc("/spotify/devices.json", Devices(sc))
		http.HandleFunc("/spotify/transfer.json", Transfer(sc))
		http.ListenAndServe(":9090", nil)
	}
}
