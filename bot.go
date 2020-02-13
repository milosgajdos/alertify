package alertify

import (
	"fmt"
	"log"
	"sync"
)

// Msg is allows to control aleritfy bot behavior
type Msg struct {
	// Cmd is specifies command name
	Cmd string
	// Data allows to attach arbitrary data to the message
	Data interface{}
	// Resp is a channel used to send response back to monitor
	Resp chan interface{}
}

// Bot plays spotify songs when requested
type Bot struct {
	// spotify is Spotify client
	spotify *SpotifyClient
	// api is HTTP API service
	api *API
	// songURI is Spotify song URI
	songURI string
	// msgChan allows to send command messages to Bot
	msgChan chan *Msg
	// closeMsgChan stops Bot message listener
	closeMsgChan chan struct{}
	// monitors are Bot monitors
	monitors []Monitor
	// isRunning checks if bot is running
	isRunning bool
	// mutex
	*sync.Mutex
}

// BotConfig configures alertify bot
type BotConfig struct {
	// Spotify configures Spotify API client
	Spotify *SpotifyConfig
}

// NewBot creates new alertify bot and returns it
// It fails with error if neither of the following couldnt be created:
// Spotify API client, Slack API client, HTTP API service
func NewBot(c *BotConfig) (*Bot, error) {
	// Create Spotify client and set Spotify Device ID
	spotifyClient, err := NewSpotifyClient(c.Spotify)
	if err != nil {
		return nil, err
	}
	// create message channel
	msgChan := make(chan *Msg)
	// create close message channel
	closeMsgChan := make(chan struct{})
	// Create HTTP API
	api, err := NewAPI(&Context{msgChan}, ":8080", nil)
	if err != nil {
		return nil, err
	}

	// monitors keeps a list of registered monitors
	monitors := make([]Monitor, 0)

	return &Bot{
		spotify:      spotifyClient,
		api:          api,
		songURI:      c.Spotify.SongURI,
		msgChan:      msgChan,
		closeMsgChan: closeMsgChan,
		monitors:     monitors,
		isRunning:    false,
		Mutex:        &sync.Mutex{},
	}, nil
}

// Alert plays songURI song on Spotify
func (b *Bot) Alert(songURI string) error {
	if songURI == "" {
		songURI = b.songURI
	}
	return b.spotify.PlaySong(songURI)
}

// Silence pauses Spotify playback
func (b *Bot) Silence() error {
	return b.spotify.Pause()
}

// RegisterMonitor registers remote monitor
func (b *Bot) RegisterMonitor(monitors ...Monitor) error {
	// add all monitors to bot list
	b.monitors = append(b.monitors, monitors...)

	return nil
}

// processMsg processes bot message and runs bot command
func (b *Bot) processMsg(msg *Msg) {
	switch msg.Cmd {
	case "alert":
		var songURI string
		var ok bool
		if msg.Data != nil {
			songURI, ok = msg.Data.(string)
			if !ok {
				songURI = b.songURI
			}
		}
		msg.Resp <- b.Alert(songURI)
	case "silence":
		msg.Resp <- b.Silence()
	default:
		msg.Resp <- fmt.Errorf("invalid command")
	}
}

// ListenAndAlert starts Bot message listener and plays Spotify song when it receives alert message
// This is a blocking function call and therefore should be run in a dedicated goroutine
func (b *Bot) ListenAndAlert() error {

	var wg sync.WaitGroup
	// Create error channel
	errChan := make(chan error, len(b.monitors)+1)

	// Start Bot message listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case msg := <-b.msgChan:
				log.Printf("Received message: %s", msg.Cmd)
				b.processMsg(msg)
			case <-b.closeMsgChan:
				log.Printf("Stopping message listener")
				errChan <- nil
				return
			}
		}
	}()

	// Start Bot HTTP API service
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting HTTP API service")
		errChan <- b.api.ListenAndServe()
	}()

	// Start all remote monitors
	for _, mon := range b.monitors {
		wg.Add(1)
		go func(m Monitor) {
			defer wg.Done()
			log.Printf("Starting %s", m)
			errChan <- m.MonitorAndAlert(b.msgChan)
			log.Printf("%s stopped", m)
		}(mon)
	}

	// set bot status to running
	b.Lock()
	b.isRunning = true
	b.Unlock()

	// wait for error
	err := <-errChan

	log.Printf("HTTP API service shutting down")
	if err := b.api.l.Close(); err != nil {
		log.Printf("Error shutting down API listener: %v", err)
	}
	log.Printf("HTTP API service stopped")

	log.Printf("Message listener shutting down")
	b.Stop()
	log.Printf("Message listener stopped")

	// Stop all remote monitors
	for _, m := range b.monitors {
		log.Printf("Shutting down %s", m)
		m.Stop()
		log.Printf("%s stopped", m)
	}
	wg.Wait()

	return err
}

// Stop stops bot listener
func (b *Bot) Stop() {
	b.Lock()
	defer b.Unlock()

	if b.isRunning {
		close(b.closeMsgChan)
		b.isRunning = false
	}
}
