package main // import "github.com/newsdev/promise"

import (
	"flag"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/newsdev/promise/director"
	log "github.com/newsdev/promise/vendor/src/github.com/Sirupsen/logrus"
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
	go d.Watch()

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
			addr, err := d.Pick(req.Host, req.URL.Path)
			if err != nil {
				log.WithFields(log.Fields{"host": req.Host, "path": req.URL.Path}).Error(err)
				return
			}

			// Set the missing portions of the URL.
			req.URL.Scheme = "http"
			req.URL.Host = addr.String()
		},
	}

	// Every other request should hit the reverse proxy.
	http.Handle("/", reverseProxy)

	// Start the server, exiting on any error.
	log.Fatal(http.ListenAndServe(addr, nil))
}
