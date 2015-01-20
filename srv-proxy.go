package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
)

var address, service, proto string

func init() {
	flag.StringVar(&address, "a", ":80", "address to listen on")
	flag.StringVar(&service, "-service", "", "SRV service name")
	flag.StringVar(&proto, "-proto", "", "SRV protocol name")
}

func main() {
	flag.Parse()

	// Build a custom ReverseProxy object.
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {

			// It seems that for the most part, the service and protocol
			// portions of the SRV host value are not in uts.
			_, addrs, err := net.LookupSRV(service, proto, req.Host)
			if err != nil {

				// Setting the request URL to nil will result in no proxy
				// request being made and an error sent back to the requester.
				// TODO: Figure out a better way to do this.
				req.URL = nil
				return
			}

			// Chose one of the addresses at random.
			addr := addrs[rand.Intn(len(addrs))]

			// Set the missing portions of the URL.
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("%s:%d", addr.Target, addr.Port)
		},
	}

	log.Fatal(http.ListenAndServe(address, reverseProxy))
}
