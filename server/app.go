package main

import (
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
	"fmt"
)

func main() {

	var srv http.Server
	srv.Addr = "localhost:4430"
	srv.ConnState = idleTimeoutHook()

	mux := http.NewServeMux()
	mux.HandleFunc("/info", reqInfoHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	})

	http2.ConfigureServer(&srv, &http2.Server{})

	go func() {
		log.Fatal(srv.ListenAndServeTLS("/Users/didi/key/server.crt", "/Users/didi/key/server.key"))
	}()

	select {}


}

func reqInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "RequestURI: %q\n", r.RequestURI)
	fmt.Fprintf(w, "URL: %#v\n", r.URL)
	fmt.Fprintf(w, "Body.ContentLength: %d (-1 means unknown)\n", r.ContentLength)
	fmt.Fprintf(w, "Close: %v (relevant for HTTP/1 only)\n", r.Close)
	fmt.Fprintf(w, "TLS: %#v\n", r.TLS)
	fmt.Fprintf(w, "\nHeaders:\n")
	r.Header.Write(w)
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
