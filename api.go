package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// APIVERSION specifies API version
const APIVERSION = "v1"

type handler func(c *Context, w http.ResponseWriter, r *http.Request)

type routes map[string]map[string]map[string]handler

func apiRoutes() routes {
	return routes{
		APIVERSION: {
			"POST": {
				"/alert/play":    alertPlay,
				"/alert/silence": alertSilence,
			},
		},
	}
}

func newRouter(c *Context) *mux.Router {
	r := mux.NewRouter()
	routeMap := apiRoutes()

	for method, routes := range routeMap[APIVERSION] {
		for route, rhandler := range routes {
			log.Printf("Registering HTTP route -> Method: %s, Path: %s", method, route)
			// local scope for http.Handler
			rh := rhandler
			wrapHandleFunc := func(w http.ResponseWriter, r *http.Request) {
				log.Printf("%s\t%s", r.Method, r.RequestURI)
				rh(c, w, r)
			}
			r.Path("/" + APIVERSION + route).Methods(method).HandlerFunc(wrapHandleFunc)
			r.Path(route).Methods(method).HandlerFunc(wrapHandleFunc)
		}
	}

	return r
}

func alertPlay(c *Context, w http.ResponseWriter, r *http.Request) {
	// TODO: should we try to re-play if it's already playing?
	// Maybe yes, if a different song is requested
	resp := make(chan interface{})
	go func() {
		c.cmdChan <- &Msg{"alert", resp}
	}()
	if err := <-resp; err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func alertSilence(c *Context, w http.ResponseWriter, r *http.Request) {
	resp := make(chan interface{})
	go func() {
		c.cmdChan <- &Msg{"silence", resp}
	}()
	if err := <-resp; err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}
