package main

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func main() {

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	http2.ConfigureTransport(transport)

	httpClient := &http.Client{
		Transport:       transport,
	}

	resp, err := httpClient.Get("https://localhost:4430/info")
	if err != nil {
		panic(err)
	}

	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(data))

}
