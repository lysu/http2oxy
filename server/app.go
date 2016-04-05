package main

import (
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

func main() {

	var srv http.Server
	srv.Addr = "localhost:4430"
	srv.ConnState = idleTimeoutHook()

	proxy := NewReverseProxy()

	http.Handle("/", proxy)

	http2.ConfigureServer(&srv, &http2.Server{})

	go func() {
		log.Fatal(srv.ListenAndServeTLS("/Users/didi/key/server.crt", "/Users/didi/key/server.key"))
	}()

	select {}

}

func NewReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = req.URL.Scheme
		req.URL.Host = req.URL.Host
		req.URL.Path = req.URL.Path
		if req.URL.RawQuery != "" {
			req.URL.RawPath = req.URL.RawQuery
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

const idleTimeout = 5 * time.Minute
const activeTimeout = 10 * time.Minute

func idleTimeoutHook() func(net.Conn, http.ConnState) {
	var mu sync.Mutex
	m := map[net.Conn]*time.Timer{}
	return func(c net.Conn, cs http.ConnState) {
		mu.Lock()
		defer mu.Unlock()
		if t, ok := m[c]; ok {
			delete(m, c)
			t.Stop()
		}
		var d time.Duration
		switch cs {
		case http.StateNew, http.StateIdle:
			d = idleTimeout
		case http.StateActive:
			d = activeTimeout
		default:
			return
		}
		m[c] = time.AfterFunc(d, func() {
			log.Printf("closing idle conn %v after %v", c.RemoteAddr(), d)
			go c.Close()
		})
	}
}
