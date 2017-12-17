package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type SpotifyTokens struct {
	AccessToken  string
	RefreshToken string
	Expiry       string
	TokenType    string
}

type Tokens struct {
	Facebook string
	Spotify  SpotifyTokens
}

var globalTokens Tokens

func ReadTokens() (tokens *Tokens) {
	file, e := ioutil.ReadFile("./tokens.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	json.Unmarshal(file, &globalTokens)

	tokens = &globalTokens

	return
}

func WriteTokens(tokens *Tokens) {
	var tokensJson, err = json.Marshal(tokens)
	if err != nil {
		fmt.Printf("JSON error: %v\n", err)

	}

	err = ioutil.WriteFile("./tokens.json", tokensJson, 0644)
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}
}
