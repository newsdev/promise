package main // import "github.com/newsdev/promise"

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/newsdev/promise/director"
)

var (
	listenerPairs, etcdPeers string
	enableCompression        bool
	prefixValidator          *regexp.Regexp
)

func init() {
	prefixValidator = regexp.MustCompile(`^\w+(\/\w+)*$`)
	flag.StringVar(&listenerPairs, "listeners", "promise:80", "etcd prefix/address pairs to setup listners")
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.BoolVar(&enableCompression, "z", false, "enable transport compresssion")
}

func main() {
	flag.Parse()

	log.Info(listenerPairs)

	errChan := make(chan error)

	machines := strings.Split(etcdPeers, ",")

	for _, listenerPair := range strings.Split(listenerPairs, ",") {
		log.Info(listenerPair)

		listenerPairComponents := strings.Split(listenerPair, ":")

		prefix := listenerPairComponents[0]

		if !prefixValidator.Match([]byte(prefix)) {
			log.Fatalf("prefix is invalid: %s", prefix)
		}

		port := listenerPairComponents[1]

		// Create a new director.
		d := director.NewEtcdDirector(prefix, machines)
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

		mux := http.NewServeMux()
		mux.Handle("/", reverseProxy)

		addr := fmt.Sprintf(":%s", port)

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		go func() {
			log.Infof("Listening at %s", addr)
			errChan <- server.ListenAndServe()

		}()

		// log.Fatal(http.ListenAndServe(addr, nil))

	}

	log.Fatal(<-errChan)

}
