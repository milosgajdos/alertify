# alertify

[![GoDoc](https://godoc.org/github.com/milosgajdos/alertify?status.svg)](https://godoc.org/github.com/milosgajdos/alertify)
[![Go Report Card](https://goreportcard.com/badge/milosgajdos/alertify)](https://goreportcard.com/report/github.com/milosgajdos/alertify)
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Travis CI](https://travis-ci.org/milosgajdos/alertify.svg?branch=master)](https://travis-ci.org/milosgajdos/alertify)
[![DeepSource](https://static.deepsource.io/deepsource-badge-light-mini.svg)](https://deepsource.io/gh/milosgajdos/alertify/?ref=repository-badge)

`alertify` is a simple `Go` package which allows to play a song on a [Spotify](https://www.spotify.com/uk/) device upon receiving alert message from a preconfigured source or via HTTP API request.

The original goal of the project was to play a Spotify song when a critical infrastructure alert is detected in a dedicated Slack channel. The project has since evolved beyond this goal and now allows to plug in different monitoring sources.

# Prerequisites

`alertify` uses Spotify API therefore there is a couple of prerequisites which need to be satisfied before you can use it.

## Spotify API access

Before you use `alertify` package you need to register your application on [Spotify developer portal](https://beta.developer.spotify.com/dashboard/applications). Retrieve the `Client ID` and `Client Secret` API keys and store them in a secure location.

`alertify` uses [Spotify Connect Web API](https://beta.developer.spotify.com/documentation/web-api/guides/using-connect-web-api/) to play a song on a Spotify device. Spotify Connect API uses `OAuth` to grant the permissions to your application with the following Spotify API access scopes.
* [user-read-currently-playing](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-read-currently-playing)
* [user-modify-playback-state](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-modify-playback-state)
* [user-read-playback-state](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-read-playback-state)

## Spotify application settings

Once your application has been registered, you need to visit your application settings in the [developer portal](https://beta.developer.spotify.com/dashboard/applications) and specify **Redirect URIs** where the `alertify` Spotify API client will redirect you after the API authentication failure or success.

# Design notes

At the core of the package is `alertify.Bot` object, which is responsible for playing the songs on the preconfigured Spotify device. Besides the ability to play the Spotify songs, `alertify.Bot` also provides a simple HTTP API service, alas at this point the API service does not provide any authentication so be careful if you use this project on publicly accessible network: luckily the API service can be bound to a local `unix` socket, so you might want to use that option.

`alertify.Bot` is intended to run in a dedicated `goroutine` and when run on its own it does nothing unless being explicitly requested to play a song via its HTTP API. The API also provides an endpoint which allows to pause the song playback.

Things get more interesting when you register some alert "monitors" with the `alertify.Bot`. The monitors are objects which satisfy `alertify.Monitor` interface and which can communicate with `alertify.Bot` by sending it `alertify.Msg` objects over the predefined `Go` channel. Please see the [Godoc](https://godoc.org/github.com/milosgajdos/alertify) for implementation details.

If I have more time I'll move the local in-process communication from `Go` channels to `protobufs` or provide `protobufs` communication interface as well,  but at this point I couldnt be bothered as it's just a fun side project and `Go channel` communication is easy to implement without any extra dependencies.

## Simple example

The code snippet below illustrates how you would normally use the `alertify` package. For a more elaborate example please read below about the `slackertify` example or see the full source code in the project examples directory.

```Go
        // create bot
        bot, err := alertify.NewBot(botConfig)
        if err != nil {
                log.Printf("Error creating bot: %v", err)
                os.Exit(1)
        }

        // Create Slack monitor
        slack, err := monitor.NewSlackMonitor(slackConfig)
        if err != nil {
                log.Printf("Error creating slack monitor: %s", err)
                os.Exit(1)
        }

        // register slack monitor
        if err := bot.RegisterMonitor(slack); err != nil {
                log.Printf("Error registering %s: %s", slack, err)
                os.Exit(1)
        }

        log.Fatal(bot.ListenAndAlert())
```

# slackertify

The project provides a slightly more elaborate example about how to use `alertify` to monitor message in a predefined [Slack](https://slack.com/) channel for a particular message pattern. It uses Slack RTM (Real Time Message) API, so beside the Spotify API keys described earlier in this README, you will need to register the example on the Slack workspace portal to retrieve Slack API keys.

## Spotify API keys

Provided you have registered `slackertify` via the earlier described Spotify API developer portal, you need to export your Spotify ID and secret keys via the following environment variables:

```
export SPOTIFY_ID="xxx"
export SPOTIFY_SECRET="xxx"
```

## Slack API keys

Similarly, once you've registered `slackertify` on Slack, export your Slack API keys via the following environment variable:

```
export SLACK_API_KEY="xxx"
```

## Get started

`go get` the project and ensure its dependencies are vendored:

```
$ go get -u github.com/milosgajdos/alertify
$ cd $GOPATH/src/github.com/milosgajdos/alertify && make dep
```

Build the `slackertify` binary:

```
$ make slackertify
mkdir -p ./_build
go build -o "./_build/slackertify" "examples/slackertify/slackertify.go"
```

You can also build all examples as follows:

```
$ make all
mkdir -p ./_build
for example in slackertify; do \
		go build -o "./_build/$example" "examples/$example/$example.go"; \
	done
```

Display the help:

```
$ ./_build/slackertify -help
Usage of ./_build/slackertify:
  -device-id string
    	Spotify device ID as recognised by Spotify API
  -device-name string
    	Spotify device name as recognised by Spotify API
  -redirect-uri string
    	Spotify API redirect URI (default "http://localhost:8080/callback")
  -slack-channel string
    	Slack channel that receives alerts (default "devops-production")
  -slack-msg string
    	A regexp we are matching the slack messages on (default "alert")
  -slack-user string
    	Slack username whose message we alert on (default "production")
  -song-uri string
    	Spotify song URI (default "spotify:track:2xYlyywNgefLCRDG8hlxZq")
```

## Running slackertify

You can run the `slackertify` like this:

```
$ ./_build/slackertify -slack-channel "test-bot" -slack-msg "alert" -slack-user "gyre"
```

On the start you will be prompted to visit Spotify authentication URL where you'll grant the access to the earlier described Spotify API scopes. Once you have successfully authentication you are ready to start alerting \o/:

```
[ slackertify ] Registering HTTP route -> Method: POST, Path: /alert/play
[ slackertify ] Registering HTTP route -> Method: POST, Path: /alert/silence
[ slackertify ] Starting bot message listener
[ slackertify ] Starting Slack Monitor
[ slackertify ] Starting HTTP API service
```

## Spotify devices

`slackertify` allows you to specify a specific Spotify device ID to play a song configured by passing in Spotify Song URI via command line parameters. If you leave these empty, `slackertify` will scan the local network for all available Spotify devices and play a default song on the first [active device](https://beta.developer.spotify.com/documentation/web-api/guides/using-connect-web-api/#viewing-active-device-list). If no active device is found, `slackertify` won't start and will fail straight away with non-zero exit status.

## API service

As discussed earlier, `alertify.Bot` implements a simple HTTP API which allows you to trigger the playback of a song or pause it. Here is a simple example what this looks like in practice:

Trigger the alert playback:

```
$ curl -X POST localhost:8080/alert/play
```

The song should start playing on either the explicitly Spotify device or on the firs available device:

```
[ slackertify ] POST	/alert/play
[ slackertify ] Received message: alert
[ slackertify ] Attempting to play: "Take Me Home, Country Roads" on Device ID: f6e2bbe128cd0b9b5137ee18dd5afdc34b6a2598 Name: ceres
```

You can also silence the alert song via the API:

```
$ curl -X POST localhost:8080/alert/silence
```

The song should now be paused:

```
[ slackertify ] POST	/alert/silence
[ slackertify ] Received message: silence
[ slackertify ] Attempting to pause alert playback on Device ID: f6e2bbe128cd0b9b5137ee18dd5afdc34b6a2598 Name: ceres
```

## Slack messages

`slackertify` listens to all Slack messages in a Slack channel specified via `-slack-channel` command line switch. `slackertify` will play a song once it detects a message sent by a user specified via `-slack-user` command line switch  which has a pattern specified via `-slack-msg` command line switch:


```
[ slackertify ] Slack alert message match detected!
[ slackertify ] Received message: alert
```

## Shutting down

`slackertify` implements basic signal handler and stops all goroutines safely:

```
^C[ slackertify ] Got signal: interrupt: Shutting down
[ slackertify ] Stopping message listener
[ slackertify ] HTTP API service shutting down
[ slackertify ] HTTP API service stopped
[ slackertify ] Message listener shutting down
[ slackertify ] Message listener stopped
[ slackertify ] Shutting down Slack Monitor
[ slackertify ] Slack Monitor stopped
[ slackertify ] Slack Monitor stopped
[ slackertify ] Bot message listener stopped
```

# Contributing

**IF YOU HAVE TIME, THEN YES PLEASE!!!**
