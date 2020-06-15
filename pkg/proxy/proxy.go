package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

type Proxy struct {
	Client  *http.Client
	Forward func(*http.Request) (*url.URL, error)
}

func (proxy *Proxy) Init() error {
	if proxy.Client == nil {
		proxy.Client = &http.Client{
			Transport: http.DefaultTransport,
		}
	}

	transport, ok := proxy.Client.Transport.(*http.Transport)
	if !ok {
		return errors.New("client transport must have type net/http.Transport")
	}

	transport.Proxy = proxy.Forward
	return nil
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		proxy.HttpConnect(w, r)
	} else {
		proxy.Http(w, r)
	}
}

func (proxy *Proxy) Http(w http.ResponseWriter, r *http.Request) {
	resp, err := proxy.Client.Transport.RoundTrip(r)
	if err != nil {
		log.Warnf("HTTP round trip: %s", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		w.Header()[name] = values
	}

	w.WriteHeader(resp.StatusCode)

	// Ignore benign errors for closed connections
	io.Copy(w, resp.Body)
}

func (proxy *Proxy) HttpConnect(w http.ResponseWriter, r *http.Request) {
	var hop *url.URL
	var err error

	if proxy.Forward == nil {
		hop = nil
	} else {
		// Check if there's another proxy to forward the CONNECT request to
		hop, err = proxy.Forward(r)
		if err != nil {
			log.Warnf("Forward rule error: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Error("Unable to get hijacker")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var target string
	if hop == nil {
		// Connecting directly to the host in the request
		target = r.Host
	} else {
		// Connecting to another proxy
		target = hop.Host
	}

	serverConn, err := proxy.dial(context.Background(), "tcp", target)
	if err != nil {
		log.Warnf("Connecting to server: %s", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer serverConn.Close()

	if hop == nil {
		// Need to respond with 200 to the CONNECT if not using a proxy
		w.WriteHeader(http.StatusOK)
	} else {
		// Write the original CONNECT to the proxy and allow it to reply
		r.Write(serverConn)
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Errorf("Hijacking client connection: %s", err)
		return
	}
	defer clientConn.Close()

	// Ignore benign errors for closed connections
	go io.Copy(clientConn, serverConn)
	io.Copy(serverConn, clientConn)
}

// Use the Transport's Dialer otherwise fall back to net.Dialer
func (proxy *Proxy) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	transport, ok := proxy.Client.Transport.(*http.Transport)
	if ok {
		if transport.DialContext != nil {
			return transport.DialContext(ctx, network, addr)
		} else if transport.Dial != nil {
			return transport.Dial(network, addr)
		}
	}

	dialer := net.Dialer{}
	return dialer.DialContext(ctx, network, addr)
}
