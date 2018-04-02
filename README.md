# alertify

`alertify` provides a simple API service that can trigger playing Spotify song on one of the active authenticated Spotify devices

**THIS IS JUST A POC WRITEN AT 2AM IN THE MORNING FOR LOLZ**

# Get started

Export your Spotify ID and secret keys via environment variables:

```
export SPOTIFY_ID="xxx"
export SPOTIFY_SECRET="xxx"
```

Get the project, ensure local dependencies are met and run it:

```
$ go get github.com/milosgajdos83/alertify
$ cd $GOPATH/src/github.com/milosgajdos83/alertify && make dep
$ go run main.go
```

In another terminal window:

```
curl localhost:8080/alert
```
