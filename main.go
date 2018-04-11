package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	// alertChannel is name of the Slack channel that receives alerts
	alertChannel string
	// alertBot is name of the Slack bot which posts alerts to alertChannel
	alertBot string
	// alertMsg is a string representing regular expression we are mathing on
	alertMsg string
)

func init() {
	flag.StringVar(&redirectURI, "redirect-uri", "http://localhost:8080/callback", "OAuth app redirect URI set in Spotify API console")
	flag.StringVar(&deviceName, "device-name", "", "Spotify device name as recognised by Spotify API")
	flag.StringVar(&deviceID, "device-id", "", "Spotify device ID as recognised by Spotify API")
	flag.StringVar(&songURI, "song-uri", "spotify:track:2xYlyywNgefLCRDG8hlxZq", "Spotify song URI")
	flag.StringVar(&alertChannel, "alert-channel", "devops-production", "Slack channel that receives alerts")
	flag.StringVar(&alertBot, "alert-bot", "production", "Slack bot which sends alerts")
	flag.StringVar(&alertMsg, "alert-msg", "alert", "Slack message regexp we are matching the alert messages on")
	// disable timestamps and set prefix
	log.SetFlags(0)
	log.SetPrefix("[ " + cliname + " ] ")
}

func parseConfigFlags() (*Config, error) {
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
		Spotify: &SpotifyConfig{
			ClientID:     spotifyID,
			ClientSecret: spotifySecret,
			RedirectURI:  redirectURI,
			DeviceName:   deviceName,
			DeviceID:     deviceID,
			SongURI:      songURI,
		},
		Slack: &SlackConfig{
			APIKey:       slackAPIKey,
			AlertBot:     alertBot,
			AlertChannel: alertChannel,
			AlertMsg:     alertMsg,
		},
	}, nil
}

func main() {
	// read config
	c, err := parseConfigFlags()
	if err != nil {
		log.Printf("Error parsing cli flags: %v", err)
		os.Exit(1)
	}
	// create bot
	bot, err := NewBot(c)
	if err != nil {
		log.Printf("Error creating bot: %v", err)
		os.Exit(1)
	}
	// start alertify bot
	if err := bot.Start(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
