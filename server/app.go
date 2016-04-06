package main

import (
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
	"io"
	"io/ioutil"
)

func main() {

	var srv http.Server
	srv.Addr = "0.0.0.0:4430"
	srv.ConnState = idleTimeoutHook()

	httpProxy := NewReverseProxy()

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		isHTTPS := req.TLS != nil
		if isHTTPS {
			ServerHTTPS(rw, req)
		} else {
			httpProxy.ServeHTTP(rw, req)
		}
	})

	http2.ConfigureServer(&srv, &http2.Server{})

	go func() {
		log.Fatal(srv.ListenAndServeTLS("cc.crt", "cc.key"))
	}()

	select {}

}

func ServerHTTPS(rw http.ResponseWriter, req *http.Request) {
	pr, pw := io.Pipe()
	req, err := http.NewRequest(req.Method, req.URL.String(), ioutil.NopCloser(pr))
	if err != nil {
		panic(err)
	}
	req.URL.Scheme = "https"
	req.URL.Host = req.Host
	req.URL.Path = req.URL.Path
	if req.URL.RawQuery != "" {
		req.URL.RawPath = req.URL.RawQuery
	}
	go func() {
		_, err := io.Copy(pw, req.Body)
		if err != nil {
			panic(err)
		}
	}()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		panic(err)
	}
}

func NewReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
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
