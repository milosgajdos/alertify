package monitor

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/milosgajdos83/alertify"
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

// SlackMonitor is Slack API client which monitors messages
type SlackMonitor struct {
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

// NewSlackMonitor creates new Slack message monitor
func NewSlackMonitor(c *SlackConfig) (*SlackMonitor, error) {
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

	return &SlackMonitor{api, rtm, c.User, c.Channel, msg, doneChan, false, m}, nil
}

// String returns the name of the monitor
func (s *SlackMonitor) String() string {
	return "Slack Monitor"
}

// Channel returns the name of the Slack channel which we monitor
func (s *SlackMonitor) Channel() string {
	return s.channel
}

// User returns slack user name whose message we monitor
func (s *SlackMonitor) User() string {
	return s.user
}

// watchMessages listens to Slack messages and notifies alertify bot when a message regexp is matched
func (s *SlackMonitor) watchMessages(alertChan chan struct{}, errChan chan error) {
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
			errChan <- fmt.Errorf("invalid Slack API credentials")

		default:
		}
	}
}

// MonitorAndAlert monitors channel and notifies alertify Bot when the preconfigured message regexp is matched
func (s *SlackMonitor) MonitorAndAlert(msgChan chan<- *alertify.Msg) error {
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
		// check if the slack monitor is running
		s.Lock()
		s.isRunning = true
		s.Unlock()
		select {
		case <-alertChan:
			log.Printf("Slack alert message match detected!")
			// send message to alertify bot to play song
			go func() {
				msgChan <- &alertify.Msg{
					Cmd:  "alert",
					Data: nil,
					Resp: respChan,
				}
			}()
			if err := <-respChan; err != nil {
				log.Printf("Could not play song: %v", err.(error))
			}
		case <-s.doneChan:
			// disconnect from RTM API
			return s.rtm.Disconnect()
		case err := <-errChan:
			return err
		}
	}
}

// Stop stops Slack event watched
func (s *SlackMonitor) Stop() {
	s.Lock()
	defer s.Unlock()

	if s.isRunning {
		close(s.doneChan)
		s.isRunning = false
	}
}
