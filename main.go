package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zmb3/spotify"
)

const (
	cliname = "alertify"
)

var (
	// redirectURI is the OAuth redirect URI for the application.
	redirectURI string
	// deviceName is Spotify client device name
	deviceName string
	// deviceID is Spotify client device ID
	deviceID string
	// songURI is Spotify song URI
	songURI string
)

func init() {
	flag.StringVar(&redirectURI, "redirect-uri", "http://localhost:8080/callback", "OAuth app redirect URI set in Spotify API console")
	flag.StringVar(&deviceName, "device-name", "", "Spotify device name as recognised by Spotify API")
	flag.StringVar(&deviceID, "device-id", "", "Spotify device ID as recognised by Spotify API")
	flag.StringVar(&songURI, "song-uri", "spotify:track:2xYlyywNgefLCRDG8hlxZq", "Spotify song URI")
	// disable timestamps and set prefix
	log.SetFlags(0)
	log.SetPrefix("[ " + cliname + " ] ")
}

// authHandler handles OAuth2 authentication callback from Spotify API
func authHandler(auth *Auth, ch chan *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err := auth.Token(auth.State, r)
		if err != nil {
			http.Error(w, "Couldn't retrieve OAuth token", http.StatusForbidden)
			log.Fatal(err)
		}
		if s := r.FormValue("state"); s != auth.State {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", s, auth.State)
		}
		// use the token to get an authenticated client
		client := auth.NewClient(tok)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Spotify login successful!")
		ch <- &client
	})
}

// configHandler returns configuration encoded in JSON
func configHandler(client *Client) http.Handler {
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
func alertHandler(client *Client, songURI string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := client.PlaySong(songURI); err != nil {
			log.Printf("Error playing the song: %v\n", err)
			http.Error(w, "Couldnnt play song", http.StatusServiceUnavailable)
			return
		}
	})
}

// stopHandler stops playing the alert song
func stopHandler(client *Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := client.Pause(); err != nil {
			log.Printf("Error stopping the playback: %v\n", err)
			http.Error(w, "Couldnt play alert song", http.StatusServiceUnavailable)
		}
	})
}

// Config contains alertify configuration
type Config struct {
	// RedirectURI is Spotify app OAuth redirect URI
	RedirectURI string
	// DeviceName is Spotify device name
	DeviceName string
	// DeviceID is Spotify device ID
	DeviceID string
	// SongURI is Spotify song URI
	SongURI string
}

func parseConfigFlags() (*Config, error) {
	// parse flags
	flag.Parse()

	if spotifyID := os.Getenv("SPOTIFY_ID"); spotifyID == "" {
		return nil, fmt.Errorf("Could not read SPOTIFY_ID environment variable")
	}

	if spotifyID := os.Getenv("SPOTIFY_SECRET"); spotifyID == "" {
		return nil, fmt.Errorf("Could not read SPOTIFY_SECRET environment variable")
	}

	return &Config{
		RedirectURI: redirectURI,
		DeviceName:  deviceName,
		DeviceID:    deviceID,
		SongURI:     songURI,
	}, nil
}

// Auth allows to authenticate with Spotify API
type Auth struct {
	// Spotify authenticator
	*spotify.Authenticator
	// State is OAuth state
	State string
	// RedirectURI is Spotify RedirectURI
	RedirectURI string
}

// NewAuth returns Auth to authenticate with Spotify API
func NewAuth(redirectURI string, state string) *Auth {
	// Spotify API authenticator
	auth := spotify.NewAuthenticator(redirectURI,
		spotify.ScopeUserReadCurrentlyPlaying,
		spotify.ScopeUserReadPlaybackState,
		spotify.ScopeUserModifyPlaybackState)

	return &Auth{&auth, state, redirectURI}
}

func (a *Auth) URL() string {
	// Spotify Login URL
	return a.AuthURL(a.State)
}

// Client wrapt spotifyClient
type Client struct {
	*spotify.Client
	// DeviceID
	deviceID *spotify.ID
}

func (c *Client) DeviceID() *spotify.ID {
	return c.deviceID
}

func (c *Client) SetDeviceID(cfg *Config) error {
	if cfg.DeviceID != "" {
		deviceID := spotify.ID(cfg.DeviceID)
		c.deviceID = &deviceID
		return nil
	}
	// get all Spotify devices
	devices, err := c.PlayerDevices()
	if err != nil {
		return err
	}
	var activeDevices []spotify.PlayerDevice
	// loop over available devices
	for _, device := range devices {
		if cfg.DeviceName != "" {
			if device.Name == cfg.DeviceName {
				c.deviceID = &device.ID
				return nil
			}
		}
		if !device.Restricted {
			activeDevices = append(activeDevices, device)
		}
	}

	if len(activeDevices) != 0 {
		c.deviceID = &activeDevices[0].ID
		return nil
	}

	return fmt.Errorf("No active device found!")
}

func (c *Client) PlaySong(songURI string) error {
	opts := &spotify.PlayOptions{
		DeviceID: c.DeviceID(),
		URIs:     []spotify.URI{spotify.URI(songURI)},
	}

	if err := c.PlayOpt(opts); err != nil {
		return err
	}

	return nil
}

func (c *Client) Pause() error {
	opts := &spotify.PlayOptions{
		DeviceID: c.DeviceID(),
	}

	if err := c.PauseOpt(opts); err != nil {
		return err
	}

	return nil
}

func main() {
	// alertify configuration
	config, err := parseConfigFlags()
	if err != nil {
		log.Fatalf("Error parsing cli flags: %v", err)
	}

	auth := NewAuth(config.RedirectURI, "abc123")
	fmt.Println("\nLog in to Spotify by visiting the following page in your browser:", auth.URL())

	// Spotify client channel
	clientChan := make(chan *spotify.Client)
	// HTTP API error channel
	errChan := make(chan error, 1)

	// Create HTTP muxer
	h := http.NewServeMux()
	// Catch all handler
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got request for: %s", r.URL.String())
	})
	// Auth handler: need to go through OAuth flow first
	h.Handle("/callback", authHandler(auth, clientChan))

	// Start the API service
	go func() {
		// TODO: handle server shutdown
		errChan <- http.ListenAndServe(":8080", h)
	}()

	// wait for auth to complete
	spotifyClient := <-clientChan
	// Set device ID
	deviceID := spotify.ID(config.DeviceID)
	client := &Client{spotifyClient, &deviceID}
	if err := client.SetDeviceID(config); err != nil {
		log.Fatal(err)
	}

	// register alertify API handlers
	h.Handle("/config", configHandler(client))
	h.Handle("/alert", alertHandler(client, config.SongURI))
	h.Handle("/stop", stopHandler(client))

	err = <-errChan
	log.Fatal(err)
}
