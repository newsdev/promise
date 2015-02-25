package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/nytinteractive/srv-proxy/director"
)

var (
	addr, etcdPeers   string
	enableCompression bool
)

func init() {
	flag.StringVar(&addr, "a", ":80", "address to listen on")
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.BoolVar(&enableCompression, "z", false, "enable transport compresssion")
}

func main() {
	flag.Parse()

	// Create a new director.
	d := director.NewEtcdDirector(strings.Split(etcdPeers, ","))

	// Build a custom ReverseProxy object.
	reverseProxy := &httputil.ReverseProxy{
		Transport: &http.Transport{
			DisableCompression: !enableCompression,
		},
		Director: func(req *http.Request) {

			// Get an address from the director. If an error occurs, we're just
			// allowing an empty URL value in the request to pass through. The idea
			// is to trigger an error and not allow arbitrary proxying of hosts we
			// do not know about, but it's a less than ideal solution.
			addr, err := d.Pick(req.Host)
			if err != nil {
				log.Println(err)
				return
			}

			// Set the missing portions of the URL.
			req.URL.Scheme = "http"
			req.URL.Host = addr.String()
		},
	}

	// Add a status check route. In the future this should possibly be optional.
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	})

	// Every other request should hit the reverse proxy.
	http.Handle("/", reverseProxy)

	// Start the server, exiting on any error.
	log.Fatal(http.ListenAndServe(addr, nil))
}
