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

type DeviceChanger struct {
	SpotifyClient *spotify.Client
}

func (dc *DeviceChanger) Transfer(id spotify.ID) (d []spotify.PlayerDevice, err error) {
	if !dc.IsPlaying(id) {
		err = dc.SpotifyClient.TransferPlayback(id, true)
		if err != nil {
			return nil, err
		}
	}

	return dc.Devices()
}

func (dc *DeviceChanger) IsPlaying(id spotify.ID) bool {
	devices, err := dc.Devices()
	if err == nil {
		for _, d := range devices {
			if d.ID == id && d.Active {
				return true
			}
		}
	}
	return false
}

func (dc *DeviceChanger) Devices() (d []spotify.PlayerDevice, err error) {
	return dc.SpotifyClient.PlayerDevices()
}

type Options struct {
	WebServer bool
	Transfer  string
}

func Devices(dc *DeviceChanger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices, err := dc.Devices()
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

func Transfer(dc *DeviceChanger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var t IdType
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("Error parsing request: %v", err)
		}

		log.Printf("Transfering %s...", t)
		devices, err := dc.Transfer(spotify.ID(t.Id))
		if err != nil {
			log.Fatalf("Error transfering playback: %v", err)
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

	dc := &DeviceChanger{
		SpotifyClient: sc,
	}

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
		http.HandleFunc("/spotify/devices.json", Devices(dc))
		http.HandleFunc("/spotify/transfer.json", Transfer(dc))
		http.ListenAndServe(":9090", nil)
	}
}
