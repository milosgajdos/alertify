package alertify

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const (
	// APIVERSION specifies API version
	APIVERSION = "v1"
	// Timeout is bot response timeout
	Timeout = 3 * time.Second
)

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
	respChan := make(chan interface{})
	// ticker timeout
	ticker := time.NewTicker(Timeout)
	defer ticker.Stop()

	go func() {
		c.msgChan <- &Msg{"alert", nil, respChan}
	}()

	// wait for Timeout seconds
	select {
	case <-ticker.C:
		log.Printf("Alert trigger timed out")
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	case err := <-respChan:
		if err != nil {
			log.Printf("Failed to trigger alert: %s", err)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func alertSilence(c *Context, w http.ResponseWriter, r *http.Request) {
	respChan := make(chan interface{})
	// ticker timeout
	ticker := time.NewTicker(Timeout)
	defer ticker.Stop()

	go func() {
		c.msgChan <- &Msg{"silence", nil, respChan}
	}()

	// wait for Timeout seconds
	select {
	case <-ticker.C:
		log.Printf("Alert silence timed out")
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	case err := <-respChan:
		if err != nil {
			log.Printf("Failed to silence alert: %s", err)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}
