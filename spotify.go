package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
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
	// deviceID is Spotify device ID
	deviceID *spotify.ID
	// songURi is Spotify song URI
	songURI spotify.URI
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
		return nil, fmt.Errorf("Failed to create TCP listener: %s", err)
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

	fmt.Println("Log in to Spotify by visiting the following page in your browser:", auth.URL())

	var client *spotify.Client
	// wait for auth to complete
	select {
	case client = <-clientChan:
		listener.Close()
	case err = <-errChan:
		log.Printf("Error: %s", err)
	}
	wg.Wait()

	deviceID := spotify.ID(c.DeviceID)
	_client := &SpotifyClient{client, &deviceID, spotify.URI(c.SongURI), &sync.Mutex{}}
	if err := _client.SetDeviceID(c.DeviceID, c.DeviceName); err != nil {
		return nil, err
	}

	return _client, err
}

// DeviceID returns Spotify device ID
func (s *SpotifyClient) DeviceID() *spotify.ID {
	return s.deviceID
}

// SongURI returns Spotify Song URI
func (s *SpotifyClient) SongURI() spotify.URI {
	return s.songURI
}

// SetDeviceID allows to set Spotify device ID on which you can play songs
// If deviceID is smpty string, it searches for device ID of device specified by deviceName
// If both deviceID and deviceName are empty Spotify client will use the first active device it finds
func (s *SpotifyClient) SetDeviceID(deviceID, deviceName string) error {
	// prevent multiple client device modifications
	s.Lock()
	defer s.Unlock()

	if deviceID != "" {
		_deviceID := spotify.ID(deviceID)
		s.deviceID = &_deviceID
		return nil
	}
	// get all Spotify devices
	devices, err := s.PlayerDevices()
	if err != nil {
		return err
	}
	var activeDevices []spotify.PlayerDevice
	// loop over available devices
	for _, device := range devices {
		if deviceName != "" {
			if device.Name == deviceName {
				s.deviceID = &device.ID
				return nil
			}
		}
		if !device.Restricted {
			activeDevices = append(activeDevices, device)
		}
	}

	if len(activeDevices) != 0 {
		s.deviceID = &activeDevices[0].ID
		return nil
	}

	return fmt.Errorf("No active device found")
}

// playAlertSong plays song specified by songURI
func (s *SpotifyClient) playAlertSong(songURI string) error {
	s.Lock()
	defer s.Unlock()
	if songURI != "" {
		s.songURI = spotify.URI(songURI)
	}

	opts := &spotify.PlayOptions{
		DeviceID: s.DeviceID(),
		URIs:     []spotify.URI{s.songURI},
	}

	return s.PlayOpt(opts)
}

// Play plays default Spotify song
func (s *SpotifyClient) Play() error {
	return s.playAlertSong("")
}

// PlaySong plays Spotify song passed in via config parameter
func (s *SpotifyClient) PlaySong(songURI string) error {
	return s.playAlertSong(songURI)
}

// Pause pauses active playback ore returns error if the playback can't be paused
func (s *SpotifyClient) Pause() error {
	opts := &spotify.PlayOptions{
		DeviceID: s.DeviceID(),
	}

	return s.PauseOpt(opts)
}
