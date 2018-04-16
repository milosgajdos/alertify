package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

// SlackConfig configures Slack API client
type SlackConfig struct {
	// APIKey is Slack API key
	APIKey string
	// AlertBot is the name of Slack alert bot
	AlertBot string
	// AlertChannel is the name of Slack channel that receives alerts
	AlertChannel string
	// AlertMsg is the message we are matching for
	AlertMsg string
}

// SlackClient is Slack API client
type SlackClient struct {
	// embedding Slack client
	*slack.Client
	// rtm is Slack RTM client
	rtm *slack.RTM
	// alertBot is Slack bot name
	alertBot string
	// alertChannel is Slack channel
	alertChannel string
	// alertMessage is RegExp we are matching for
	alertMsg *regexp.Regexp
	// doneChan stops Slack client
	doneChan chan struct{}
}

// NewSlackClient creates new Slack client and returns it
func NewSlackClient(c *SlackConfig) (*SlackClient, error) {
	api := slack.New(c.APIKey)
	rtm := api.NewRTM()
	alertMsg, err := regexp.Compile(c.AlertMsg)
	if err != nil {
		return nil, err
	}
	doneChan := make(chan struct{})

	return &SlackClient{api, rtm, c.AlertBot, c.AlertChannel, alertMsg, doneChan}, nil
}

// AlertChannel returns the name of Slack channel that receives alerts
func (s *SlackClient) AlertChannel() string {
	return s.alertChannel
}

// AlertBot returns slack alert bot name to which alertify listens
func (s *SlackClient) AlertBot() string {
	return s.alertBot
}

// watchMessages watches Slack messages and notifies alertify when alerts is detected
func (s *SlackClient) watchMessages(alertChan chan struct{}, errChan chan error) {
	// monitor all slack messages
	for msg := range s.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			// if alertbot said something, send alert
			user := s.rtm.GetInfo().User.Name
			if strings.EqualFold(user, s.alertBot) {
				if s.alertMsg.MatchString(ev.Text) {
					alertChan <- struct{}{}
				}
			}

		case *slack.LatencyReport:
			log.Printf("Current Slack RTM latency: %v", ev.Value)

		case *slack.RTMError:
			// can do error.New(ev.Error())
			errChan <- fmt.Errorf(ev.Error())

		case *slack.InvalidAuthEvent:
			errChan <- fmt.Errorf("Invalid Slack API credentials")

		default:
		}
	}
}

// WatchAndAlert watches alert channel and notifies alertify Bot which plays music
func (s *SlackClient) WatchAndAlert(cmdChan chan *Msg) error {
	// start RTM connection
	go s.rtm.ManageConnection()
	// slack message notification channel
	alertChan := make(chan struct{})
	// errChan is error channel
	errChan := make(chan error)
	// listen on incoming messages
	go s.watchMessages(alertChan, errChan)

	// bot response channel
	respChan := make(chan interface{})

	for {
		select {
		case <-alertChan:
			log.Printf("Slack alert detected")
			// send message to alertify bot to play song
			go func() { cmdChan <- &Msg{"alert", nil, respChan} }()
			if err := <-respChan; err != nil {
				log.Printf("Could not play song: %v", err.(error))
			}
		case <-s.doneChan:
			log.Printf("Shutting down slack listener")
			// close IncomingEvents channel
			close(s.rtm.IncomingEvents)
			return s.rtm.Disconnect()
		case err := <-errChan:
			return err
		}
	}
}

// Stop stops Slack event watched
func (s *SlackClient) Stop() {
	close(s.doneChan)

	return
}
