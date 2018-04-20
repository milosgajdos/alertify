# alertify

[![GoDoc](https://godoc.org/github.com/milosgajdos83/alertify?status.svg)](https://godoc.org/github.com/milosgajdos83/alertify)
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Travis CI](https://travis-ci.org/milosgajdos83/alertify.svg?branch=master)](https://travis-ci.org/milosgajdos83/alertify)

**THIS IS JUST A POC WRITEN FOR LOLZ**

`alertify` is a simple service which listens to Slack messages in a provided Slack channel and plays a song on Spotify when a requested message pattern is matched

# Prerequisities

`alertify` uses both Spotify and Slack API therefore there is a couple of prerequisities that need to be satisfied before you can run it.

## Spotify API access

`alertify` uses [Spotify Connect Web API](https://beta.developer.spotify.com/documentation/web-api/guides/using-connect-web-api/) to play a preconfigured song on a Spotify device. Spotify Connect API uses OAuth to authenticate your application and providing it with required API access scopes.

Before you run `alertify` you need to register it as an app on [Spotify developer portal](https://beta.developer.spotify.com/dashboard/applications) first. Retrieve the `Client ID` and `Client Secret` API keys and store them safely. In the app settings you need to specify **Redirect URIs** where `alertify` redirects you after the Spotify authentication failure or success.

## Slack API access

Retrieve Slack API keys via Slack dashboard. `alertify` uses the old bot API keys which provide full access to all channels: this might change in the future into latest API which provides OAuth authentication with finegrained API access permissions.

# Get started

Export your Spotify ID and secret keys via environment variables:

```
export SPOTIFY_ID="xxx"
export SPOTIFY_SECRET="xxx"
```

Export your Slack API key via environment variable:

```
export SLACK_API_KEY="xxx"
```

`go get` the project and ensure its dependencies are met:

```
$ go get github.com/milosgajdos83/alertify
$ cd $GOPATH/src/github.com/milosgajdos83/alertify && make dep
```

Build the binary:

```
$ make build
```

Run the service:

```
$ ./_build/alertify -help
Usage of ./_build/alertify:
  -alert-bot string
    	Slack bot which sends alerts (default "production")
  -alert-channel string
    	Slack channel that receives alerts (default "devops-production")
  -alert-msg string
    	Slack message regexp we are matching the alert messages on (default "alert")
  -device-id string
    	Spotify device ID as recognised by Spotify API
  -device-name string
    	Spotify device name as recognised by Spotify API
  -redirect-uri string
    	OAuth app redirect URI set in Spotify API console (default "http://localhost:8080/callback")
  -song-uri string
    	Spotify song URI (default "spotify:track:2xYlyywNgefLCRDG8hlxZq")
```

# Running alertify

`alertify` will prompt you to authenticate with Spotify API on start requesting API access permissions defined by the following scopes:
* [user-read-currently-playing](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-read-currently-playing)
* [user-modify-playback-state](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-modify-playback-state)
* [user-read-playback-state](https://beta.developer.spotify.com/documentation/general/guides/scopes/#user-read-playback-state)

Once you've authenticated and granted `alertify` required permissions you are all set to be notified by a song played on a specified Spotify device.

## Spotify devices

`alertify` allows you to specify a device ID of a Spotify [player] device to play the alert song on via command line parameters. If you leave these empty, `alertify` will scan the local network for all available Spotify devices and play the song on the first [active device](https://beta.developer.spotify.com/documentation/web-api/guides/using-connect-web-api/#viewing-active-device-list). If none is found, `alertify` won't start and will fail with non-zero exit status code.

## API service

`alertify` also provides a simple API service **without any authentication** that also allows you to trigger the playback of a song or pause it. Here is a simple example what this looks like in practice:

Start `alertify`:

```
[ alertify ] Registering HTTP route -> Method: POST, Path: /alert/play
[ alertify ] Registering HTTP route -> Method: POST, Path: /alert/silence
[ alertify ] Starting Slack API message watcher
[ alertify ] Starting bot command listener
[ alertify ] Starting HTTP API service
```

Trigger the alert playback:

```
$ curl -X POST localhost:8080/alert/play
```

The song should start playing on either the explicitly Spotify device or on the firs available device:

```
[ alertify ] Starting HTTP API service
[ alertify ] POST	/alert/play
[ alertify ] Received command: alert
[ alertify ] Attempting to play: Take Me Home, Country Roads on device: xxxxxxxxx - ceres
```

Silence the alert song:

```
$ curl -X POST localhost:8080/alert/silence
```

The song should now be paused:

```
[ alertify ] POST	/alert/silence
[ alertify ] Received command: silence
[ alertify ] Attempting to silence alert  playback on device: xxxxxxxxx - ceres
```

## Slack messages

By default `alertify` listens to all Slack messages in channel called `devops-production` -- you can change the channel name via `-alert-channel` cli option. `alertify` will play song on Spotify once it detects a message sent by `production` (configurable via `-alert-bot`)  which contains `alert` in the message (configurable via `-alert-msg`).

Once `alertify` detects a message that matches regexp passed in to `-alert-msg` cli parameter it will start playing the song:

```
[ alertify ] Slack alert detected
[ alertify ] Received command: alert
[ alertify ] Attempting to play: Take Me Home, Country Roads on device: xxxxxxxxx - ceres
```

# Contributing

**IF YOU HAVE TIME, THEN YES PLEASE!!!**
