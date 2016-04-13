package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

func main() {

	httpProxy := NewReverseProxy()

	handler := func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("----------", req.Method)
		if req.Method == "CONNECT" {
			fmt.Println("Handle https")
			ServerHTTPS(rw, req)
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
	remoteConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Connected %s\n", req.Host)
	go func() {
		remoteConn.SetDeadline(time.Now().Add(200 * time.Millisecond))
		_, err := io.Copy(remoteConn, req.Body)
		if err != nil {
			panic(err)
		}
	}()
	go func() {

		remoteConn.SetDeadline(time.Now().Add(200 * time.Millisecond))
		_, err = io.Copy(rw, remoteConn)
		if err != nil {
			panic(err)
		}
	}()
	rw.Write([]byte("HTTP/1.0 200 Connection established\r\n\r\n"))
}

func NewReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = req.Host
		req.URL.Path = req.URL.Path
		if req.URL.RawQuery != "" {
			req.URL.RawPath = req.URL.RawQuery
		}
	}
	return &httputil.ReverseProxy{Director: director}
}
