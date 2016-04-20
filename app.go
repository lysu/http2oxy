// +build go1.6

package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

const SoTimeout = 30 * time.Second

var (
	certFile, keyFile *string
	httpReverseProxy  *httputil.ReverseProxy
)

func init() {

	log.SetOutput(os.Stdout)

	certFile = flag.String("cert", "", "https cert file")
	keyFile = flag.String("key", "", "https key file")
	flag.Parse()

	if *certFile == "" || *keyFile == "" {
		log.Fatalln("Https Cert and Key files are required.")
	}

	httpReverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
			req.URL.Path = req.URL.Path
			if req.URL.RawQuery != "" {
				req.URL.RawPath = req.URL.RawQuery
			}
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   SoTimeout,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func main() {
	var s http.Server
	s.Addr = "0.0.0.0:443"
	s.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == "CONNECT" {

			if h1Hijack, ok := rw.(http.Hijacker); ok {
				clientConn, _, err := h1Hijack.Hijack()
				if err != nil {
					log.Fatalln("Error getting client conn:", err)
					return
				}
				handleHTTPS1x(clientConn, req.Host)
				return
			}

			handleHTTP2(rw, req)
			return
		}

		log.Println("Handle http", req.Host)
		httpReverseProxy.ServeHTTP(rw, req)
		return
	})

	go func() {
		log.Fatal(s.ListenAndServeTLS(*certFile, *keyFile))
	}()

	select {}
}

func handleHTTPS1x(clientConn net.Conn, host string) {
	remoteConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln("H1 Connect to", host, "failured.")
	}
	clientConn.Write([]byte("HTTP/1.0 200 Connection established\r\n\r\n"))
	go func() {
		_, err := io.Copy(timeoutWriter{remoteConn, SoTimeout}, timeoutReader{clientConn, SoTimeout})
		if err != nil {
			log.Fatalln("H1 Handle Read", host, "Failure with", err.Error())
		}
	}()

	_, err = io.Copy(timeoutWriter{clientConn, SoTimeout}, timeoutReader{remoteConn, SoTimeout})
	if err != nil {
		log.Fatalln("H1 Handle Write", host, "Failure with", err.Error())
	}
}

func handleHTTP2(rw http.ResponseWriter, req *http.Request) {
	remoteConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		log.Fatalln("H2 Connect to", req.Host, "failured.")
	}
	log.Println("H2 Connect to", req.Host)

	rw.WriteHeader(200)
	f, _ := rw.(http.Flusher)
	f.Flush()

	go func() {
		_, err := io.Copy(timeoutWriter{remoteConn, SoTimeout}, timeoutReader{req.Body, SoTimeout})
		if err != nil {
			log.Fatalln("H2 Handle Read", req.Host, "Failure with", err.Error())
		}
	}()

	_, err = io.Copy(timeoutWriter{flushWriter{rw}, SoTimeout}, timeoutReader{remoteConn, SoTimeout})
	if err != nil {
		log.Fatalln("H2 Handle Write", req.Host, "Failure with", err.Error())
	}
}
