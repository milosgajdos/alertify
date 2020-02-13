package alertify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
)

// API provides a simple HTTP API
type API struct {
	h *http.Server
	l net.Listener
}

// Context provides API service context
type Context struct {
	msgChan chan *Msg
}

// newListener creates a new TCP listener
func newListener(proto, addr string, tlsConfig *tls.Config) (net.Listener, error) {
	var (
		l   net.Listener
		err error
	)

	switch proto {
	case "unix", "unixpacket":
		// Unix sockets must be unlink()ed before being reused again
		if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		l, err = net.Listen(proto, addr)
	case "tcp":
		l, err = net.Listen(proto, addr)
	default:
		return nil, fmt.Errorf("unsupported protocol: %q", proto)
	}

	if tlsConfig != nil {
		//tlsConfig.NextProtos = []string{"http/1.1"}
		l = tls.NewListener(l, tlsConfig)
	}

	return l, err
}

// NewAPI creates and configures API server
//
// NewAPI creates and initializes API server with provided configuration
// It returns error if either configuration is invalid or if API server could not be created
func NewAPI(ctx *Context, address string, tlsConfig *tls.Config) (*API, error) {
	api := newRouter(ctx)
	server := &http.Server{
		Handler: api,
	}

	protoAddrParts := strings.SplitN(address, "://", 2)
	if len(protoAddrParts) == 1 {
		protoAddrParts = []string{"tcp", protoAddrParts[0]}
	}

	listener, err := newListener(protoAddrParts[0], protoAddrParts[1], tlsConfig)
	if err != nil {
		return nil, err
	}
	server.Addr = protoAddrParts[1]

	return &API{
		h: server,
		l: listener,
	}, nil
}

// ListenAndServe starts API server and listens for HTTP requests
//
// ListenAndServe blocks until http server returns error
// Due to its blocking behaviour this function should be run in its own goroutine
func (a *API) ListenAndServe() error {
	return a.h.Serve(a.l)
}
