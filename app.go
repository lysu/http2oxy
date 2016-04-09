package main

import (
	"net/http/httputil"
	"io"
	"io/ioutil"
	"fmt"
	"net/http"
	"log"
)

func main() {


	httpProxy := NewReverseProxy()

	handler := func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == "Connect" {
			fmt.Println("Handle https")
		} else {
			fmt.Println("Handle as http")
			httpProxy.ServeHTTP(rw, req)
		}
	}


	go func() {
		log.Fatal(http.ListenAndServeTLS("0.0.0.0:443", "cc.crt", "cc.key", http.HandlerFunc(handler)))
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
