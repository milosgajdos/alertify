package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	// cliname is command line interface name
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
	// slackChannel is name of the Slack channel that receives alerts
	slackChannel string
	// slackUser is name of the Slack bot which posts alerts to slackChannel
	slackUser string
	// slackMsg is a string representing regular expression we are matching on
	slackMsg string
)

func init() {
	flag.StringVar(&redirectURI, "redirect-uri", "http://localhost:8080/callback", "OAuth app redirect URI set in Spotify API console")
	flag.StringVar(&deviceName, "device-name", "", "Spotify device name as recognised by Spotify API")
	flag.StringVar(&deviceID, "device-id", "", "Spotify device ID as recognised by Spotify API")
	flag.StringVar(&songURI, "song-uri", "spotify:track:2xYlyywNgefLCRDG8hlxZq", "Spotify song URI")
	flag.StringVar(&slackChannel, "slack-channel", "devops-production", "Slack channel that receives alerts")
	flag.StringVar(&slackUser, "slack-user", "production", "Slack username whose message we alert on")
	flag.StringVar(&slackMsg, "slack-msg", "alert", "A regexp we are matching the slack messages on")
	// disable timestamps and set prefix
	log.SetFlags(0)
	log.SetPrefix("[ " + cliname + " ] ")
}

// Config contains configuration parameters
type Config struct {
	// Bot configures alertify bot
	Bot *BotConfig
	// Slack configures Slack API client
	Slack *SlackConfig
}

func parseCliFlags() (*Config, error) {
	// parse flags
	flag.Parse()

	spotifyID := os.Getenv("SPOTIFY_ID")
	if spotifyID == "" {
		return nil, fmt.Errorf("Could not read SPOTIFY_ID environment variable")
	}

	spotifySecret := os.Getenv("SPOTIFY_SECRET")
	if spotifySecret == "" {
		return nil, fmt.Errorf("Could not read SPOTIFY_SECRET environment variable")
	}

	slackAPIKey := os.Getenv("SLACK_API_KEY")
	if slackAPIKey == "" {
		return nil, fmt.Errorf("Could not read SLACK_API_KEY environment variable")
	}

	return &Config{
		Bot: &BotConfig{
			Spotify: &SpotifyConfig{
				ClientID:     spotifyID,
				ClientSecret: spotifySecret,
				RedirectURI:  redirectURI,
				DeviceName:   deviceName,
				DeviceID:     deviceID,
				SongURI:      songURI,
			},
		},
		Slack: &SlackConfig{
			APIKey:  slackAPIKey,
			Channel: slackChannel,
			User:    slackUser,
			Msg:     slackMsg,
		},
	}, nil
}

func main() {
	var wg sync.WaitGroup

	// read config
	cfg, err := parseCliFlags()
	if err != nil {
		log.Printf("Error parsing cli flags: %v", err)
		os.Exit(1)
	}

	// create bot
	bot, err := NewBot(cfg.Bot)
	if err != nil {
		log.Printf("Error creating bot: %v", err)
		os.Exit(1)
	}

	// Signal handler to stop the bot when termination signal is received
	sigc := make(chan os.Signal, 1)
	// register signal handler
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	// Create error channel
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting bot message listener")
		errChan <- bot.ListenAndAlert()
		log.Printf("Bot message listener stopped")
	}()

	// Create Slack client
	slack, err := NewSlackListener(cfg.Slack)
	if err != nil {
		log.Printf("Error creating slack listener: %s", err)
		os.Exit(1)
	}

	// start Slack API message listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting Slack API message listener")
		errChan <- slack.ListenAndAlert(bot.MsgChan())
		log.Printf("Slack message listener stopped")
	}()

	var e error
	select {
	case sig := <-sigc:
		log.Printf("Shutting down -> got signal: %s", sig)
	case e = <-errChan:
	}

	// stop all listeners
	bot.Stop()
	slack.Stop()
	wg.Wait()

	// if we are shutting down due to an error, exit with non-zero status
	if err != nil {
		log.Printf("Error: %s", e)
		os.Exit(1)
	}
}
