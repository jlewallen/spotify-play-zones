package main

import (
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	authenticator = spotify.NewAuthenticator(spotifyRedirectUrl, spotify.ScopePlaylistModifyPrivate, spotify.ScopePlaylistModifyPublic, spotify.ScopeUserLibraryModify, spotify.ScopeUserReadPrivate, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	clientChannel = make(chan *spotify.Client)
)

func AuthenticateSpotify() (spotifyClient *spotify.Client, err error) {
	var tokens = ReadTokens()

	log.Printf("Authenticating with Spotify...")

	if tokens.Spotify.AccessToken == "" {
		http.HandleFunc("/spotify/callback", CompleteAuth)
		go http.ListenAndServe(":9090", nil)

		authenticator.SetAuthInfo(spotifyClientId, spotifyClientSecret)

		url := authenticator.AuthURL(spotifyOauthStateString)
		log.Println("Please log in to Spotify by visiting the following page in your browser:", url)

		spotifyClient = <-clientChannel
	} else {
		var oauthToken oauth2.Token
		oauthToken.AccessToken = tokens.Spotify.AccessToken
		oauthToken.RefreshToken = tokens.Spotify.RefreshToken
		oauthToken.Expiry, _ = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", tokens.Spotify.Expiry)
		oauthToken.TokenType = tokens.Spotify.TokenType
		newClient := authenticator.NewClient(&oauthToken)
		authenticator.SetAuthInfo(spotifyClientId, spotifyClientSecret)
		spotifyClient = &newClient
	}

	user, err := spotifyClient.CurrentUser()
	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	log.Println("spotify: You are logged in as", user.ID)

	return
}

func CompleteAuth(w http.ResponseWriter, r *http.Request) {
	token, err := authenticator.Token(spotifyOauthStateString, r)
	if err != nil {
		http.Error(w, "Unable to get token", http.StatusForbidden)
		log.Fatal(err)
	}

	if actualState := r.FormValue("state"); actualState != spotifyOauthStateString {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", actualState, spotifyOauthStateString)
	}

	var tokens = ReadTokens()
	tokens.Spotify.AccessToken = token.AccessToken
	tokens.Spotify.RefreshToken = token.RefreshToken
	tokens.Spotify.Expiry = token.Expiry.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	tokens.Spotify.TokenType = token.TokenType
	WriteTokens(tokens)

	client := authenticator.NewClient(token)
	clientChannel <- &client
}
