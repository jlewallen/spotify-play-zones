package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zmb3/spotify"
	"io"
	"log"
	"net/http"
	"os"
)

type DeviceChanger struct {
	SpotifyClient *spotify.Client
}

func (dc *DeviceChanger) Transfer(base string, id spotify.ID) (ws *WebState, err error) {
	if !dc.IsPlaying(id) {
		err = dc.SpotifyClient.TransferPlayback(id, true)
		if err != nil {
			return nil, err
		}
	}

	return dc.WebState(base)
}

func (dc *DeviceChanger) TransferByName(base string, name string) (ws *WebState, err error) {
	devices, err := dc.Devices()
	if err == nil {
		for _, d := range devices {
			if d.Name == name {
				return dc.Transfer(base, d.ID)
			}
		}
	}

	return dc.WebState(base)
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
	URLs    []string
}

func getArtistNames(t *spotify.FullTrack) []string {
	names := make([]string, 0)
	if t == nil {
		return names
	}
	for _, a := range t.Artists {
		names = append(names, a.Name)
	}
	return names
}

func detectBase(r *http.Request) string {
	return baseUrl
}

func getTransferUrls(base string, devices []spotify.PlayerDevice) []string {
	urls := make([]string, 0)
	for _, a := range devices {
		urls = append(urls, fmt.Sprintf("%s/transfer/tag?id=%s&name=%s", base, a.ID, a.Name))
	}
	return urls
}

func (dc *DeviceChanger) WebState(base string) (ws *WebState, err error) {
	playerState, err := dc.SpotifyClient.PlayerState()
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, fmt.Errorf("Error getting state: %v", err)
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
		return nil, fmt.Errorf("Error getting state: %v", err)
	}

	return &WebState{
		Devices: devices,
		URLs:    getTransferUrls(base, devices),
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
		state, err := dc.WebState(detectBase(r))
		if err != nil {
			log.Printf("Error listing devices: %v", err)
			http.Error(w, fmt.Sprintf("Error listing devices: %v", err), http.StatusInternalServerError)
			return
		}

		b, _ := json.Marshal(state)
		w.Header().Set("Content-Type", "application/json")
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
			http.Error(w, fmt.Sprintf("Error parsing request: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Transfering %s...", t)

		state, err := dc.Transfer(detectBase(r), spotify.ID(t.Id))
		if err != nil {
			log.Printf("Error transfering playback: %v", err)
			http.Error(w, fmt.Sprintf("Error transfering playback: %v", err), http.StatusInternalServerError)
			return
		}

		b, _ := json.Marshal(state)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func TagTransfer(dc *DeviceChanger, tokens []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(tokens) > 0 {
			given := r.URL.Query()["token"]
			if given == nil {
				log.Printf("No token, even though one is required.")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			valid := false
			for _, t := range tokens {
				if given[0] == t {
					valid = true
					break
				}
			}
			if !valid {
				log.Printf("Invalid token")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			log.Printf("Token is good!")
		}
		id := r.URL.Query()["id"]
		name := r.URL.Query()["name"]
		if id != nil && len(id) == 1 {
			log.Printf("Transfering %s...", id)

			_, err := dc.Transfer(detectBase(r), spotify.ID(id[0]))
			if err != nil {
				log.Printf("Error transfering playback: %v", err)
				http.Error(w, fmt.Sprintf("Error transfering playback: %v", err), http.StatusInternalServerError)
				return
			}
		} else if name != nil && len(name) == 1 {
			log.Printf("Transfering %s...", name)

			_, err := dc.TransferByName(detectBase(r), name[0])
			if err != nil {
				log.Printf("Error transfering playback: %v", err)
				http.Error(w, fmt.Sprintf("Error transfering playback: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Nowhere given to transfer to")
		}

		log.Printf("Redirect!")
		http.Redirect(w, r, "/spotify", 307)
	}
}

func main() {
	o := Options{}

	flag.BoolVar(&o.WebServer, "web-server", false, "start web server")
	flag.StringVar(&o.Transfer, "transfer", "", "transfer playback to the specified device")

	flag.Parse()

	log.Printf("%+v", o)

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
		http.HandleFunc("/spotify/transfer/tag/token", TagTransfer(dc, tokens))
		http.HandleFunc("/spotify/transfer/tag", TagTransfer(dc, make([]string, 0)))
		err = http.ListenAndServe(":9090", nil)
		if err != nil {
			log.Fatalf("Unable to start server: %v", err)
		}
	}
}
