package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"

	"github.com/nytinteractive/srv-proxy/director"
	log "github.com/nytinteractive/srv-proxy/vendor/src/github.com/Sirupsen/logrus"
)

var (
	addr, etcdPeers, target, domains string
	enableCompression                bool
	domainValidator                  *regexp.Regexp
)

func init() {
	flag.StringVar(&addr, "a", ":80", "address to listen on")
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.StringVar(&target, "t", os.Getenv("TARGET"), "target")
	flag.StringVar(&domains, "d", os.Getenv("DOMAINS"), "domains")
	flag.BoolVar(&enableCompression, "z", false, "enable transport compresssion")
	domainValidator = regexp.MustCompile(`^[\w-]+(\.[\w-]+)*$`)
}

func main() {
	flag.Parse()

	// Validate the target.
	if !domainValidator.MatchString(target) {
		log.Fatalf("invalid target domain: \"%s\"\n", target)
	}
	target := fmt.Sprintf(".%s", target)

	// Parse and validate the domains.
	domainsParsed := strings.Split(domains, `,`)
	for i, domain := range domainsParsed {
		if !domainValidator.MatchString(domain) {
			log.Fatalf("invalid domain: \"%s\"\n", domain)
		}
		domainsParsed[i] = fmt.Sprintf(".%s", domain)
	}

	// Create a new director.
	d := director.NewEtcdDirector(strings.Split(etcdPeers, ","))
	go func() {
		for {
			if err := d.Watch(); err != nil {
				log.Println(err)
			}
		}
	}()

	// Build a custom ReverseProxy object.
	reverseProxy := &httputil.ReverseProxy{
		Transport: &http.Transport{
			DisableCompression: !enableCompression,
		},
		Director: func(req *http.Request) {

			// Map the requested host to a target host.
			host := req.Host
			for _, domain := range domainsParsed {
				if base := strings.TrimSuffix(host, domain); base != host {
					host = fmt.Sprintf("%s%s", base, target)
					break
				}
			}

			// Get an address from the director. If an error occurs, we're just
			// allowing an empty URL value in the request to pass through. The idea
			// is to trigger an error and not allow arbitrary proxying of hosts we
			// do not know about, but it's a less than ideal solution.
			addr, err := d.Pick(host)
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
