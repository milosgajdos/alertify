package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/milosgajdos/alertify"
	"github.com/milosgajdos/alertify/monitor"
)

const (
	// cliname is command line interface name
	cliname = "slackertify"
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
	flag.StringVar(&redirectURI, "redirect-uri", "http://localhost:8080/callback", "Spotify API redirect URI")
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
	Bot *alertify.BotConfig
	// Slack configures Slack message monitor
	Slack *monitor.SlackConfig
}

func parseCliFlags() (*Config, error) {
	// parse flags
	flag.Parse()

	spotifyID := os.Getenv("SPOTIFY_ID")
	if spotifyID == "" {
		return nil, fmt.Errorf("could not read SPOTIFY_ID environment variable")
	}

	spotifySecret := os.Getenv("SPOTIFY_SECRET")
	if spotifySecret == "" {
		return nil, fmt.Errorf("could not read SPOTIFY_SECRET environment variable")
	}

	slackAPIKey := os.Getenv("SLACK_API_KEY")
	if slackAPIKey == "" {
		return nil, fmt.Errorf("could not read SLACK_API_KEY environment variable")
	}

	return &Config{
		Bot: &alertify.BotConfig{
			Spotify: &alertify.SpotifyConfig{
				ClientID:     spotifyID,
				ClientSecret: spotifySecret,
				RedirectURI:  redirectURI,
				DeviceName:   deviceName,
				DeviceID:     deviceID,
				SongURI:      songURI,
			},
		},
		Slack: &monitor.SlackConfig{
			APIKey:  slackAPIKey,
			Channel: slackChannel,
			User:    slackUser,
			Msg:     slackMsg,
		},
	}, nil
}

// registerSignals registers signal notification channel
func registerSignals(sig ...os.Signal) <-chan os.Signal {
	// Signal handler to stop the bot when termination signal is received
	sigChan := make(chan os.Signal, 1)
	// register signal handler
	signal.Notify(sigChan, sig...)

	return sigChan
}

// listenAndAlert starts bot and all monitors
func listenAndAlert(bot *alertify.Bot) error {
	// Create error channel
	errChan := make(chan error, 1)

	// OS signal notification channel
	sigChan := registerSignals(os.Interrupt, os.Kill, syscall.SIGTERM)

	var wg sync.WaitGroup
	// start bot goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting bot message listener")
		errChan <- bot.ListenAndAlert()
		log.Printf("Bot message listener stopped")
	}()

	var err error
	select {
	case sig := <-sigChan:
		log.Printf("Got signal: %s: Shutting down", sig)
	case err = <-errChan:
	}

	// stop all listeners
	bot.Stop()
	wg.Wait()

	return err
}

func main() {
	// read config
	cfg, err := parseCliFlags()
	if err != nil {
		log.Printf("Error parsing cli flags: %v", err)
		os.Exit(1)
	}

	// create bot
	bot, err := alertify.NewBot(cfg.Bot)
	if err != nil {
		log.Printf("Error creating bot: %v", err)
		os.Exit(1)
	}

	// Create Slack monitor
	slack, err := monitor.NewSlackMonitor(cfg.Slack)
	if err != nil {
		log.Printf("Error creating slack monitor: %s", err)
		os.Exit(1)
	}

	// register slack message monitor
	if err := bot.RegisterMonitor(slack); err != nil {
		log.Printf("Error registering %s: %s", slack, err)
		os.Exit(1)
	}

	// start alertify bot
	if err := listenAndAlert(bot); err != nil {
		log.Printf("Error: %s", err)
		os.Exit(1)
	}
}
