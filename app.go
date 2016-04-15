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
	req.Header.Set("Connection", "close")
	remoteConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Connected %s\n", req.Host)

	go Pipe(remoteConn, &clientConn{
		Reader: req.Body,
		Writer: rw,
	})
	rw.WriteHeader(http.StatusOK)
	fmt.Println(remoteConn)

	//a, err := ioutil.ReadAll(req.Body)
	//if err != nil {
	//	panic(err)
	//}
	//
	//log.Printf("----===%s\n", string(a))
}

type x interface {
	Write(b []byte) (n int, err error)
	Read(b []byte) (n int, err error)
	SetDeadline(t time.Time) error
}

type clientConn struct {
	io.Reader
	io.Writer
}

func (c *clientConn) SetDeadline(t time.Time) error {
	return nil
}

func Pipe(conn1 x, conn2 x) {
	chan1 := makeConnChan(conn1, 1)
	chan2 := makeConnChan(conn2, 2)

	for {
		select {
		case b1 := <-chan1:
			if b1 == nil {
				return
			} else {
				conn2.Write(b1)
			}
		case b2 := <-chan2:
			if b2 == nil {
				return
			} else {
				conn1.Write(b2)
			}
		}
	}
}

func makeConnChan(conn x, num int) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(b)

			if n > 0 {
				res := make([]byte, n)
				copy(res, b[:n])
				c <- res
			}

			if err != nil {
				c <- nil
				if err != io.EOF {
					log.Printf("%d Piping error %s", num, err)
				} else {
					log.Printf("reading finished")
				}
			}
		}
	}()

	return c
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
