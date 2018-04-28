package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/nlopes/slack"
)

// SlackConfig configures Slack API client
type SlackConfig struct {
	// APIKey is Slack API key
	APIKey string
	// Channel is the name of Slack channel
	Channel string
	// User is the name of Slack user
	User string
	// Msg is the message we are matching for
	Msg string
}

// SlackClient is Slack API client
type SlackClient struct {
	// embedding Slack client
	*slack.Client
	// rtm is Slack RTM client
	rtm *slack.RTM
	// user is Slack bot name
	user string
	// channel is Slack channel
	channel string
	// msg is RegExp we are matching for
	msg *regexp.Regexp
	// doneChan stops Slack client
	doneChan chan struct{}
	// isRunning checks if bot is running
	isRunning bool
	// mutex
	*sync.Mutex
}

// NewSlackListener creates new Slack message listener.
func NewSlackListener(c *SlackConfig) (*SlackClient, error) {
	api := slack.New(c.APIKey)
	rtm := api.NewRTM()
	// compile message regexp
	msg, err := regexp.Compile(c.Msg)
	if err != nil {
		return nil, err
	}
	// create notification channel
	doneChan := make(chan struct{})
	// mutex
	m := &sync.Mutex{}

	return &SlackClient{api, rtm, c.User, c.Channel, msg, doneChan, false, m}, nil
}

// Channel returns the name of the Slack channel which we monitor
func (s *SlackClient) Channel() string {
	return s.channel
}

// User returns slack user name whose message we monitor
func (s *SlackClient) User() string {
	return s.user
}

// watchMessages listens to Slack messages and notifies alertify bot when a message regexp is matched
func (s *SlackClient) watchMessages(alertChan chan struct{}, errChan chan error) {
	// monitor all slack messages
	for msg := range s.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			// if alertbot said something, send alert
			user := s.rtm.GetInfo().User.Name
			if strings.EqualFold(user, s.user) {
				if s.msg.MatchString(ev.Text) {
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

// ListenAndAlert monitors channel and notifies alertify Bot when the preconfigured message regexp is matched
func (s *SlackClient) ListenAndAlert(msgChan chan<- *Msg) error {
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
		s.Lock()
		s.isRunning = true
		s.Unlock()
		select {
		case <-alertChan:
			log.Printf("Slack alert message match detected!")
			// send message to alertify bot to play song
			go func() { msgChan <- &Msg{"alert", nil, respChan} }()
			if err := <-respChan; err != nil {
				log.Printf("Could not play song: %v", err.(error))
			}
		case <-s.doneChan:
			log.Printf("Shutting down Slack message listener")
			// disconnect from RTM API
			return s.rtm.Disconnect()
		case err := <-errChan:
			return err
		}
	}
}

// Stop stops Slack event watched
func (s *SlackClient) Stop() {
	s.Lock()
	defer s.Unlock()

	if s.isRunning {
		close(s.doneChan)
		s.isRunning = false
	}
	return
}
