package alertify

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/zmb3/spotify"
)

// SpotifyConfig configures Spotify API client
type SpotifyConfig struct {
	// ClientID is Spotify Client ID
	ClientID string
	// ClientSecret is Spotify Client secret
	ClientSecret string
	// RedirectURI is Spotify app OAuth redirect URI
	RedirectURI string
	// DeviceID is Spotify device ID
	DeviceID string
	// DeviceName is Spotify device name
	DeviceName string
	// SongURI is Spotify song URI
	SongURI string
}

// SpotifyAuth allows to authenticate with Spotify API
type SpotifyAuth struct {
	// Spotify authenticator
	*spotify.Authenticator
	// State is OAuth state
	State string
	// RedirectURI is Spotify RedirectURI
	RedirectURI string
}

// NewSpotifyAuth returns SpotifyAuth which is used to authenticate with Spotify API
func NewSpotifyAuth(clientID, clientSecret, redirectURI, state string) *SpotifyAuth {
	// Spotify API authenticator
	auth := spotify.NewAuthenticator(redirectURI,
		spotify.ScopeUserReadCurrentlyPlaying,
		spotify.ScopeUserReadPlaybackState,
		spotify.ScopeUserModifyPlaybackState)
	auth.SetAuthInfo(clientID, clientSecret)

	return &SpotifyAuth{&auth, state, redirectURI}
}

// URL returns Spotify API OAuth URL
func (a *SpotifyAuth) URL() string {
	// Spotify Login URL
	return a.AuthURL(a.State)
}

// SpotifyClient implements Spotify client
type SpotifyClient struct {
	// Spotify client
	*spotify.Client
	// device is Spotify player device
	device *spotify.PlayerDevice
	// mutex
	*sync.Mutex
}

// authHandler handles OAuth2 authentication callback from Spotify API
func authHandler(auth *SpotifyAuth, ch chan *spotify.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err := auth.Token(auth.State, r)
		if err != nil {
			http.Error(w, "Couldn't retrieve OAuth token", http.StatusForbidden)
			// TODO: handle the errors better
			log.Fatal(err)
		}
		if s := r.FormValue("state"); s != auth.State {
			http.NotFound(w, r)
			log.Fatalf("OAuth State mismatch: %s != %s\n", s, auth.State)
		}
		// use the token to get an authenticated client
		client := auth.NewClient(tok)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Spotify login successful!")
		ch <- &client
	})
}

// NewSpotifyClient authenticates with Spotify API and returns SpotifyClient
// It returns error if Spotify API authentication fails
func NewSpotifyClient(c *SpotifyConfig) (*SpotifyClient, error) {
	clientChan := make(chan *spotify.Client)
	errChan := make(chan error, 1)
	// create OAuth listener for RedirectURI callback
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener: %s", err)
	}
	// Spotify authenticator
	auth := NewSpotifyAuth(c.ClientID, c.ClientSecret, c.RedirectURI, "abc123")
	// Create HTTP muxer
	h := http.NewServeMux()
	h.Handle("/callback", authHandler(auth, clientChan))
	// HTTP server for Spotify OAuth
	server := &http.Server{
		Handler: h,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errChan <- server.Serve(listener)
	}()

	fmt.Println("Log in to Spotify by visiting the following URL in your browser:", auth.URL())

	var client *spotify.Client
	// wait for auth to complete
	select {
	case client = <-clientChan:
		if err := listener.Close(); err != nil {
			log.Printf("Error closing auth listener: %v", err)
		}
	case err = <-errChan:
	}
	wg.Wait()

	if err != nil {
		return nil, fmt.Errorf("failed to create Spotify client: %s", err)
	}

	// configure Spotify player device
	deviceID := spotify.ID(c.DeviceID)
	device := &spotify.PlayerDevice{ID: deviceID}
	_client := &SpotifyClient{client, device, &sync.Mutex{}}
	if err := _client.SetDevice(c.DeviceID, c.DeviceName); err != nil {
		return nil, err
	}

	return _client, err
}

// Device returns Spotify active playback device
func (s *SpotifyClient) Device() *spotify.PlayerDevice {
	return s.device
}

// SetDevice allows to set Spotify player device on which you can play alert song
// If deviceID is empty string, it searches for device ID of the device specified by deviceName
// If both deviceID and deviceName are empty Spotify client will use the first active device it finds
func (s *SpotifyClient) SetDevice(deviceID, deviceName string) error {
	// prevent multiple client device modifications
	s.Lock()
	defer s.Unlock()

	// get all available Spotify player devices
	devices, err := s.PlayerDevices()
	if err != nil {
		return err
	}

	var activeDevices []spotify.PlayerDevice
	// loop over available devices
	for _, device := range devices {
		if !device.Restricted {
			// Search by device ID
			if deviceID == device.ID.String() {
				s.device.ID = device.ID
				s.device.Name = device.Name
				return nil
			}
			// search by device Name
			if device.Name == deviceName {
				s.device.ID = device.ID
				s.device.Name = device.Name
				return nil
			}
			activeDevices = append(activeDevices, device)
		}
	}

	if len(activeDevices) != 0 {
		s.device.ID = activeDevices[0].ID
		s.device.Name = activeDevices[0].Name
		return nil
	}

	return fmt.Errorf("no active Spotify devices found")
}

// PlaySong plays Spotify song passed in as songURI
func (s *SpotifyClient) PlaySong(songURI string) error {
	s.Lock()
	defer s.Unlock()
	// if empty, play default song
	if songURI == "" {
		songURI = "spotify:track:7yTIKQzqRQfXDKKiPw3GJY"
	}
	// set playback options
	opts := &spotify.PlayOptions{
		DeviceID: &s.device.ID,
		URIs:     []spotify.URI{spotify.URI(songURI)},
	}

	var trackName string
	trackInfo := strings.Split(songURI, ":")
	if len(trackInfo) != 3 {
		log.Printf("Could not parse Spotify ID from %s", songURI)
	} else {
		track, err := s.GetTrack(spotify.ID(trackInfo[2]))
		if err != nil {
			log.Printf("Failed to get %s track name: %s", songURI, err)
			trackName = "Unknown"
		} else {
			trackName = track.Name
		}
	}

	log.Printf("Attempting to play: \"%s\" on Device ID: %s Name: %s", trackName, s.device.ID, s.device.Name)

	return s.PlayOpt(opts)
}

// Pause pauses active playback on a currently active Spotify device
// It returns error if the playback can't be paused
func (s *SpotifyClient) Pause() error {
	opts := &spotify.PlayOptions{
		DeviceID: &s.device.ID,
	}

	log.Printf("Attempting to pause alert playback on Device ID: %s Name: %s", s.device.ID, s.device.Name)

	return s.PauseOpt(opts)
}
