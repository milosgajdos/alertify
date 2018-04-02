package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/zmb3/spotify"
)

// redirectURI is the OAuth redirect URI for the application.
const redirectURI = "http://localhost:8080/callback"

// authHandler handles OAuth2 authentication callback from Spotify API
func authHandler(auth *spotify.Authenticator, state string, ch chan *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err := auth.Token(state, r)
		if err != nil {
			http.Error(w, "Couldn't retrieve OAauth2 token", http.StatusForbidden)
			log.Fatal(err)
		}
		if s := r.FormValue("state"); s != state {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", s, state)
		}
		// use the token to get an authenticated client
		client := auth.NewClient(tok)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Login successful!")
		ch <- &client
	})
}

// configHandler returns configuration encoded in JSON
func configHandler(client *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// use the client to make calls that require authorization
		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("You are logged in as: %s\n", user.ID)

		devices, err := client.PlayerDevices()
		if err != nil {
			log.Printf("Error %v\n", err)
			log.Fatal(err)
		}
		for i, device := range devices {
			log.Printf("Spotify device %d: %v\n", i, device.Name)
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "User: %s", user.ID)
	})
}

// alertHandler starts playing configured music
func alertHandler(client *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID := new(spotify.ID)
		*deviceID = "f6e2bbe128cd0b9b5137ee18dd5afdc34b6a2598"
		opts := &spotify.PlayOptions{
			DeviceID: deviceID,
			URIs:     []spotify.URI{"spotify:track:2xYlyywNgefLCRDG8hlxZq"},
		}
		if err := client.PlayOpt(opts); err != nil {
			log.Printf("Couldnt play song: %v\n", err)
			http.Error(w, "Couldnt play song", http.StatusServiceUnavailable)
			return
		}
	})
}

// stopHandler stops playing the alert song
func stopHandler(client *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID := new(spotify.ID)
		*deviceID = "f6e2bbe128cd0b9b5137ee18dd5afdc34b6a2598"
		opts := &spotify.PlayOptions{
			DeviceID: deviceID,
		}
		if err := client.PauseOpt(opts); err != nil {
			log.Printf("Couldnt pause song: %v\n", err)
			http.Error(w, "Couldnt play song", http.StatusServiceUnavailable)
			return
		}
	})
}

func main() {
	auth := spotify.NewAuthenticator(redirectURI,
		spotify.ScopeUserReadCurrentlyPlaying,
		spotify.ScopeUserReadPlaybackState,
		spotify.ScopeUserModifyPlaybackState)
	state := "abc123"
	clientChan := make(chan *spotify.Client)
	errChan := make(chan error, 1)

	// Create HTTP muxer
	h := http.NewServeMux()

	h.Handle("/callback", authHandler(&auth, state, clientChan))

	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	go func() {
		errChan <- http.ListenAndServe(":8080", h)
	}()

	url := auth.AuthURL(state)
	fmt.Println("Log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-clientChan

	// register new http handlers
	h.Handle("/config", configHandler(client))
	h.Handle("/alert", alertHandler(client))
	h.Handle("/stop", stopHandler(client))

	err := <-errChan
	log.Fatal(err)
}
