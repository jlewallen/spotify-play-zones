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

func (dc *DeviceChanger) Transfer(id spotify.ID) (ws *WebState, err error) {
	if !dc.IsPlaying(id) {
		err = dc.SpotifyClient.TransferPlayback(id, true)
		if err != nil {
			return nil, err
		}
	}

	return dc.WebState()
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

type Playing struct {
	Name    string
	Album   string
	Artists []string
}

type WebState struct {
	Devices []spotify.PlayerDevice
	Playing Playing
}

func getArtistNames(t *spotify.FullTrack) []string {
	names := make([]string, 0)
	for _, a := range t.Artists {
		names = append(names, a.Name)
	}
	return names
}

func (dc *DeviceChanger) WebState() (ws *WebState, err error) {
	playerState, err := dc.SpotifyClient.PlayerState()
	if err != nil {
		log.Printf("Error: %v", err)
	}
	log.Printf("%+v", playerState.Device.Name)
	item := playerState.CurrentlyPlaying.Item
	if item != nil {
		log.Printf("%+v", item.Album)
		log.Printf("%+v", item.Artists)
		log.Printf("%+v", item.Name)
	}

	devices, err := dc.SpotifyClient.PlayerDevices()
	if err != nil {
		log.Printf("Error: %v", err)
	}

	return &WebState{
		Devices: devices,
		Playing: Playing{
			Name:    item.Name,
			Album:   item.Album.Name,
			Artists: getArtistNames(item),
		},
	}, nil
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
		state, err := dc.WebState()
		if err != nil {
			log.Printf("Error listing devices: %v", err)
		}

		b, _ := json.Marshal(state)
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

		state, err := dc.Transfer(spotify.ID(t.Id))
		if err != nil {
			log.Fatalf("Error transfering playback: %v", err)
		}

		b, _ := json.Marshal(state)
		w.Write(b)
	}
}

func main() {
	o := Options{}

	flag.BoolVar(&o.WebServer, "web-server", false, "start web server")
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
	log.Printf("%+v", playerState)

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
