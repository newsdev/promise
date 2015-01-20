package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
)

var (
	address, target, domains, service, proto string
	domainValidator                          *regexp.Regexp
)

func init() {
	flag.StringVar(&address, "a", ":80", "address to listen on")
	flag.StringVar(&target, "t", os.Getenv("TARGET"), "target")
	flag.StringVar(&domains, "d", os.Getenv("DOMAINS"), "domains")
	flag.StringVar(&service, "-service", "", "SRV service name")
	flag.StringVar(&proto, "-proto", "", "SRV protocol name")
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

	// Build a custom ReverseProxy object.
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {

			// Map the requested host to a target host.
			// TODO: Reject the request if there are no matches.
			host := req.Host
			for _, domain := range domainsParsed {
				if base := strings.TrimSuffix(host, domain); base != host {
					host := fmt.Sprintf("%s%s", base, target)
					break
				}
			}

			// It seems that for the most part, the service and protocol
			// portions of the SRV host value are not in use.
			_, addrs, err := net.LookupSRV(service, proto, host)
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
