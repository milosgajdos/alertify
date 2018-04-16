package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Bot wraps Spotify and Slack clients
// and offers a simple API to interact with it
type Bot struct {
	// sptf is Spotify client
	sptf *SpotifyClient
	// slck ic Slack API client
	slck *SlackClient
	// api is HTTP API service
	api *API
	// alertSongURI is Spotify song URI
	alertSongURI string
	// cmdChan is Bot command control channel
	cmdChan chan *Msg
	// doneChan stops Bot command listener
	doneChan chan struct{}
}

// Config contains alertify bot configuration
type Config struct {
	// Spotify contains Spotify API configu
	Spotify *SpotifyConfig
	// Slack contains Slack API config
	Slack *SlackConfig
}

// Msg is a Bot message that allows to control aleritfy bot
type Msg struct {
	Cmd      string
	Data     interface{}
	respChan chan interface{}
}

// NewBot creates new alertify bot and returns it
func NewBot(c *Config) (*Bot, error) {
	// Create Spotify client and set Spotify Device ID
	spotifyClient, err := NewSpotifyClient(c.Spotify)
	if err != nil {
		return nil, err
	}
	// Create Slack client
	slackClient, err := NewSlackClient(c.Slack)
	if err != nil {
		return nil, err
	}
	// create command channel
	cmdChan := make(chan *Msg)
	doneChan := make(chan struct{})
	ctx := &Context{cmdChan}
	api, err := NewAPI(ctx, ":8080", nil)
	if err != nil {
		return nil, err
	}

	return &Bot{
		sptf:         spotifyClient,
		slck:         slackClient,
		api:          api,
		alertSongURI: c.Spotify.SongURI,
		cmdChan:      cmdChan,
		doneChan:     doneChan,
	}, nil
}

// Alert plays a Spotify song
func (b *Bot) Alert(msg *Msg) error {
	if msg.Data == nil {
		return b.sptf.PlaySong(b.alertSongURI)
	}

	songURI, ok := msg.Data.(string)
	if !ok {
		return fmt.Errorf("Invalid Spotify SongURI: %s", msg.Data)
	}

	return b.sptf.PlaySong(songURI)
}

// Silence pauses Spotify playback
func (b *Bot) Silence() error {
	return b.sptf.Pause()
}

// runCmd runs bot command
func (b *Bot) runCmd(msg *Msg) {
	switch msg.Cmd {
	case "alert":
		msg.respChan <- b.Alert(msg)
	case "silence":
		msg.respChan <- b.Silence()
	default:
		msg.respChan <- fmt.Errorf("Invalid command")
	}
}

// Start starts alertify bot
func (b *Bot) Start() error {
	// TODO: connects to Slack channel and listens for events
	// This should launch Slack.ListenAndAlert()
	var err error
	var wg sync.WaitGroup

	// Create error channel
	errChan := make(chan error, 3)

	// Signal handler to stop the framework scheduler and API
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)

	// start Bot command listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting bot command listener")
		for {
			select {
			case msg := <-b.cmdChan:
				log.Printf("Received command: %s", msg.Cmd)
				b.runCmd(msg)
			case <-b.doneChan:
				log.Printf("Shutting down bot")
				return
			}
		}
	}()

	// Start API service
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting HTTP API service")
		errChan <- b.api.ListenAndServe()
	}()

	// start Slack API listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting Slack API message watcher")
		errChan <- b.slck.WatchAndAlert(b.cmdChan)
	}()

	select {
	case sig := <-sigc:
		log.Printf("Shutting down -> got signal: %s", sig)
	case err = <-errChan:
		log.Printf("Bot error: %s", err)
	}

	log.Printf("Stopping HTTP API service")
	b.api.l.Close()
	log.Printf("Stopping Slack API message watcher")
	b.slck.Stop()
	log.Printf("Stopping Bot command listener")
	close(b.doneChan)
	wg.Wait()

	return err
}
